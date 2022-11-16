package tcp

import (
	"bufio"
	"context"
	"go-redis/lib/logger"
	atomic "go-redis/lib/sync/atomic"
	"go-redis/lib/sync/wait"
	"io"
	"net"
	"sync"
	"time"
)

// 回复内容

// 客户端信息 (快捷键：command + n 实现一个 Close的接口)
type EchoClient struct {
	Conn    net.Conn  // 客户端连接，底层就是一个 socket,把这个 stocket关闭
	Waiting wait.Wait // 不用系统的，使用自定义的，自定义超时机制
}

func (e *EchoClient) Close() error {
	// 等待当前的连接工作做完后或超时，再关闭连接
	e.Waiting.WaitWithTimeout(10 * time.Second)
	_ = e.Conn.Close() // 如果关闭过程中出错，不处理了

	return nil
}

//++++++++++++++++++++++++++++++++客户端++++++++++++++++++++++++++++++++

// 服务端业务引擎
type EchoHandler struct {
	activeConn sync.Map       //记录有多少个客户端连接,并发线程问题
	closing    atomic.Boolean //如果正在关闭服务端，不要转发请求过来了，bool类型在多线程并发的问题
}

func NewEchoHandler() *EchoHandler {
	return &EchoHandler{}
}

func (handler *EchoHandler) Handle(ctx context.Context, conn net.Conn) {
	// 服务端关闭，那连接过来的客户端关闭，不建立连接
	if handler.closing.Get() {
		_ = conn.Close()
	}

	client := &EchoClient{
		Conn: conn,
	}

	// map原子操作，使用store方法, 入参是 key,val，把客户端存储起来
	handler.activeConn.Store(client, struct{}{})

	// 使用网络报文进行数据传输，以 \n 作为数据传输结束标志（bufio包里的）
	buffer := bufio.NewReader(conn)
	for {
		// 【may occurs，可能出现的情况: client EOF, client timeout, server early close】
		//buffer.ReadString(),该方法从输入中读取内容，直到碰到 delim 指定的字符，然后将读取到的内容连同 delim 字符一起放到缓冲区。
		//如果找不到 delim 返回 err;读完就 io.EOF，（数据结束符）
		msg, err := buffer.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				logger.Info("Connect close")
				handler.activeConn.Delete(client)
			} else {
				logger.Warn(err)
			}

			// 读取出错后，消息也回发不了，直接 return
			return
		}

		// add和done是等待业务执行完才允许关闭
		client.Waiting.Add(1)
		buffer := []byte(msg)
		_, _ = conn.Write(buffer) //回写操作
		client.Waiting.Done()
	}
}

func (handler *EchoHandler) Close() error {
	logger.Info("handler shutting down")
	handler.closing.Set(true)

	// 服务端关闭，遍历把客户端也关闭
	handler.activeConn.Range(func(key, value any) bool {
		client := key.(*EchoClient)
		_ = client.Close()

		return true
	})

	return nil
}

//++++++++++++++++++++++++++++++++服务端++++++++++++++++++++++++++++++++
