package main

import (
	"fmt"
	"github.com/xuewenchen/gonet"
	"time"
)

type Client struct {
	gonet.TcpTask
	mclient *gonet.TcpClient
}

func NewClient() *Client {
	s := &Client{
		TcpTask: *gonet.NewTcpTask(nil),
	}
	s.Derived = s
	return s
}

func (this *Client) Connect(addr string) bool {

	conn, err := this.mclient.Connect(addr)
	if err != nil {
		fmt.Println("连接失败 ", addr)
		return false
	}

	this.Conn = conn

	this.Start()

	fmt.Println("连接成功 ", addr)
	return true
}

func (this *Client) ParseMsg(data []byte, flag byte) bool {

	fmt.Println("client:", string(data))

	return true
}

func (this *Client) OnClose() {

}

func main() {

	client := NewClient()

	if !client.Connect("127.0.0.1:10000") {
		return
	}

	for {

		client.SendData([]byte("abc"), byte(1))
		time.Sleep(time.Second * 1)
	}
}
