package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Aditya-9-6/DevProxy/pkg/analysis"
	"github.com/Aditya-9-6/DevProxy/pkg/certs"
	"github.com/Aditya-9-6/DevProxy/pkg/contract"
	"github.com/Aditya-9-6/DevProxy/pkg/dashboard"
	"github.com/Aditya-9-6/DevProxy/pkg/mock"
	"github.com/Aditya-9-6/DevProxy/pkg/proxy"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"github.com/Aditya-9-6/DevProxy/pkg/storage"
	"github.com/Aditya-9-6/DevProxy/pkg/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// resultBridge forwards analysis results to storage and WebSocket hub.
type resultBridge struct {
	store *storage.Store
	hub   *dashboard.Hub
	tel   *telemetry.Exporter
	metricsCounter metric.Int64Counter
}

func (b *resultBridge) HandleResult(event *ringbuffer.TrafficEvent, findings []*analysis.Finding) {
	// Persist to in-memory SQLite
	if err := b.store.SaveTransaction(event, findings); err != nil {
		log.Printf("[Storage Error] %v", err)
	}

	// Broadcast via WebSocket to dashboard
	b.hub.BroadcastEvent(event, findings)

	// Print high-severity alerts to terminal
	for _, f := range findings {
		if f.Severity == analysis.SeverityCritical || f.Severity == analysis.SeverityHigh {
			fmt.Printf("\033[1;31m[ALERT - %s]\033[0m %s: %s (%s %s)\n", f.Severity, f.Title, f.RuleName, f.Method, f.URL)
		}
	}

	// Emit OpenTelemetry Security finding metrics and SIEM span events
	if len(findings) > 0 && b.tel != nil {
		ctx := context.Background()
		if event.TraceID != "" && event.SpanID != "" {
			// Synthesize a span context to attach events directly to the distributed trace
			traceID, _ := trace.TraceIDFromHex(event.TraceID)
			spanID, _ := trace.SpanIDFromHex(event.SpanID)
			spanContext := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    traceID,
				SpanID:     spanID,
				TraceFlags: trace.FlagsSampled,
				Remote:     true,
			})
			ctx = trace.ContextWithSpanContext(ctx, spanContext)
		}

		_, span := b.tel.Tracer().Start(ctx, "devproxy.security.analysis")
		defer span.End()

		for _, f := range findings {
			span.AddEvent("security.finding", trace.WithAttributes(
				attribute.String("rule", f.RuleName),
				attribute.String("severity", f.Severity),
				attribute.String("title", f.Title),
				attribute.String("url", f.URL),
			))
			if b.metricsCounter != nil {
				b.metricsCounter.Add(ctx, 1, metric.WithAttributes(
					attribute.String("severity", f.Severity),
				))
			}
		}
	}
}

func main() {
	proxyPort := flag.Int("port", 8080, "Port for the HTTP/HTTPS proxy engine")
	webPort := flag.Int("web-port", 8081, "Port for the local dashboard and REST/WebSocket API")
	bufferSize := flag.Int("buffer-size", 16384, "Capacity of the lock-free ring buffer")
	workers := flag.Int("workers", 4, "Number of concurrent analysis worker goroutines")
	caCertPath := flag.String("ca-cert", "", "Path to custom Root CA certificate (PEM)")
	caKeyPath := flag.String("ca-key", "", "Path to custom Root CA private key (PEM)")
	rulesPath := flag.String("rules", "", "Path to custom YAML rules file (defaults to ./devproxy.yaml if present)")
	openapiPath := flag.String("openapi", "", "Path to OpenAPI 3.0 / Swagger specification file (JSON/YAML)")
	showEBPF := flag.Bool("ebpf", false, "Display eBPF and container transparent redirection guide")
	showVersion := flag.Bool("version", false, "Print DevProxy version and build info")
	showInstallCA := flag.Bool("install-ca", false, "Display instructions to install and trust the Root CA")
	insecureUpstream := flag.Bool("insecure-upstream", false, "Allow upstream HTTPS connections to skip TLS verification (for local self-signed dev microservices)")
	upstreamProxy := flag.String("upstream-proxy", "", "Route DevProxy's own egress through an upstream proxy: http://, https:// or socks5://host:port (falls back to HTTPS_PROXY/ALL_PROXY when empty)")
	otlpEndpoint := flag.String("otlp-endpoint", "", "OpenTelemetry OTLP gRPC endpoint for exporting distributed traces and SIEM metrics")
	ciMode := flag.Bool("ci", false, "Enable Headless CI/CD mode. Exits with code 1 if CRITICAL/HIGH findings are detected, and skips Web UI.")
	flag.Parse()

	if *showVersion {
		fmt.Println("DevProxy v1.0.0 - Zero-Latency Decoupled Development Security Proxy")
		fmt.Println("Repository: https://github.com/Aditya-9-6/DevProxy")
		return
	}

	if *showEBPF {
		ebpfMgr := proxy.NewEBPFManager(*proxyPort)
		fmt.Println(ebpfMgr.GenerateEBPFInstructions())
		return
	}

	if *showInstallCA {
		ca, err := certs.NewCertificateAuthority(*caCertPath, *caKeyPath)
		if err != nil {
			log.Fatalf("Failed to initialize CA: %v", err)
		}
		_ = ca
		fmt.Println(`
================================================================================
 DEVPROXY ROOT CA INSTALLATION INSTRUCTIONS
================================================================================
DevProxy uses dynamic TLS bumping to decrypt and inspect HTTPS traffic.
To prevent SSL certificate warnings in curl, browsers, and mobile emulators:

[ Windows - PowerShell (Run as Administrator) ]
  Import-Certificate -FilePath ~/.devproxy/devproxy-ca.crt -CertStoreLocation Cert:\LocalMachine\Root

[ macOS ]
  sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ~/.devproxy/devproxy-ca.crt

[ Linux - Ubuntu / Debian ]
  sudo cp ~/.devproxy/devproxy-ca.crt /usr/local/share/ca-certificates/devproxy-ca.crt
  sudo update-ca-certificates

[ Node.js & Python Development ]
  export NODE_EXTRA_CA_CERTS="$HOME/.devproxy/devproxy-ca.crt"
  export SSL_CERT_FILE="$HOME/.devproxy/devproxy-ca.crt"
  export REQUESTS_CA_BUNDLE="$HOME/.devproxy/devproxy-ca.crt"

[ Curl One-Liner ]
  curl -x http://localhost:8080 --cacert ~/.devproxy/devproxy-ca.crt https://httpbin.org/get
================================================================================`)
		return
	}

	printBanner(*proxyPort, *webPort)

	// 1. Initialize Root CA & Dynamic Leaf Minting Manager
	log.Println("[1/5] Initializing Root CA and Dynamic TLS Bumping engine...")
	ca, err := certs.NewCertificateAuthority(*caCertPath, *caKeyPath)
	if err != nil {
		log.Fatalf("Failed to initialize CA: %v", err)
	}
	certManager := certs.NewCertificateManager(ca)

	// 2. Initialize In-Memory SQLite Ephemeral Datastore
	log.Println("[2/5] Initializing Ephemeral In-Memory SQLite Datastore...")
	store, err := storage.NewStore()
	if err != nil {
		log.Fatalf("Failed to initialize SQLite store: %v", err)
	}
	defer store.Close()

	// 3. Initialize Lock-Free Decoupled Ring Buffer
	log.Printf("[3/5] Allocating Lock-Free Ring Buffer (Capacity: %d events)...", *bufferSize)
	ringBuf := ringbuffer.NewRingBuffer(*bufferSize)

	// 4. Initialize Rules Engine & Worker Pool
	log.Printf("[4/5] Initializing High-Speed Rules Engine with %d background workers...", *workers)
	engine := analysis.NewSecurityRulesEngine()

	// Load custom rules if specified or if devproxy.yaml exists locally
	customRulesFile := *rulesPath
	if customRulesFile == "" {
		if _, err := os.Stat("devproxy.yaml"); err == nil {
			customRulesFile = "devproxy.yaml"
		}
	}
	if customRulesFile != "" {
		customRules, err := analysis.LoadCustomRulesFromFile(customRulesFile)
		if err != nil {
			log.Printf("[Warning] Failed to load custom rules from %s: %v", customRulesFile, err)
		} else {
			engine.AddRules(customRules)
			log.Printf(" Loaded %d custom rules from %s", len(customRules), customRulesFile)
		}
	}

	// 5. Initialize Mock & Chaos Engine + OpenAPI Contract Validator
	mockEngine := mock.NewEngine()
	contractValidator := contract.NewValidator()
	engine.AddRule(contractValidator)

	// Auto-load OpenAPI specification if provided or found
	specPath := *openapiPath
	if specPath == "" {
		if _, err := os.Stat("openapi.yaml"); err == nil {
			specPath = "openapi.yaml"
		} else if _, err := os.Stat("swagger.json"); err == nil {
			specPath = "swagger.json"
		}
	}
	if specPath != "" {
		if data, err := os.ReadFile(specPath); err == nil {
			if err := contractValidator.LoadSpec(data); err == nil {
				log.Printf(" Loaded OpenAPI contract specification from %s", specPath)
			}
		}
	}

	var tel *telemetry.Exporter
	if *otlpEndpoint != "" {
		log.Printf("[0/5] Initializing OpenTelemetry Exporter at %s...", *otlpEndpoint)
		ctx := context.Background()
		var err error
		tel, err = telemetry.NewExporter(ctx, *otlpEndpoint)
		if err != nil {
			log.Fatalf("Failed to initialize OpenTelemetry: %v", err)
		}
		defer tel.Shutdown(ctx)
	}

	hub := dashboard.NewHub()
	if !*ciMode {
		go hub.Run()
	}

	var counter metric.Int64Counter
	if tel != nil {
		counter, _ = tel.Meter().Int64Counter("devproxy.security.findings")
	}

	bridge := &resultBridge{
		store: store,
		hub:   hub,
		tel:   tel,
		metricsCounter: counter,
	}

	workerPool := analysis.NewAnalysisWorkerPool(ringBuf, engine, bridge, *workers)
	workerPool.Start()
	defer workerPool.Stop()

	// 6. Initialize & Start Dashboard Server
	var dashServer *dashboard.Server
	if !*ciMode {
		webAddr := fmt.Sprintf(":%d", *webPort)
		dashServer = dashboard.NewServer(webAddr, store, hub, ca)
		dashServer.SetMockEngine(mockEngine)
		dashServer.SetContractValidator(contractValidator)
		dashServer.SetRingBuffer(ringBuf)
		go func() {
			if err := dashServer.Start(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Dashboard server error: %v", err)
			}
		}()
	} else {
		log.Println("[CI MODE] Headless execution enabled. Web UI Dashboard disabled.")
	}

	// 7. Initialize & Start Primary Proxy Server
	proxyAddr := fmt.Sprintf(":%d", *proxyPort)
	proxyServer := proxy.NewProxyServer(proxyAddr, certManager, ringBuf)
	proxyServer.SetMockEngine(mockEngine)
	proxyServer.SetTelemetry(tel)
	if *insecureUpstream {
		proxyServer.SetInsecureUpstreamTLS(true)
		log.Println(" Upstream TLS verification: INSECURE/SKIP (dev microservices mode)")
	}
	if *upstreamProxy != "" {
		if err := proxyServer.SetUpstreamProxy(*upstreamProxy); err != nil {
			log.Fatalf("Invalid -upstream-proxy: %v", err)
		}
		// Deliberately does not log the proxy URL: it may carry a host you
		// would rather not write to disk.
		log.Println(" Upstream proxy: enabled for all DevProxy egress")
	}
	go func() {
		if err := proxyServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Proxy server error: %v", err)
		}
	}()

	log.Printf(" DevProxy is ACTIVE and listening on %s", proxyAddr)
	if !*ciMode {
		log.Printf(" Dashboard UI is live at http://localhost:%d", *webPort)
		log.Printf(" Download Root CA certificate at http://localhost:%d/api/ca.crt", *webPort)
	}
	log.Println("--------------------------------------------------------------------------------")

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n\nShutting down DevProxy gracefully...")
	_ = proxyServer.Close()
	if dashServer != nil {
		_ = dashServer.Close()
	}
	workerPool.Stop()

	// Print final stats
	analyzed, findingsCount, avgUs := workerPool.Stats()
	_, dropped, total := ringBuf.Stats()
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf(" [DevProxy Session Summary]\n")
	fmt.Printf("   - Packets Forwarded:    %d\n", total)
	fmt.Printf("   - Traffic Analyzed:     %d requests\n", analyzed)
	fmt.Printf("   - Dropped via Saturation: %d requests\n", dropped)
	fmt.Printf("   - Vulnerabilities Found: %d\n", findingsCount)
	fmt.Printf("   - Average Analysis Time: %.2f µs/request\n", avgUs)
	fmt.Println("--------------------------------------------------------------------------------")

	if *ciMode {
		fmt.Println("[CI MODE] Generating CI Security Report...")
		findings, err := store.GetRecentFindings(10000, "")

		highOrCritical := 0
		var reportFindings []*analysis.Finding
		if err == nil {
			reportFindings = findings
			for _, f := range findings {
				if f.Severity == analysis.SeverityCritical || f.Severity == analysis.SeverityHigh {
					highOrCritical++
				}
			}
		}

		reportData, _ := json.MarshalIndent(map[string]interface{}{
			"total_findings": len(reportFindings),
			"critical_high_count": highOrCritical,
			"findings": reportFindings,
		}, "", "  ")

		if err := os.WriteFile("devproxy-report.json", reportData, 0644); err != nil {
			fmt.Printf("[CI MODE] Failed to write report: %v\n", err)
		} else {
			fmt.Println("[CI MODE] Wrote devproxy-report.json successfully.")
		}

		if highOrCritical > 0 {
			fmt.Printf("\033[1;31m[CI MODE] FAILED: %d HIGH/CRITICAL security vulnerabilities detected.\033[0m\n", highOrCritical)
			os.Exit(1)
		} else {
			fmt.Println("\033[1;32m[CI MODE] SUCCESS: No HIGH/CRITICAL vulnerabilities detected.\033[0m")
			os.Exit(0)
		}
	}
}

func printBanner(proxyPort, webPort int) {
	fmt.Println(`
   ___             ___                      
  / _ \___ _  __  / _ \_______ __ ____ __  
 / // / -_) |/ / / ___/ __/ _ \\ \ / // /  
/____/\__/|___/ /_/  /_/  \___/_\_\\_, /   
                                   /___/    
 ZERO-LATENCY PASSIVE SECURITY PROXY FOR DEVELOPERS`)
	fmt.Printf(" Proxy Port:      127.0.0.1:%d\n", proxyPort)
	fmt.Printf(" Dashboard URL:   http://127.0.0.1:%d\n", webPort)
	fmt.Println(" Architecture:   Decoupled Data-Path (Asynchronous Ring Buffer)")
	fmt.Println(" Rules Engine:   Single-Pass Zero-Allocation Pattern Matcher")
	fmt.Println(" Storage:        Ephemeral SQLite (In-Memory)")
	fmt.Println("--------------------------------------------------------------------------------")
}
