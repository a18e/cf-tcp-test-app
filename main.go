package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	rand.Seed(time.Now().Unix())
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "8080"
	}

	index, _ := os.LookupEnv("CF_INSTANCE_INDEX")

	mux := http.NewServeMux()
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

	log.Printf("Listening on :%s", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), mux)
}
