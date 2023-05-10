package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
)

var appIsHealthy = true
var startDelay = 0 * time.Second
var drainDelay = 0 * time.Second
var stopDelay = 0 * time.Second

func main() {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "8080"
	}

	initialHealth, ok := os.LookupEnv("INITIAL_HEALTH")
	if ok {
		appIsHealthy = strings.ToLower(initialHealth) == "true"
	}

	durationFromEnv("START_DELAY", &startDelay)
	durationFromEnv("DRAIN_DELAY", &drainDelay)
	durationFromEnv("STOP_DELAY", &stopDelay)

	log.Printf("waiting %s for startup\n", startDelay)
	time.Sleep(startDelay)
	log.Printf("done waiting")

	log.Printf("Listening on :%s | app-health: %t", port, appIsHealthy)
	setupTcpServer(port)
}

func setupTcpServer(port string) {
	// Listen for incoming connections on port 8080
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Println(err)
		return
	}

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(1)

	go setupDelayedShutdown(ln, waitGroup)

	// Infinite loop to handle incoming connections
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			break
		}

		// Handle incoming connection in a separate goroutine
		go handleConnection(conn)
	}

	//wait for delayed shutdown
	log.Println("connection loop done, waiting for waitgroup")
	waitGroup.Wait()
}

func durationFromEnv(envVar string, duration *time.Duration) {
	var err error
	durationString, ok := os.LookupEnv(envVar)
	if ok {
		*duration, err = time.ParseDuration(durationString)
		if err != nil {
			log.Fatalf("invalid %s: %s", envVar, durationString)
		}
	} // else keep default
}

func setupDelayedShutdown(ln net.Listener, group *sync.WaitGroup) {
	defer group.Done()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case sig := <-sigChan:
			log.Printf("Received signal %v...", sig)

			log.Printf("draining for %v", drainDelay)
			time.Sleep(drainDelay)
			log.Printf("done drainig. closing listener.")

			err := ln.Close()
			if err != nil {
				log.Print(err)
			}

			log.Printf("sleeping for %v before stop.", stopDelay)
			time.Sleep(stopDelay)
			log.Printf("done sleeping. exiting.")
			syscall.Exit(0) // we need to syscall exit here in case the incoming connection loop doesn't receive any more connections and doesn't break
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func getHttpPathAndMethod(requestBody []byte) (string, string, error) {
	//requestLine:= strings.Split(string(requestBody), "\n")
	re := regexp.MustCompile(`^(\S+)\s+(\S+)\s+HTTP/\d\.\d`)
	matches := re.FindStringSubmatch(string(requestBody))
	if len(matches) < 3 {
		return "", "", fmt.Errorf("could not parse HTTP request %s", string(requestBody))
	}

	method := matches[1]
	path := matches[2]
	return path, method, nil
}

func handleConnection(conn net.Conn) {
	// Close the connection when this function exits
	defer conn.Close()

	// Create a buffer to read incoming data
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		log.Println(err)
		return
	}

	path, method, err := getHttpPathAndMethod(buf)
	if err != nil {
		log.Println(err)
	}

	switch path {
	case "/health":
		log.Printf("received health check request. Healthy: %t", appIsHealthy)
		if appIsHealthy {
			response := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nApp is healthy!"
			conn.Write([]byte(response))
		} else {
			conn.Close()
		}

	case "/togglehealth":
		appIsHealthy = !appIsHealthy
		log.Printf("received toggle-health request. New health: %t", appIsHealthy)
		response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\napp healthy: %t", appIsHealthy)
		conn.Write([]byte(response))

	case "/drop":
		log.Printf("received drop request. New health: %t", appIsHealthy)
		conn.Close()

	case "/always_up":
		log.Printf("received always_up request: %t", appIsHealthy)
		response := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nHello World!\r\n"
		conn.Write([]byte(response))

	default:
		log.Printf("received request: method: %s, path: %s | app-health: %t", method, path, appIsHealthy)
		if appIsHealthy {
			response := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nHello World!\r\n"
			conn.Write([]byte(response))
		} else {
			conn.Close()
		}
	}
}
