package proxy

import (
	"context"
	"errors"
	"net"
)

func echoServer(ctx context.Context, address string) (int, error) {
	l, err := net.Listen("tcp", address)
	if err != nil {
		return 0, err
	}
	go func() {
		<-ctx.Done()
		l.Close()
	}()

	conn, err := l.Accept()
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	b := make([]byte, 65535)
	rB, err := conn.Read(b)
	if err != nil {
		return 0, err
	}
	wB, err := conn.Write(b[:rB])
	if err != nil {
		return 0, err
	}
	if rB != wB {
		return 0, errors.New("error read write")
	}
	return rB, nil
}
