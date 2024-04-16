package main

import (
	"bufio"
	"log"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func DefaultDealHttpGet(ctx *gin.Context) {

	path := ctx.Request.URL.Path
	log.Println("DefaultDealHttpGet:", path, ",url:", ctx.Request.URL.String())
	if path == "/customsearch/v1" {
		DealGoogleSearchApi(ctx)
	} else if path == "/chat/stream" {
		DealWebsocket(ctx)
	} else {
		DealHttpGet(ctx, "http", GetBackend())
	}

}
func DealHttpGet(ctx *gin.Context, schema string, host string) {
	// step 1: resolve proxy address, change scheme and host in requets

	oldreq := ctx.Request

	req := http.Request{}

	url := *oldreq.URL
	req.URL = &url

	req.URL.Scheme = schema
	req.URL.Host = host
	req.Method = oldreq.Method
	v, ok := oldreq.Header["Cookie"]
	req.Header = make(http.Header, 0)
	if ok {
		req.Header["Cookie"] = v
	}

	// step 2: use http.Transport to do request to real server.
	transport := http.DefaultTransport
	//不要打印，https的请求中有机密信息
	//log.Println(req.URL.String())

	resp, err := transport.RoundTrip(&req)
	if err != nil {
		log.Printf("error in roundtrip: %v", err)
		ctx.String(http.StatusInternalServerError, "error")
		return
	}

	// step 3: return real server response to upstream.
	for k, vv := range resp.Header {
		for _, v := range vv {
			ctx.Header(k, v)
		}
	}
	defer resp.Body.Close()
	bufio.NewReader(resp.Body).WriteTo(ctx.Writer)

}

func DealGoogleSearchApi(ctx *gin.Context) {
	log.Println("DealGoogleSearchApi:", ctx.Request.URL.String())
	req := ctx.Request
	queries := req.URL.Query()
	queries.Add("cx", google_cx)
	queries.Add("key", google_key)
	req.URL.RawQuery = queries.Encode()

	DealHttpGet(ctx, "https", "www.googleapis.com")

}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WSMessage struct {
	MsgType int
	Message []byte
}

func connectWSBackend(ctx *gin.Context, backend string, path string, backendListener chan WSMessage, frontListener chan WSMessage, backendStop *int32, frontStop *int32) {

	defer func() {
		if err := recover(); err != nil {
			log.Println("panic:", err)
		}
		//告诉前端，后端线程已经没有消息了
		close(backendListener)
		atomic.StoreInt32(backendStop, 1)
		log.Println("close backendListener")
	}()

	u := url.URL{Scheme: "ws", Host: backend, Path: path}

	v, ok := ctx.Request.Header["Cookie"]
	header := make(http.Header)
	if ok {
		header["Cookie"] = v
	}

	log.Printf("connecting to %s", u.String())

	c, resp, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		log.Println("dial:", err)

		return
	}
	defer c.Close()
	defer resp.Body.Close()

	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Println("panic:", err)
			}
			atomic.StoreInt32(backendStop, 1)
		}()
		tick := time.NewTicker(time.Millisecond * 100)
		defer tick.Stop()

		for atomic.LoadInt32(backendStop) == 0 {
			msgType, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read backend:", err)
				atomic.StoreInt32(backendStop, 1)
				break
			}
			log.Printf("recv:%s", message)
			if atomic.LoadInt32(frontStop) >= 1 {
				//前端故障，后端可以提前结束了
				log.Print("frontStop:", time.Now().Format("2016-01-02 15:04:05"))
				atomic.StoreInt32(backendStop, 1)
				break
			}
		retryWrite:
			for atomic.LoadInt32(backendStop) == 0 {
				//避免阻塞时tick触发丢失消息，需要循环重试
				select {
				case backendListener <- WSMessage{MsgType: msgType, Message: message}:
					//已经写入就等待下一条消息
					break retryWrite
				case curTime := <-tick.C:
					if atomic.LoadInt32(frontStop) >= 1 {
						//前端关闭就退出
						log.Print("frontStop:", curTime.Format("2016-01-02 15:04:05"))
						atomic.StoreInt32(backendStop, 1)
						break retryWrite
					}
				}
			}

		}

	}()

	go func() {
		//前端不收消息了，需要同时保证frontStop==1
		for wsMessage := range frontListener {
			err := c.WriteMessage(wsMessage.MsgType, wsMessage.Message)
			if err != nil {
				atomic.StoreInt32(backendStop, 1)
				log.Println("write backend:", err)
				return
			}
		}
	}()

	tick := time.NewTicker(time.Millisecond * 100)
	defer tick.Stop()
	//等待后端的收消息线程退出
	for curTime := range tick.C {
		if atomic.LoadInt32(frontStop) >= 1 {
			//前端关闭就退出
			log.Print("frontStop:", curTime.Format("2016-01-02 15:04:05"))
			atomic.StoreInt32(backendStop, 1)
			break
		}
		if atomic.LoadInt32(backendStop) >= 1 {
			log.Print("backendStop:", curTime.Format("2016-01-02 15:04:05"))
			break
		}
	}

	bufio.NewReader(resp.Body).WriteTo(ctx.Writer)

}

func DealWebsocket(ctx *gin.Context) {
	log.Println("DealWebsocket:", ctx.Request.URL.String())
	w, r := ctx.Writer, ctx.Request
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer c.Close()

	backendListener := make(chan WSMessage)
	frontListener := make(chan WSMessage)
	frontStop := int32(0)
	backendStop := int32(0)
	go connectWSBackend(ctx, GetBackend(), r.URL.Path, backendListener, frontListener, &backendStop, &frontStop)

	go func() {
		//后端退出会关闭backendListener，没有消息了才能关闭前端
		for wsMessage := range backendListener {
			err := c.WriteMessage(wsMessage.MsgType, wsMessage.Message)
			if err != nil {
				//前端异常了
				atomic.StoreInt32(&frontStop, 1)
				log.Println("write backend:", err)
				return
			}
		}
		//没有更多消息给前端了
		atomic.StoreInt32(&frontStop, 1)
	}()
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Println("panic:", err)
			}
			//告诉后端，前端线程已经没有消息了
			close(frontListener)
			log.Println("close frontLister")
		}()
		tick := time.NewTicker(time.Millisecond * 100)
		defer tick.Stop()
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read front:", err)
				break
			}
			log.Printf("recv:%s", message)
			if atomic.LoadInt32(&backendStop) >= 1 {
				//后端退出，前端不用再读，可以退出了，但是可能还有消息可以发送个前端
				log.Print("backendStop:", time.Now().Format("2016-01-02 15:04:05"))
				break
			}

		retryWrite:
			for atomic.LoadInt32(&backendStop) == 0 {
				//避免阻塞时tick触发丢失消息，需要循环重试
				//后端已经停止，不需要后端处理了
				select {
				case frontListener <- WSMessage{MsgType: mt, Message: message}:
					//已经写入就等待下一条消息
					break retryWrite
				case curTime := <-tick.C:
					if atomic.LoadInt32(&backendStop) >= 1 {
						//前端关闭就退出
						log.Print("backendStop:", curTime.Format("2016-01-02 15:04:05"))

						break retryWrite
					}
				}
			}

		}
	}()

	tick := time.NewTicker(time.Millisecond * 100)
	defer tick.Stop()
	//等待后端的收消息线程退出
	for curTime := range tick.C {
		if atomic.LoadInt32(&frontStop) >= 1 {
			//前端关闭就退出
			log.Print("frontStop:", curTime.Format("2016-01-02 15:04:05"))
			break
		}
	}
	log.Println("deal websocket end")
}
func StartGinProxy() {
	r := gin.Default()
	//r.GET("/customsearch/v1", DealGoogleSearchApi)
	//r.GET("/chat/stream", DealWebsocket)
	//r.GET("/", DefaultDealHttpGet)
	r.GET("/*path", DefaultDealHttpGet)

	if err := r.Run(GetProxyAddr()); err != nil {
		log.Printf("Error: %v", err)
	}
}
