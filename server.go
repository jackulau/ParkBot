package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	_ "embed"
)

//go:embed web/index.html
var indexHTML []byte

// Server manages the HTTP interface, SSE log streaming, and bot lifecycle.
type Server struct {
	cfgPath string
	cfg     *Config

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc

	clientsMu sync.RWMutex
	clients   map[chan string]struct{}
}

func NewServer(cfgPath string, cfg *Config) *Server {
	return &Server{
		cfgPath: cfgPath,
		cfg:     cfg,
		clients: make(map[chan string]struct{}),
	}
}

// LogWriter returns an io.Writer that broadcasts each write to all SSE clients.
func (s *Server) LogWriter() io.Writer {
	return &sseLogWriter{s: s}
}

type sseLogWriter struct{ s *Server }

func (w *sseLogWriter) Write(p []byte) (int, error) {
	w.s.broadcast(string(p))
	return len(p), nil
}

func (s *Server) broadcast(msg string) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()
	for ch := range s.clients {
		select {
		case ch <- msg:
		default: // skip slow clients
		}
	}
}

func (s *Server) addClient(ch chan string) {
	s.clientsMu.Lock()
	s.clients[ch] = struct{}{}
	s.clientsMu.Unlock()
}

func (s *Server) removeClient(ch chan string) {
	s.clientsMu.Lock()
	delete(s.clients, ch)
	s.clientsMu.Unlock()
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /config", s.handleGetConfig)
	mux.HandleFunc("POST /config", s.handleSaveConfig)
	mux.HandleFunc("POST /run", s.handleRun)
	mux.HandleFunc("POST /stop", s.handleStop)
	mux.HandleFunc("GET /events", s.handleEvents)
	mux.HandleFunc("GET /status", s.handleStatus)
	mux.HandleFunc("POST /unlock", s.handleUnlock)
	log.Printf("GUI available at http://%s", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	cfg := s.cfg
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

func (s *Server) handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	var newCfg Config
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	// Fill default Chrome profile if empty
	if newCfg.ChromeProfile == "" {
		newCfg.ChromeProfile = defaultChromeProfile()
	}
	if err := newCfg.Save(s.cfgPath); err != nil {
		http.Error(w, "saving config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.mu.Lock()
	s.cfg = &newCfg
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		http.Error(w, "bot is already running", http.StatusConflict)
		return
	}

	// Validate and normalize config before running
	cfg := *s.cfg
	cfg.PermitKeyword = strings.ToUpper(strings.TrimSpace(cfg.PermitKeyword))
	cfg.VehicleKeyword = strings.ToUpper(strings.TrimSpace(cfg.VehicleKeyword))
	cfg.AddressKeyword = strings.ToUpper(strings.TrimSpace(cfg.AddressKeyword))
	if cfg.ChromeProfile == "" {
		cfg.ChromeProfile = defaultChromeProfile()
	}
	if err := cfg.validate(); err != nil {
		http.Error(w, "Config error: "+err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running = true

	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("Bot panic: %v", rec)
			}
			s.mu.Lock()
			s.running = false
			s.cancel = nil
			s.mu.Unlock()
			s.broadcast("__done__")
		}()
		log.Println("=== Bot started ===")
		if err := runBot(ctx, &cfg); err != nil {
			log.Printf("Bot error: %v", err)
		} else {
			log.Println("=== Bot finished successfully ===")
		}
	}()

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running || s.cancel == nil {
		http.Error(w, "bot is not running", http.StatusBadRequest)
		return
	}
	s.cancel()
	log.Println("Stop requested — cancelling bot...")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	ch := make(chan string, 512)
	s.addClient(ch)
	defer s.removeClient(ch)

	// Send current status on connect
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()
	locked := false
	if _, err := os.Stat(lockFile); err == nil {
		locked = true
	}
	initData, _ := json.Marshal(map[string]any{
		"type":    "connected",
		"locked":  locked,
		"running": running,
	})
	fmt.Fprintf(w, "data: %s\n\n", initData)
	flusher.Flush()

	for {
		select {
		case msg := <-ch:
			if msg == "__done__" {
				doneData, _ := json.Marshal(map[string]any{"type": "done"})
				fmt.Fprintf(w, "data: %s\n\n", doneData)
				flusher.Flush()
				// Keep connection alive for next run
				continue
			}
			b, _ := json.Marshal(map[string]any{"type": "log", "msg": msg})
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()

		case <-r.Context().Done():
			return

		case <-time.After(20 * time.Second):
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()
	locked := false
	if _, err := os.Stat(lockFile); err == nil {
		locked = true
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"running": running,
		"locked":  locked,
	})
}

func (s *Server) handleUnlock(w http.ResponseWriter, r *http.Request) {
	if err := os.Remove(lockFile); err != nil && !os.IsNotExist(err) {
		http.Error(w, "removing lock: "+err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println("Lock file removed — bot can run again.")
	w.WriteHeader(http.StatusNoContent)
}
