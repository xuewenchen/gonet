package gonet

import (
	"container/list"
	"github.com/gorilla/websocket"
	"sync"
	"sync/atomic"
	"time"
)

type IWebSocketTask interface {
	ParseMsg(data []byte, flag byte) bool
	OnClose()
}

type WebSocketTask struct {
	closed      int32
	verified    bool
	stopedChan  chan bool
	sendmsglist *list.List
	sendMutex   sync.Mutex
	Conn        *websocket.Conn
	Derived     IWebSocketTask
	msgchan     chan []byte
	signal      chan int
}

func NewWebSocketTask(conn *websocket.Conn) *WebSocketTask {
	return &WebSocketTask{
		closed:      -1,
		verified:    false,
		Conn:        conn,
		stopedChan:  make(chan bool, 1),
		sendmsglist: list.New(),
		msgchan:     make(chan []byte, 1024),
		signal:      make(chan int, 1),
	}
}

func (this *WebSocketTask) Signal() {
	select {
	case this.signal <- 1:
	default:
	}
}

func (this *WebSocketTask) Stop() {
	if !this.IsClosed() && len(this.stopedChan) == 0 {
		this.stopedChan <- true
	}
}

func (this *WebSocketTask) Start() {
	if atomic.CompareAndSwapInt32(&this.closed, -1, 0) {
		go this.sendloop()
		go this.recvloop()
	}
}

func (this *WebSocketTask) Close() {
	if atomic.CompareAndSwapInt32(&this.closed, 0, 1) {
		this.Conn.Close()
		close(this.stopedChan)
		this.Derived.OnClose()
	}
}

func (this *WebSocketTask) Reset() {
	if atomic.LoadInt32(&this.closed) == 1 {
		this.closed = -1
		this.verified = false
		this.stopedChan = make(chan bool)
	}
}

func (this *WebSocketTask) IsClosed() bool {
	return atomic.LoadInt32(&this.closed) != 0
}

func (this *WebSocketTask) Verify() {
	this.verified = true
}

func (this *WebSocketTask) IsVerified() bool {
	return this.verified
}

func (this *WebSocketTask) Terminate() {
	this.Close()
}

func (this *WebSocketTask) AsyncSend(buffer []byte, flag byte) bool {
	if this.IsClosed() {
		return false
	}

	bsize := len(buffer)

	//glog.Info("[AsynSend] raw", buffer, bsize)

	totalsize := bsize + 4
	sendbuffer := make([]byte, 0, totalsize)
	sendbuffer = append(sendbuffer, byte(bsize), byte(bsize>>8), byte(bsize>>16), flag)
	sendbuffer = append(sendbuffer, buffer...)
	this.msgchan <- sendbuffer

	//glog.Info("[AsynSend] final", sendbuffer, bsize)

	return true
}

func (this *WebSocketTask) recvloop() {
	defer func() {
		if err := recover(); err != nil {
		}
	}()
	defer this.Close()

	var (
		datasize int
	)

	for {
		_, bytemsg, err := this.Conn.ReadMessage()
		if nil != err {
			return
		}

		datasize = int(bytemsg[0]) | int(bytemsg[1])<<8 | int(bytemsg[2])<<16

		if datasize > cmd_recive_max_size {
			return
		}

		this.Derived.ParseMsg(bytemsg[cmd_header_size:], bytemsg[3])
	}
}

func (this *WebSocketTask) sendloop() {
	defer func() {
		if err := recover(); err != nil {
		}
	}()
	defer this.Close()

	var (
		timeout = time.NewTimer(time.Second * cmd_verify_time)
	)

	defer timeout.Stop()

	for {
		select {
		case bytemsg := <-this.msgchan:
			if nil != bytemsg && len(bytemsg) > 0 {
				err := this.Conn.WriteMessage(websocket.BinaryMessage, bytemsg)
				if nil != err {
					return
				}
			} else {
				return
			}
		case <-this.stopedChan:
			return
		case <-timeout.C:
			if !this.IsVerified() {
				return
			}
		}
	}
}
