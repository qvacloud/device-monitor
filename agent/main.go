package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"time"

	"devicemonitor/collector"

	_ "embed"
)

//go:embed web/index.html
var selfPage []byte

func main() {
	addr := flag.String("addr", ":55555", "listen address")
	token := flag.String("token", "", "shared secret required in X-Auth-Token header (empty = no auth)")
	interval := flag.Duration("interval", time.Second, "cpu sampling interval")
	flag.Parse()

	c := collector.New()
	c.Start(*interval)
	defer c.Stop()

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if *token != "" && r.Header.Get("X-Auth-Token") != *token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(c.Collect())
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(selfPage)
	})

	log.Printf("device-monitor agent listening on %s (api: /metrics)", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}
