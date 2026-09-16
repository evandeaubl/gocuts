package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	addr := flag.String("addr", envOr("GOCUTS_ADDR", ":8080"), "listen address")
	configPath := flag.String("config", envOr("GOCUTS_CONFIG", ""), "path to TOML shortcuts file (or set GOCUTS_CONFIG)")
	baseURL := flag.String("base-url", envOr("GOCUTS_BASE_URL", ""), "external base URL used in OpenSearch templates (default: derived from each request)")
	suggest := flag.Bool("suggest", envBool("GOCUTS_SUGGEST", true), "serve the /suggest endpoint and advertise it in opensearch.xml (or set GOCUTS_SUGGEST)")
	list := flag.Bool("list", envBool("GOCUTS_LIST", true), "list shortcuts on the home page and on unknown-shortcut errors (or set GOCUTS_LIST)")
	key := os.Getenv("GOCUTS_KEY")
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "gocuts: no config file given; use -config <path> or set GOCUTS_CONFIG")
		os.Exit(2)
	}
	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("gocuts: %v", err)
	}
	srv := &server{cfg: cfg, baseURL: strings.TrimRight(*baseURL, "/"), suggest: *suggest, list: *list, key: key}

	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           srv.handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	log.Printf("gocuts: serving %d shortcuts on %s (suggest=%t list=%t auth=%t)", len(cfg.names), *addr, *suggest, *list, key != "")
	log.Fatal(httpSrv.ListenAndServe())
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		log.Printf("gocuts: invalid %s value %q, using default %v", key, v, fallback)
		return fallback
	}
	return b
}
