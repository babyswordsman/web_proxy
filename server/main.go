package main

import (
	"flag"
	"net/http"

	"github.com/yeqown/log"

	"github.com/valyala/fasthttp"

	proxy "github.com/yeqown/fasthttp-reverse-proxy"
)

var (
	proxyServer       *proxy.ReverseProxy
	proxyWsServer     *proxy.WSReverseProxy
	proxyGoogleSearch *proxy.ReverseProxy

	ws_path string
	logger  *log.Logger

	proxy_addr   string
	backend_addr string

	google_key string
	google_cx  string
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
		if string(ctx.Path()) == "/customsearch/v1" {
			args := ctx.URI().QueryArgs()
			args.Add("cx", google_cx)
			args.Add("key", google_key)
			url := "https://www.googleapis.com/customsearch/v1?" + args.String()
			code, buf, err := fasthttp.Get(nil, url)
			logger.Info("url:", url)
			if err != nil {
				ctx.Response.SetStatusCode(http.StatusInternalServerError)
				len, err := ctx.Write([]byte(err.Error()))
				if err != nil {
					logger.Error(err.Error())
				} else {
					logger.Info("response len:", len)
				}
			} else {
				ctx.Response.SetStatusCode(code)
				len, err := ctx.Write(buf)
				if err != nil {
					logger.Error(err.Error())
				} else {
					logger.Info("response len:", len)
				}
			}

		} else {
			proxyServer.ServeHTTP(ctx)
		}

	}

}

func main() {
	flag.StringVar(&ws_path, "ws_path", "/chat/stream", "websocket path")
	flag.StringVar(&proxy_addr, "proxy_addr", ":8082", "http host")
	flag.StringVar(&backend_addr, "backend_addr", "localhost:80", "")
	flag.StringVar(&google_key, "google_key", "***", "key")
	flag.StringVar(&google_cx, "google_cx", "***", "cx")
	flag.Parse()
	logger, _ = log.NewLogger()
	logger.Infof("ws_path:%s,proxy_addr:%s,backend_addr:%s", ws_path, proxy_addr, backend_addr)
	initproxy(backend_addr, ws_path)
	if err := fasthttp.ListenAndServe(proxy_addr, ProxyHandler); err != nil {
		logger.Fatal(err)
	}

}
