package proxy

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/balugcath/pgpf/internal/pkg/config"
	"github.com/balugcath/pgpf/internal/pkg/metric"
)

var (
	// ErrTerminate ...
	ErrTerminate = errors.New("terminate")
)

// Proxy ...
type Proxy struct {
	*config.Config
	metric.Metricer
	net.Listener
}

// NewProxy ...
func NewProxy(config *config.Config, metricer metric.Metricer) *Proxy {
	s := &Proxy{
		Metricer: metricer,
		Config:   config,
	}
	return s
}

// Listen ...
func (s *Proxy) Listen(address string) (*Proxy, error) {
	l, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	s.Listener = l
	return s, nil
}

// Serve ...
func (s *Proxy) Serve(doneCtx context.Context, hostname string) error {
	var wg sync.WaitGroup
	defer wg.Wait()

	addressServer := s.Servers[hostname].Address

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer s.Listener.Close()
		<-doneCtx.Done()
	}()

	for {
		client, err := s.Listener.Accept()
		if err != nil {
			select {
			case <-doneCtx.Done():
				return ErrTerminate
			default:
				log.Println(err)
				continue
			}
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			var wgCopy sync.WaitGroup
			defer wgCopy.Wait()

			server, err := net.DialTimeout("tcp", addressServer, time.Duration(s.Config.TimeoutMasterDial))
			if err != nil {
				log.Println(err)
				client.Close()
				return
			}

			s.ClientConnInc(hostname)
			defer s.ClientConnDec(hostname)

			defer client.Close()
			defer server.Close()

			terminateCtx, cancel := context.WithCancel(context.Background())

			wgCopy.Add(2)
			go func() {
				defer wgCopy.Done()
				defer cancel()
				if err := s.copy(server, client, hostname, "read"); err != nil {
					log.Println(err)
				}
			}()
			go func() {
				defer wgCopy.Done()
				defer cancel()
				if err := s.copy(client, server, hostname, "write"); err != nil {
					log.Println(err)
				}
			}()

			select {
			case <-doneCtx.Done():
			case <-terminateCtx.Done():
			}

		}()
	}
}

func (s *Proxy) copy(dst, src io.ReadWriter, hostname, t string) error {
	b := make([]byte, 65535)
	for {
		n, err := src.Read(b)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		_, err = dst.Write(b[:n])
		if err != nil {
			return err
		}
		s.TransferBytes(hostname, t, n)
	}
}
