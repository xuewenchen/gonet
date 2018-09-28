package gonet

import (
	"github.com/gorilla/websocket"
	"net"
	"net/url"
	"time"
)

type TcpClient struct {
}

func (this *TcpClient) Connect(address string) (*net.TCPConn, error) {
	return TcpDial(address)
}

func TcpDial(address string) (*net.TCPConn, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", address)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, err
	}

	conn.SetKeepAlive(true)
	conn.SetKeepAlivePeriod(1 * time.Minute)
	conn.SetNoDelay(true)
	conn.SetWriteBuffer(128 * 1024)
	conn.SetReadBuffer(128 * 1024)

	return conn, nil
}

type WebClient struct {
}

func (this *WebClient) Connect(address string) (*websocket.Conn, error) {

	u := url.URL{Scheme: "ws", Host: address, Path: "/"}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
