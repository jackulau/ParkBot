package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"
)

const serverAddr = "localhost:8420"

func main() {
	log.SetFlags(log.Ltime | log.Lmsgprefix)
	log.SetPrefix("[ParkBot] ")

	cfgPath := "config.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	// Load config (missing file is fine — GUI lets users fill it in)
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		log.Printf("Warning: could not load config (%v) — starting with empty config", err)
		cfg = &Config{OneTime: true}
	}

	srv := NewServer(cfgPath, cfg)

	// Tee log output to both stderr and all connected SSE clients
	log.SetOutput(io.MultiWriter(os.Stderr, srv.LogWriter()))

	// Open the browser after a short delay to let the server start
	go func() {
		time.Sleep(600 * time.Millisecond)
		openBrowser("http://" + serverAddr)
	}()

	log.Fatal(srv.Start(serverAddr))
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default: // darwin
		cmd = exec.Command("open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("Could not open browser: %v — navigate to http://%s", err, serverAddr)
	}
}
