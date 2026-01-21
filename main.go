package main

import (
	"log"
	"net/http"
)

const (
	port       = ":6733"
	devicesCSV = "devices.csv"
)

func main() {
	log.Println("[STARTUP] SafelyYou Device Monitoring API")

	// Load devices from CSV
	store := NewStore()
	var configErr error

	if err := store.LoadDevicesFromCSV(devicesCSV); err != nil {
		log.Printf("[ERROR] Failed to load devices from %s: %v", devicesCSV, err)
		configErr = err
	} else {
		log.Printf("[CONFIG] Loaded %d devices from %s", store.DeviceCount(), devicesCSV)
	}

	// Create server (will return 500s if configErr is set)
	server := NewServer(store, configErr)

	// Start HTTP server
	log.Printf("[STARTUP] Server listening on %s", port)
	log.Printf("[STARTUP] Base URL: http://127.0.0.1%s/api/v1", port)

	if err := http.ListenAndServe(port, server.Router()); err != nil {
		log.Fatalf("[ERROR] Server failed: %v", err)
	}
}
