package gonet

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type ITcpTask interface {
	ParseMsg(data []byte, flag byte) bool // data is real data, flag is whther compress
	OnClose()
}

const (
	cmd_header_size     = 4          // 3字节指令长度,1字节是否压缩
	cmd_recive_max_size = 128 * 1024 // recive max size
	cmd_send_max_size   = 64 * 1024  // send max size
	cmd_verify_time     = 30         // verify time
)

type TcpTask struct {
	closed     int32         // whther is close
	verified   bool          // whther is  verified
	stopedChan chan struct{} //
	recvBuff   *ByteBuffer
	sendBuff   *ByteBuffer
	sendMutex  sync.Mutex
	Conn       net.Conn
	Derived    ITcpTask
	signal     chan struct{}
	Uid        uint64
	Nid        uint64
}

var netId uint64

func AutoIncNetId() uint64 {
	netId += 1
	if netId == 65535 {
		netId = 1
	}
	return netId
}

func NewTcpTask(conn net.Conn) *TcpTask {
	return &TcpTask{
		closed:     -1,
		verified:   false,
		Conn:       conn,
		stopedChan: make(chan struct{}, 1),
		recvBuff:   NewByteBuffer(),
		sendBuff:   NewByteBuffer(),
		signal:     make(chan struct{}, 1),
		Nid:        AutoIncNetId(),
	}
}

func (this *TcpTask) Signal() {
	select {
	case this.signal <- struct{}{}:
	default:
	}
}

func (this *TcpTask) SetUid(uid uint64) {
	this.Uid = uid
}

func (this *TcpTask) RemoteAddr() string {
	if this.Conn == nil {
		return ""
	}
	return this.Conn.RemoteAddr().String()
}

func (this *TcpTask) LocalAddr() string {
	if this.Conn == nil {
		return ""
	}
	return this.Conn.LocalAddr().String()
}

func (this *TcpTask) Stop() bool {
	if this.IsClosed() {
		return false
	}
	select {
	case this.stopedChan <- struct{}{}:
	default:
		return false
	}
	return true
}

func (this *TcpTask) Start() {
	if !atomic.CompareAndSwapInt32(&this.closed, -1, 0) {
		return
	}
	job := &sync.WaitGroup{}
	job.Add(1)
	go this.sendloop(job)
	go this.recvloop()
	job.Wait()
}

func (this *TcpTask) Close() {
	if !atomic.CompareAndSwapInt32(&this.closed, 0, 1) {
		return
	}
	this.Conn.Close()
	this.recvBuff.Reset()
	this.sendBuff.Reset()
	close(this.stopedChan)
	this.Derived.OnClose()
}

func (this *TcpTask) Reset() bool {
	if atomic.LoadInt32(&this.closed) != 1 {
		return false
	}
	if !this.IsVerified() {
		return false
	}
	this.closed = -1
	this.verified = false
	this.stopedChan = make(chan struct{})
	return true
}

func (this *TcpTask) IsClosed() bool {
	return atomic.LoadInt32(&this.closed) != 0
}

func (this *TcpTask) Verify() {
	this.verified = true
}

func (this *TcpTask) IsVerified() bool {
	return this.verified
}

func (this *TcpTask) Terminate() {
	this.Close()
}

func (this *TcpTask) SendData(buffer []byte, flag byte) bool {
	if this.IsClosed() {
		return false
	}
	bsize := len(buffer)
	this.sendMutex.Lock()
	this.sendBuff.Append(byte(bsize), byte(bsize>>8), byte(bsize>>16), flag)
	this.sendBuff.Append(buffer...)
	this.sendMutex.Unlock()
	this.Signal()
	return true
}

func (this *TcpTask) SendDataWithHead(head []byte, buffer []byte, flag byte) bool {
	if this.IsClosed() {
		return false
	}
	bsize := len(buffer) + len(head)
	this.sendMutex.Lock()
	this.sendBuff.Append(byte(bsize), byte(bsize>>8), byte(bsize>>16), flag)
	this.sendBuff.Append(head...)
	this.sendBuff.Append(buffer...)
	this.sendMutex.Unlock()
	this.Signal()
	return true
}

func (this *TcpTask) recvloop() {
	defer func() {
		if err := recover(); err != nil {
		}
	}()
	defer this.Close()
	var (
		neednum   int
		readnum   int
		err       error
		totalsize int
		datasize  int
		msgbuff   []byte
	)
	for {
		totalsize = this.recvBuff.RdSize()
		if totalsize < cmd_header_size {
			neednum = cmd_header_size - totalsize
			if this.recvBuff.WrSize() < neednum {
				this.recvBuff.WrGrow(neednum)
			}
			readnum, err = io.ReadAtLeast(this.Conn, this.recvBuff.WrBuf(), neednum)
			if err != nil {
				return
			}
			this.recvBuff.WrFlip(readnum)
			totalsize = this.recvBuff.RdSize()
		}
		msgbuff = this.recvBuff.RdBuf()
		datasize = int(msgbuff[0]) | int(msgbuff[1])<<8 | int(msgbuff[2])<<16
		if datasize > cmd_recive_max_size {
			return
		}
		if totalsize < cmd_header_size+datasize {
			neednum = cmd_header_size + datasize - totalsize
			if this.recvBuff.WrSize() < neednum {
				this.recvBuff.WrGrow(neednum)
			}
			readnum, err = io.ReadAtLeast(this.Conn, this.recvBuff.WrBuf(), neednum)
			if err != nil {
				return
			}
			this.recvBuff.WrFlip(readnum)
			msgbuff = this.recvBuff.RdBuf()
		}

		this.Derived.ParseMsg(msgbuff[cmd_header_size:cmd_header_size+datasize], msgbuff[3])

		// 清空，回退buffer
		this.recvBuff.RdFlip(cmd_header_size + datasize)
	}
}

func (this *TcpTask) sendloop(job *sync.WaitGroup) {
	defer func() {
		if err := recover(); err != nil {
		}
	}()
	defer this.Close()
	var (
		tmpByte  = NewByteBuffer()
		timeout  = time.NewTimer(time.Second * cmd_verify_time)
		writenum int
		err      error
	)
	defer timeout.Stop()
	job.Done()
	for {
		select {
		case <-this.signal:
			for {
				this.sendMutex.Lock()
				if this.sendBuff.RdReady() {
					tmpByte.Append(this.sendBuff.RdBuf()[:this.sendBuff.RdSize()]...)
					this.sendBuff.Reset()
				}
				this.sendMutex.Unlock()
				if !tmpByte.RdReady() {
					break
				}
				writenum, err = this.Conn.Write(tmpByte.RdBuf()[:tmpByte.RdSize()])
				if err != nil {
					return
				}
				tmpByte.RdFlip(writenum)
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
