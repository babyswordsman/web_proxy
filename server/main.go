package main

import (
	"flag"
)

var (
	ws_path string

	proxy_addr   string
	backend_addr string

	google_key string
	google_cx  string
)

func GetBackend() string {
	return backend_addr
}

func GetProxyAddr() string {
	return proxy_addr
}
func main() {
	flag.StringVar(&ws_path, "ws_path", "/chat/stream", "websocket path")
	flag.StringVar(&proxy_addr, "proxy_addr", ":8082", "http host")
	flag.StringVar(&backend_addr, "backend_addr", "localhost:80", "")
	flag.StringVar(&google_key, "google_key", "***", "key")
	flag.StringVar(&google_cx, "google_cx", "***", "cx")
	flag.Parse()
	StartGinProxy()
}
