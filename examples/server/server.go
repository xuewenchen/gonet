package main

import (
	"fmt"
	"github.com/xuewenchen/gonet"
	"net"
)

type EchoTask struct {
	gonet.TcpTask
}

func NewEchoTask(conn net.Conn) *EchoTask {
	s := &EchoTask{
		TcpTask: *gonet.NewTcpTask(conn),
	}
	s.Derived = s
	return s
}

func (this *EchoTask) ParseMsg(data []byte, flag byte) bool {

	this.Verify()

	fmt.Println("server:", string(data))

	this.SendData(data, flag)

	return true
}

func (this *EchoTask) OnClose() {

}

type EchoServer struct {
	gonet.Service
	tcpser *gonet.TcpServer
}

var serverm *EchoServer

func EchoServer_GetMe() *EchoServer {
	if serverm == nil {
		serverm = &EchoServer{
			tcpser: &gonet.TcpServer{},
		}
		serverm.Derived = serverm
	}
	return serverm
}

func (this *EchoServer) Init() bool {
	err := this.tcpser.Bind(":10000")
	if err != nil {
		fmt.Println("绑定端口失败")
		return false
	}
	return true
}

func (this *EchoServer) MainLoop() {
	conn, err := this.tcpser.Accept()
	if err != nil {
		return
	}
	NewEchoTask(conn).Start()
}

func (this *EchoServer) Reload() {
}

func (this *EchoServer) Final() bool {
	this.tcpser.Close()
	return true
}

func main() {

	EchoServer_GetMe().Main()

}
