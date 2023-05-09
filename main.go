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
var startStopDelay = 0 * time.Second

// immediately closes the listener after app has become unhealthy
var instantCloseListener = false

func main() {
	var err error
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "8080"
	}

	initialHealth, ok := os.LookupEnv("INITIAL_HEALTH")
	if ok {
		appIsHealthy = strings.ToLower(initialHealth) == "true"
	}

	closeListenerString, ok := os.LookupEnv("INSTANT_CLOSE_LISTENER")
	if ok {
		instantCloseListener = strings.ToLower(closeListenerString) == "true"
	}

	startStopDelayString, ok := os.LookupEnv("START_STOP_DELAY")
	if !ok {
		startStopDelay = 0
	}
	startStopDelay, err = time.ParseDuration(startStopDelayString)
	if err != nil {
		log.Fatalf("invalid START_STOP_DELAY %s", startStopDelayString)
	}
	log.Printf("waiting %s for startup\n", startStopDelay)
	time.Sleep(startStopDelay)
	log.Printf("done waiting")

	log.Printf("Listening on :%s | app-health: %t", port, appIsHealthy)
	setupTcpServer()

}

func setupTcpServer() {
	// Listen for incoming connections on port 8080
	ln, err := net.Listen("tcp", ":8080")
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

func setupDelayedShutdown(ln net.Listener, group *sync.WaitGroup) {
	defer group.Done()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case sig := <-sigChan:
			log.Printf("Received signal %v...", sig)

			if instantCloseListener {
				log.Printf("closing listener")
				err := ln.Close()
				if err != nil {
					log.Print(err)
				}
			} else {
				log.Printf("keeping listener open")
			}

			log.Printf("sleeping for %v", startStopDelay)
			time.Sleep(startStopDelay)
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
		fmt.Println(err)
		return
	}

	path, method, err := getHttpPathAndMethod(buf)
	if err != nil {
		fmt.Println(err)
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
