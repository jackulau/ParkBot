package main

import (
	"log"
	"os"
)

func main() {
	log.SetFlags(log.Ltime | log.Lmsgprefix)
	log.SetPrefix("[ParkBot] ")

	cfgPath := defaultConfigPath()
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := loadConfig(cfgPath)
	if err != nil {
		cfg = &Config{OneTime: true}
	}

	runGUI(cfgPath, cfg)
}
