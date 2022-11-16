package tcp

import (
	"context"
	"go-redis/interface/tcp"
	"go-redis/lib/logger"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// 服务启动的配置
type Config struct {
	Address string // ip+端口
}

func ListenAndServeWithSignal(cfg *Config, handler tcp.Handler) error {
	// 初始化变量
	closeChan := make(chan struct{})
	signalChan := make(chan os.Signal) //操作系统给线程的信号，使用Go中的singnal包

	//SIGHUP=1,终端控制进程结束(终端连接断开)
	//SIGINT=2,用户发送INTR字符(Ctrl+C)触发
	//SIGQUIT=3,用户发送QUIT字符(Ctrl+/)触发
	//SIGTERM=15,结束程序(可以被捕获、阻塞或忽略)
	signal.Notify(signalChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	//signal.Notify 向系统注册监控以下几个信号，如果接收到信号会写入 signalChan

	// 开启协程等待信号处理（即系统向 signalChan发信号 -》signalChan向closeChan发信号）
	go func() {
		sig := <-signalChan //如果没接收到系统信号，就卡在这一行
		switch sig {
		// 多判断一行，是否确实是上面的那几个信号
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM:
			closeChan <- struct{}{}
		}
	}()

	//指定通信协议，监听IP和端口（创建监听套接字）
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return err
	}

	logger.Info("start listen")
	ListenAndServe(listener, handler, closeChan)

	return nil
}

func ListenAndServe(listener net.Listener, handler tcp.Handler, closeChan <-chan struct{}) {
	// 情况二：程序被杀掉关闭，感知系统的信号；(因程序走不到defer)
	go func() {
		<-closeChan // 没数据的话，协程会卡在这一行
		logger.Info("shutting down")

		listener.Close()
		handler.Close()
	}()

	// 情况一：正常连接关闭(waitDone.Wait()之后执行)
	defer func() {
		listener.Close()
		handler.Close()
	}()

	ctx := context.Background()
	var waitDone sync.WaitGroup
	for {
		// 阻塞监听客户端新连接请求，连接成功，返回通信的socket（等待电话接入）
		conn, err := listener.Accept()
		if err != nil { //接收新连接错误了
			break
		}

		// 开启协程，每一个客户端通过socket通信，业务即循环等待读取操作
		waitDone.Add(1)
		logger.Info("accepted link")
		go func() {
			//协程服务完成后，等待组-1；(defer)handle出现panic时，也要 -1
			defer func() {
				waitDone.Done()
			}()

			// 业务逻辑
			handler.Handle(ctx, conn)
		}()
	}

	//接收新链接出错后 break，
	waitDone.Wait()
}
