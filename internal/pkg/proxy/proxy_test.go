package proxy

import (
	"bytes"
	"context"
	"io"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/balugcath/pgpf/internal/pkg/config"
	"github.com/balugcath/pgpf/internal/pkg/metric"
)

func TestProxy_copy(t *testing.T) {
	type fields struct {
		Metricer metric.Metricer
	}
	type args struct {
		dst      io.ReadWriter
		src      io.ReadWriter
		hostname string
		t        string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		buf     string
	}{
		{
			name: "test 1",
			args: args{
				dst:      bytes.NewBuffer(make([]byte, 0)),
				hostname: "one",
				t:        "read",
			},
			fields: fields{
				Metricer: metric.NewMock(),
			},
			wantErr: false,
			buf:     "1234",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Proxy{
				Metricer: tt.fields.Metricer,
			}
			tt.args.src = bytes.NewBufferString(tt.buf)
			if err := s.copy(tt.args.dst, tt.args.src, tt.args.hostname, tt.args.t); (err != nil) != tt.wantErr {
				t.Errorf("Proxy.copy() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.dst, bytes.NewBufferString(tt.buf)) {
				t.Errorf("Proxy.copy() got = %v, want %v", tt.args.dst, bytes.NewBufferString(tt.buf))
			}
		})
	}
}

func TestProxy_Serve1(t *testing.T) {
	type fields struct {
		Config   *config.Config
		Metricer metric.Metricer
	}
	type args struct {
		listenAddress string
		serverAddress string
		hostname      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test 1",
			args: args{
				serverAddress: "127.0.0.1:5454",
				listenAddress: "127.0.0.1:5455",
				hostname:      "one",
			},
			fields: fields{
				Config: &config.Config{
					Servers: map[string]*config.Server{
						"one": {
							Address: "127.0.0.1:5454",
						},
					},
				},
				Metricer: metric.NewMock(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctxServ, cancelServ := context.WithCancel(context.Background())
			defer cancelServ()
			go echoServer(ctxServ, tt.args.serverAddress)

			s, err := NewProxy(tt.fields.Config, tt.fields.Metricer).Listen(tt.args.listenAddress)
			if err != nil {
				t.Errorf("Proxy.Listen() error = %s", err)
			}

			ctx, cancel := context.WithCancel(context.Background())

			bWrite := []byte{1, 2, 3}
			bRead := make([]byte, 65535)
			go func() {
				time.Sleep(time.Second)
				conn, err := net.Dial("tcp", tt.args.listenAddress)
				if err != nil {
					t.Fatal(err)
				}
				conn.Write(bWrite)
				n, _ := conn.(io.ReadWriter).Read(bRead)
				bRead = bRead[:n]
				conn.Close()
				cancel()
			}()

			if err := s.Serve(ctx, tt.args.hostname); (err != nil) != tt.wantErr {
				t.Errorf("Proxy.Serve() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(bRead, bWrite) {
				t.Errorf("Proxy.Serve1() = %v, want %v", bWrite, bRead)
			}

		})
	}
}

func TestProxy_Serve2(t *testing.T) {
	type fields struct {
		Config   *config.Config
		Metricer metric.Metricer
	}
	type args struct {
		listenAddress string
		serverAddress string
		hostname      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test 1",
			args: args{
				serverAddress: "127.0.0.1:5454",
				listenAddress: "127.0.0.1:5455",
				hostname:      "one",
			},
			fields: fields{
				Config: &config.Config{
					Servers: map[string]*config.Server{
						"one": {
							Address: "127.0.0.1:5454",
						},
					},
				},
				Metricer: metric.NewMock(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewProxy(tt.fields.Config, tt.fields.Metricer).Listen(tt.args.listenAddress)
			if err != nil {
				t.Errorf("Proxy.Listen() error = %s", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(time.Second)
				conn, err := net.Dial("tcp", tt.args.listenAddress)
				if err != nil {
					t.Fatal(err)
				}
				conn.Close()
				cancel()
			}()

			if err := s.Serve(ctx, tt.args.hostname); (err != nil) != tt.wantErr {
				t.Errorf("Proxy.Serve2() error = %v, wantErr %v", err, tt.wantErr)
			}

		})
	}
}

func TestProxy_Serve3(t *testing.T) {
	type fields struct {
		Config   *config.Config
		Metricer metric.Metricer
	}
	type args struct {
		listenAddress string
		serverAddress string
		hostname      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test 1",
			args: args{
				serverAddress: "127.0.0.1:5454",
				listenAddress: "127.0.0.1:5455",
				hostname:      "one",
			},
			fields: fields{
				Config: &config.Config{
					Servers: map[string]*config.Server{
						"one": {
							Address: "127.0.0.1:5454",
						},
					},
				},
				Metricer: metric.NewMock(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctxServ, cancelServ := context.WithCancel(context.Background())
			defer cancelServ()
			go echoServer(ctxServ, tt.args.serverAddress)

			s, err := NewProxy(tt.fields.Config, tt.fields.Metricer).Listen(tt.args.listenAddress)
			if err != nil {
				t.Errorf("Proxy.Listen() error = %s", err)
			}

			ctx, cancel := context.WithCancel(context.Background())

			bWrite := []byte{1, 2, 3}
			bRead := make([]byte, 65535)
			go func() {
				time.Sleep(time.Second)
				conn, err := net.Dial("tcp", tt.args.listenAddress)
				if err != nil {
					t.Fatal(err)
				}
				conn.Write(bWrite)
				cancelServ()
				n, _ := conn.(io.ReadWriter).Read(bRead)
				bRead = bRead[:n]
				conn.Close()
				cancel()
			}()

			if err := s.Serve(ctx, tt.args.hostname); (err != nil) != tt.wantErr {
				t.Errorf("Proxy.Serve() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(bRead) != 0 {
				t.Errorf("Proxy.Serve3() = %v, want %v", bRead, []byte{})
			}

		})
	}
}
