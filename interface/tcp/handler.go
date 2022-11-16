package tcp

import (
	"context"
	"net"
)

// TCP处理，只处理链接；业务交给 Handle 里处理
type Handler interface {
	Handle(ctx context.Context, conn net.Conn)
	Close() error
}
