package main

import (
	"log"
	"net/http"

	"github.com/valyala/fasthttp"

	proxy "github.com/yeqown/fasthttp-reverse-proxy"
)

var (
	proxyServer       *proxy.ReverseProxy
	proxyWsServer     *proxy.WSReverseProxy
	proxyGoogleSearch *proxy.ReverseProxy
)

func initproxy(addr string, ws_path_ string) {
	ws_path = ws_path_
	proxyServer = proxy.NewReverseProxy(addr)
	proxyWsServer = proxy.NewWSReverseProxy(addr, ws_path)

}

func ProxyHandler(ctx *fasthttp.RequestCtx) {
	//todo:暂时只支持单个websocket路径
	log.Printf("uri:%s", ctx.URI())

	if ctx.Request.Header.ConnectionUpgrade() {
		if ws_path != string(ctx.Path()) {
			log.Printf("path {%s} is not %s", string(ctx.Path()), ws_path)
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
			log.Printf("url:", url)
			if err != nil {
				ctx.Response.SetStatusCode(http.StatusInternalServerError)
				len, err := ctx.Write([]byte(err.Error()))
				if err != nil {
					log.Printf(err.Error())
				} else {
					log.Printf("response len:", len)
				}
			} else {
				ctx.Response.SetStatusCode(code)
				len, err := ctx.Write(buf)
				if err != nil {
					log.Printf(err.Error())
				} else {
					log.Printf("response len:", len)
				}
			}

		} else {
			proxyServer.ServeHTTP(ctx)
		}

	}

}

func StartFastHttpProxy() {

	log.Printf("ws_path:%s,proxy_addr:%s,backend_addr:%s", ws_path, proxy_addr, backend_addr)
	initproxy(backend_addr, ws_path)
	if err := fasthttp.ListenAndServe(proxy_addr, ProxyHandler); err != nil {
		log.Fatalln(err)
	}

}
