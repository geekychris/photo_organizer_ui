package main

import (
	"flag"
	"fmt"
	"log"

	"photodatabase/internal/config"
	"photodatabase/internal/database"
	"photodatabase/internal/scanner"
	"photodatabase/internal/server"
)

func main() {
	var (
		configFile = flag.String("config", "config.yaml", "Configuration file path")
		scan       = flag.String("scan", "", "Scan directory path")
		rescan     = flag.Bool("rescan", false, "Rescan all managed directories")
		serve      = flag.Bool("serve", false, "Start web server")
		port       = flag.String("port", "8080", "Server port")
	)
	flag.Parse()

	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	if *scan != "" {
		log.Printf("Scanning directory: %s", *scan)
		s := scanner.New(db, cfg.ThumbnailDir)
		if err := s.ScanDirectory(*scan); err != nil {
			log.Fatalf("Failed to scan directory: %v", err)
		}
		log.Println("Scan completed")
		return
	}

	if *rescan {
		log.Println("Rescanning all managed directories")
		s := scanner.New(db, cfg.ThumbnailDir)
		if err := s.RescanAll(); err != nil {
			log.Fatalf("Failed to rescan: %v", err)
		}
		log.Println("Rescan completed")
		return
	}

	if *serve {
		log.Printf("Starting web server on port %s", *port)
		srv := server.New(db, cfg)
		if err := srv.Start(*port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
		return
	}

	fmt.Println("Photo Database Manager")
	fmt.Println("Usage:")
	fmt.Println("  -scan <directory>    Scan a directory and add to database")
	fmt.Println("  -rescan             Rescan all managed directories")
	fmt.Println("  -serve              Start web server")
	fmt.Println("  -port <port>        Server port (default: 8080)")
	fmt.Println("  -config <file>      Configuration file (default: config.yaml)")
}