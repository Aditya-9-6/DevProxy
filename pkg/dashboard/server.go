package dashboard

import (
	"net/http"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/analysis"
	"github.com/Aditya-9-6/DevProxy/pkg/certs"
	"github.com/Aditya-9-6/DevProxy/pkg/contract"
	"github.com/Aditya-9-6/DevProxy/pkg/mock"
	"github.com/Aditya-9-6/DevProxy/pkg/replay"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"github.com/Aditya-9-6/DevProxy/pkg/storage"
)

type Server struct {
	store             *storage.Store
	hub               *Hub
	ca                *certs.CertificateAuthority
	addr              string
	httpSrv           *http.Server
	mockEngine        *mock.Engine
	contractValidator *contract.Validator
	replayer          *replay.Replayer
	ringBuf           *ringbuffer.RingBuffer
}

func NewServer(addr string, store *storage.Store, hub *Hub, ca *certs.CertificateAuthority) *Server {
	s := &Server{store: store, hub: hub, ca: ca, addr: addr, mockEngine: mock.NewEngine(), contractValidator: contract.NewValidator(), replayer: replay.NewReplayer(15*time.Second, true)}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	s.httpSrv = &http.Server{Addr: s.addr, Handler: mux}
	return s
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {}
