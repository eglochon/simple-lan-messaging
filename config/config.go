package config

import (
	"os"
	"strconv"
)

// UDP Multicast Address to be used in discovery service
var MULTICAST_ADDR string

// The port where to listen
var SERVICE_PORT uint16 = 40480

func Setup() {
	// Get MULTICAST_ADDR env variable
	multicastAddr, exists := os.LookupEnv("MULTICAST_ADDR")
	if exists && multicastAddr != "" {
		MULTICAST_ADDR = multicastAddr
	} else {
		MULTICAST_ADDR = "224.0.0.250:40400"
	}

	// Get SERVICE_PORT env variable
	servicePort, exists := os.LookupEnv("SERVICE_PORT")
	if exists && servicePort != "" {
		port, err := strconv.Atoi(servicePort)
		if err == nil {
			SERVICE_PORT = uint16(port)
		}
	}
}
