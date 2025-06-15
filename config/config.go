package config

import (
	"os"
	"strconv"
	"time"
)

// UDP Multicast Address to be used in discovery service
var ANNOUNCE_ADDR string

// Duration interval for announcing in discovery service
var ANNOUNCE_INTERVAL time.Duration = time.Duration(3) * time.Second

// The port where to listen
var SERVICE_PORT uint16 = 40480

func Setup() {
	// Get ANNOUNCE_ADDR env variable
	multicastAddr, exists := os.LookupEnv("ANNOUNCE_ADDR")
	if exists && multicastAddr != "" {
		ANNOUNCE_ADDR = multicastAddr
	} else {
		ANNOUNCE_ADDR = "224.0.0.250:40400"
	}

	// Get SERVICE_PORT env variable
	servicePort, exists := os.LookupEnv("SERVICE_PORT")
	if exists && servicePort != "" {
		port, err := strconv.Atoi(servicePort)
		if err == nil {
			SERVICE_PORT = uint16(port)
		}
	}

	// Get ANNOUNCE_INTERVAL env variable
	announceInterval, exists := os.LookupEnv("ANNOUNCE_INTERVAL")
	if exists && announceInterval != "" {
		seconds, err := strconv.Atoi(announceInterval)
		if err == nil {
			ANNOUNCE_INTERVAL = time.Duration(seconds) * time.Second
		}
	}
}
