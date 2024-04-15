package main

import (
	"flag"

	"github.com/yeqown/log"

	"github.com/valyala/fasthttp"

	proxy "github.com/yeqown/fasthttp-reverse-proxy"
)

var (
	proxyServer   *proxy.ReverseProxy
	proxyWsServer *proxy.WSReverseProxy

	ws_path string
	logger  *log.Logger

	proxy_addr   string
	backend_addr string
)

func initproxy(addr string, ws_path_ string) {
	ws_path = ws_path_
	proxyServer = proxy.NewReverseProxy(addr)
	proxyWsServer = proxy.NewWSReverseProxy(addr, ws_path)
}

func ProxyHandler(ctx *fasthttp.RequestCtx) {
	//todo:暂时只支持单个websocket路径
	logger.Infof("uri:%s", ctx.URI())

	if ctx.Request.Header.ConnectionUpgrade() {
		if ws_path != string(ctx.Path()) {
			logger.Errorf("path {%s} is not %s", string(ctx.Path()), ws_path)
			return
		}

		proxyWsServer.ServeHTTP(ctx)
	} else {
		proxyServer.ServeHTTP(ctx)
	}

}

func main() {
	flag.StringVar(&ws_path, "ws_path", "/chat/stream", "websocket path")
	flag.StringVar(&proxy_addr, "proxy_addr", ":8082", "http host")
	flag.StringVar(&backend_addr, "backend_addr", "localhost:80", "")
	flag.Parse()
	logger, _ = log.NewLogger()
	logger.Infof("ws_path:%s,proxy_addr:%s,backend_addr:%s", ws_path, proxy_addr, backend_addr)
	initproxy(backend_addr, ws_path)
	if err := fasthttp.ListenAndServe(proxy_addr, ProxyHandler); err != nil {
		logger.Fatal(err)
	}
}
