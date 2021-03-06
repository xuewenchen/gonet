package gonet

import (
	"net"
	"time"
)

type TcpServer struct {
	listener *net.TCPListener
}

func (this *TcpServer) Bind(address string) error {

	tcpAddr, err := net.ResolveTCPAddr("tcp4", address)
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}
	this.listener = listener
	return nil
}

func (this *TcpServer) Accept() (*net.TCPConn, error) {

	this.listener.SetDeadline(time.Now().Add(time.Second * 1))

	conn, err := this.listener.AcceptTCP()
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

func (this *TcpServer) BindAccept(address string, handler func(*net.TCPConn)) error {
	err := this.Bind(address)
	if err != nil {
		return err
	}
	go func() {
		for {
			conn, err := this.Accept()
			if err != nil {
				continue
			}
			handler(conn)
		}
	}()
	return nil
}

func (this *TcpServer) Close() error {
	return this.listener.Close()
}
