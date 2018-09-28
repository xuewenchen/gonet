package gonet

import (
	"github.com/gorilla/websocket"
	"net/http"
)

type IWebSocketServer interface {
	OnWebAccept(conn *websocket.Conn)
}

type WebSocketServer struct {
	WebDerived IWebSocketServer
}

var upgrader = websocket.Upgrader{} // use default options

func (this *WebSocketServer) WebBind(addr string) error {
	http.HandleFunc("/", this.WebListen)
	err := http.ListenAndServe(addr, nil)
	if nil != err {
	}
	return nil
}

func (this *WebSocketServer) WebListen(w http.ResponseWriter, r *http.Request) {

	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	this.WebDerived.OnWebAccept(c)
}

func (this *WebSocketServer) WebClose() error {
	return nil
}
