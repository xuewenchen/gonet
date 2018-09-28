package gonet

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

type ServiceDriver interface {
	Init() bool
	MainLoop()
	Reload()
	Final() bool
}

type Service struct {
	terminate bool
	Driver    ServiceDriver
}

func (this *Service) Terminate() {
	this.terminate = true
}

func (this *Service) isTerminate() bool {
	return this.terminate
}

func (this *Service) SetCpuNum(num int) {
	if num > 0 {
		runtime.GOMAXPROCS(num)
	} else if num == -1 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
}

func (this *Service) Main() bool {

	defer func() {
		if err := recover(); err != nil {
		}
	}()

	runtime.GOMAXPROCS(runtime.NumCPU())

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGABRT, syscall.SIGTERM, syscall.SIGPIPE, syscall.SIGHUP)
	go func() {
		for sig := range ch {
			switch sig {
			case syscall.SIGHUP:
				this.Driver.Reload()
			case syscall.SIGPIPE:
			default:
				this.Terminate()
			}
		}
	}()

	if !this.Driver.Init() {
		return false
	}

	for !this.isTerminate() {
		this.Driver.MainLoop()
	}

	this.Driver.Final()
	return true
}
