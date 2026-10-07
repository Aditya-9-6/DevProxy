package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestHeadlessCI(t *testing.T) {
	// Rebuild the current dir to test execution behavior without executing the actual main in process
	cmd := exec.Command("go", "build", "-o", "devproxy-ci-test", ".")
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build binary for CI test: %v", err)
	}
	defer os.Remove("devproxy-ci-test")
	defer os.Remove("devproxy-report.json") // cleanup

	// Run with -ci flag, we expect SUCCESS because no traffic happens immediately
	runCmd := exec.Command("./devproxy-ci-test", "-ci", "-port", "8099", "-web-port", "8100")
	var stdout, stderr bytes.Buffer
	runCmd.Stdout = &stdout
	runCmd.Stderr = &stderr

	err = runCmd.Start()
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}

    // Give it a moment to boot before interrupting
    time.Sleep(500 * time.Millisecond)

	// Send an interrupt signal to gracefully trigger the shutdown CI logic
	runCmd.Process.Signal(os.Interrupt)

	err = runCmd.Wait()

	// Exit status 0 is expected because we didn't send any traffic, so no findings
	// However, sometimes wait returns an error simply because we killed it via signal, so we check the output
	// if it successfully parsed our graceful shutdown hook!

	output := stderr.String() + stdout.String() // log output usually goes to stderr in Go
	if !strings.Contains(output, "Headless execution enabled") {
		t.Errorf("Expected output to contain 'Headless execution enabled', but got: %s", output)
	}
	if !strings.Contains(output, "SUCCESS: No HIGH/CRITICAL vulnerabilities detected") {
		t.Errorf("Expected output to contain 'SUCCESS' message, but got: %s", output)
	}

	// Ensure report was generated
	if _, err := os.Stat("devproxy-report.json"); os.IsNotExist(err) {
		t.Errorf("Expected devproxy-report.json to be created, but it was not found")
	}
}
