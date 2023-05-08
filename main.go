package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	appIsHealthy := true
	rand.Seed(time.Now().Unix())
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "8080"
	}

	index, _ := os.LookupEnv("CF_INSTANCE_INDEX")

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

	mux := setupMux(index, &appIsHealthy)

	log.Printf("Listening on :%s", port)
	go http.ListenAndServe(fmt.Sprintf(":%s", port), mux)

	setupAppStop(startStopDelay)

}

func setupMux(index string, appIsHealthy *bool) *http.ServeMux {

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {
		if *appIsHealthy {
			writer.WriteHeader(200)
			writer.Write([]byte("all is well"))
		} else {
			writer.WriteHeader(404)
			writer.Write([]byte("feeling a bit under the weather"))
		}

	})
	mux.HandleFunc("/fail", func(w http.ResponseWriter, req *http.Request) {

		q := req.URL.Query()
		p, _ := strconv.Atoi(q.Get("p"))
		r := rand.Intn(100)
		if p < 0 {
			p = 0
		}
		if p > 100 {
			p = 100
		}
		if r < p {
			log.Fatalf("%d < %d - CRASH!\n", r, p)
		}

		i := q.Get("i")
		if i == index {
			log.Fatalf("instance %s = i %s - CRASH!\n", index, i)
		}

		w.WriteHeader(200)
		w.Write([]byte(fmt.Sprintf("(instance: %s) %d >= %d - OK\n", index, r, p)))
		log.Printf("(instance: %s) %d >= %d - OK\n", index, r, p)
	})
	mux.HandleFunc("/drop", func(w http.ResponseWriter, req *http.Request) {

		q := req.URL.Query()
		p, _ := strconv.Atoi(q.Get("p"))
		r := rand.Intn(100)
		if p < 0 {
			p = 0
		}
		if p > 100 {
			p = 100
		}
		if r < p {
			req.Context().Done()
		}

		i := q.Get("i")
		if i == index {
			req.Context().Done()
			log.Fatalf("instance %s = i %s - CRASH!\n", index, i)
		}

		w.WriteHeader(200)
		w.Write([]byte(fmt.Sprintf("(instance: %s) %d >= %d - OK\n", index, r, p)))
		log.Printf("(instance: %s) %d >= %d - OK\n", index, r, p)
	})
	mux.HandleFunc("/drop_if_unhealthy", func(w http.ResponseWriter, req *http.Request) {
		if *appIsHealthy {
			w.WriteHeader(200)
			w.Write([]byte("app is healthy - returning 200"))
		} else {
			req.Context().Done()
		}
	})
	mux.HandleFunc("/toggle_health", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
		*appIsHealthy = !*appIsHealthy
		if *appIsHealthy {
			w.Write([]byte("app will report healthy"))
		} else {
			w.Write([]byte("app will report UN-healthy"))
		}
	})
	return mux
}

func setupAppStop(startStopDelay time.Duration) {
	// Create a channel to receive signals
	signals := make(chan os.Signal, 1)

	// Register the channel to receive SIGINT signals
	signal.Notify(signals, syscall.SIGTERM)

	// Wait for a SIGINT signal
	fmt.Println("Waiting for SIGTERM signal...")
	<-signals
	fmt.Println("SIGTERM signal received. Shutting down...")

	time.Sleep(startStopDelay)
}
