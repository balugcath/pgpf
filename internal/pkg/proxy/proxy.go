package proxy

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/balugcath/pgpf/internal/pkg/config"
	"github.com/balugcath/pgpf/pkg/promwrap"
	log "github.com/sirupsen/logrus"
)

var (
	// ErrTerminate ...
	ErrTerminate = errors.New("terminate")
)

const (
	pgpfClientConnectionsMetricName = "pgpf_client_connections"
	pgpfClientConnectionsMetricHelp = "how many client connected, partition by host"
	pgpfTransferBytesMetricName     = "pgpf_transfer_bytes"
	pgpfTransferBytesMetricHelp     = "how many bytes transferred, partition by host"
)

// Proxy ...
type Proxy struct {
	*config.Config
	metric promwrap.Interface
	net.Listener
}

var timeUnit = time.Second

// NewProxy ...
func NewProxy(config *config.Config, metric promwrap.Interface) *Proxy {
	s := &Proxy{
		metric: metric,
		Config: config,
	}
	s.metric.Register(promwrap.GaugeVec, pgpfClientConnectionsMetricName, pgpfClientConnectionsMetricHelp, []string{"host"}...)
	s.metric.Register(promwrap.CounterVec, pgpfTransferBytesMetricName, pgpfClientConnectionsMetricHelp, []string{"host", "type"}...)
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
	log.Debugf("proxy.Serve() host %s start", hostname)
	defer log.Debugf("proxy.Serve() host %s exit", hostname)

	addressServer := s.Servers[hostname].Address

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer s.Listener.Close()
		<-doneCtx.Done()
		log.Debugf("proxy.Serve() host %s done context received", hostname)
	}()

	for {
		client, err := s.Listener.Accept()
		if err != nil {
			select {
			case <-doneCtx.Done():
				return ErrTerminate
			default:
				log.Errorf("proxy.Serve() host %s %s", hostname, err)
				continue
			}
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			var wgCopy sync.WaitGroup
			defer wgCopy.Wait()

			server, err := net.DialTimeout("tcp", addressServer, time.Duration(s.Config.TimeoutMasterDialSec)*timeUnit)
			if err != nil {
				log.Errorf("proxy.Serve() host %s %s", hostname, err)
				client.Close()
				return
			}

			log.Debugf("proxy.Serve() start %s - %s : %s - %s", client.RemoteAddr().String(), client.LocalAddr().String(),
				server.LocalAddr().String(), server.RemoteAddr().String())
			defer log.Debugf("proxy.Serve() stop %s - %s : %s - %s", client.RemoteAddr().String(), client.LocalAddr().String(),
				server.LocalAddr().String(), server.RemoteAddr().String())

			s.metric.Inc(pgpfClientConnectionsMetricName, []interface{}{hostname}...)
			defer s.metric.Dec(pgpfClientConnectionsMetricName, []interface{}{hostname}...)

			defer client.Close()
			defer server.Close()

			terminateCtx, cancel := context.WithCancel(context.Background())

			wgCopy.Add(2)
			go func() {
				defer wgCopy.Done()
				defer cancel()
				if _, err := s.copy(server, client, hostname, "read"); err != nil && err != io.EOF {
					log.Errorf("proxy.Serve() host %s %s", hostname, err)
				}
			}()
			go func() {
				defer wgCopy.Done()
				defer cancel()
				if _, err := s.copy(client, server, hostname, "write"); err != nil && err != io.EOF {
					log.Errorf("proxy.Serve() host %s %s", hostname, err)
				}
			}()

			select {
			case <-doneCtx.Done():
			case <-terminateCtx.Done():
			}

		}()
	}
}

func (s *Proxy) copy(dst io.Writer, src io.Reader, hostname, t string) (int, error) {
	log.Debugf("proxy.copy() host %s mode %s start", hostname, t)
	defer log.Debugf("proxy.copy() host %s mode %s exit", hostname, t)
	b := make([]byte, 65535)
	total := 0
	for {
		n, err := src.Read(b)
		total += n
		if n != 0 {
			_, err := dst.Write(b[:n])
			if err != nil {
				return total, err
			}
		}
		if err != nil {
			return total, err
		}
		s.metric.Add(pgpfTransferBytesMetricName, []interface{}{hostname, t, float64(n)}...)
		log.Debugf("proxy.copy() host %s mode %s %d bytes", hostname, t, n)
	}
}
