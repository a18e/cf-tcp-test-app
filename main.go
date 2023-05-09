package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	//appIsHealthy := true
	rand.Seed(time.Now().Unix())
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "8080"
	}

	//index, _ := os.LookupEnv("CF_INSTANCE_INDEX")

	var startStopDelay time.Duration
	startStopDelayString, ok := os.LookupEnv("START_STOP_DELAY")
	if !ok {
		startStopDelay = 0
	}
	startStopDelay, err := time.ParseDuration(startStopDelayString)
	if err != nil {
		log.Fatalf("invalid START_STOP_DELAY %s", startStopDelayString)
	}
	log.Printf("waiting %s for startup\n", startStopDelay)
	time.Sleep(startStopDelay)
	log.Printf("done waiting")

	log.Printf("Listening on :%s", port)

	//go setupAppStop(startStopDelay)

	setupTcpServer()

}

func setupAppStop(startStopDelay time.Duration) {
	// Create a channel to receive signals
	signals := make(chan os.Signal, 1)

	// Register the channel to receive SIGINT signals
	signal.Notify(signals, syscall.SIGTERM)

	// Wait for a SIGINT signal
	log.Println("Waiting for SIGTERM signal...")
	<-signals
	log.Println("SIGTERM signal received. Shutting down...")

	log.Printf("sleeping for %s before exiting\n", startStopDelay)
	time.Sleep(startStopDelay)

}

func setupTcpServer() {
	// Listen for incoming connections on port 8080
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println(err)
		return
	}

	// Infinite loop to handle incoming connections
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		// Handle incoming connection in a separate goroutine
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	// Close the connection when this function exits
	defer conn.Close()

	// Create a buffer to read incoming data
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Check if the incoming data is an HTTP request
	if strings.HasPrefix(string(buf), "GET /") {
		// Respond with a simple HTTP response
		response := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nHello, World!"
		conn.Write([]byte(response))
	} else {
		// Respond with an error message
		response := "HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain\r\n\r\nInvalid request"
		conn.Write([]byte(response))
	}
}
