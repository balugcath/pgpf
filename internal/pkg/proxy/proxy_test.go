package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"testing/iotest"

	"github.com/balugcath/pgpf/internal/pkg/config"
)

type m struct {
}

func (m) Register(_ int, _, _ string, _ ...string) error { return nil }
func (m) Add(_ string, _ ...interface{}) error           { return nil }
func (m) Set(_ string, _ ...interface{}) error           { return nil }
func (m) Inc(_ string, _ ...interface{}) error           { return nil }
func (m) Dec(_ string, _ ...interface{}) error           { return nil }

type rw struct {
	data io.ReadWriter
	err  error
}

func (s *rw) Read(p []byte) (int, error) {
	return s.data.Read(p)
}

func (s *rw) Write(p []byte) (n int, err error) {
	n, err = s.data.Write(p)
	if s.err != nil {
		err = s.err
	}
	return
}

const (
	data = "Hi gophers!"
)

func TestProxy_copy(t *testing.T) {
	type args struct {
		dst io.ReadWriter
		src io.Reader
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
		buf     string
	}{
		{
			name: "test 1",
			args: args{
				dst: &bytes.Buffer{},
				src: bytes.NewBufferString(data),
			},
			wantErr: io.EOF,
		},
		{
			name: "test 2",
			args: args{
				dst: &bytes.Buffer{},
				src: iotest.TimeoutReader(bytes.NewBufferString(data)),
			},
			wantErr: iotest.ErrTimeout,
		},
		{
			name: "test 3",
			args: args{
				dst: &rw{data: &bytes.Buffer{}, err: io.EOF},
				src: bytes.NewBufferString(data),
			},
			wantErr: io.EOF,
		},
		{
			name: "test 4",
			args: args{
				dst: &bytes.Buffer{},
				src: iotest.DataErrReader(bytes.NewBufferString(data)),
			},
			wantErr: io.EOF,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Proxy{
				metric: m{},
			}
			n, err := s.copy(tt.args.dst, tt.args.src, "a", "b")
			if err != tt.wantErr {
				t.Errorf("Proxy.copy() error = %v, wantErr %v", err, tt.wantErr)
			}
			t.Log(n, err)
			if n != 0 {
				b := make([]byte, n)
				tt.args.dst.Read(b)
				if data != string(b) {
					t.Errorf("Proxy.copy() want %+v, got %+v", data, b)
				}
			}
		})
	}
}

func TestProxy_Proxy1(t *testing.T) {
	t.Run("TestProxy_Proxy1", func(t *testing.T) {
		lAddr := "127.0.0.1:3434"

		terminateCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, data)
		}))
		defer ts.Close()

		s, err := NewProxy(&config.Config{
			Servers: map[string]*config.Server{
				"one": {
					Address: ts.Listener.Addr().String(),
				},
			}}, m{}).Listen(lAddr)
		if err != nil {
			t.Fatal(err)
		}

		go func() {
			s.Serve(terminateCtx, "one")
		}()

		res, err := http.Get("http://" + lAddr + "/")
		if err != nil {
			t.Fatal(err)
		}

		reply, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}

		cancel()

		if string(reply) != data+"\n" {
			t.Errorf("TestProxy_Proxy1 got %+v, want %+v", string(reply), data+"\n")
		}
	})
}

func TestProxy_Proxy2(t *testing.T) {
	t.Run("TestProxy_Proxy2", func(t *testing.T) {
		lAddr := "127.0.0.1:3435"

		terminateCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, data)
		}))
		defer ts.Close()

		s, err := NewProxy(&config.Config{
			Servers: map[string]*config.Server{
				"one": {
					Address: ts.Listener.Addr().String(),
				},
			}}, m{}).Listen(lAddr)
		if err != nil {
			t.Fatal(err)
		}

		go func() {
			s.Serve(terminateCtx, "one")
		}()

		cli, err := net.Dial("tcp", lAddr)
		if err != nil {
			t.Fatal(err)
		}
		cli.Close()
		runtime.Gosched()
		cancel()
	})
}

func TestProxy_Proxy3(t *testing.T) {
	t.Run("TestProxy_Proxy3", func(t *testing.T) {

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, data)
		}))
		defer ts.Close()

		_, err := NewProxy(&config.Config{
			Servers: map[string]*config.Server{
				"one": {
					Address: ts.Listener.Addr().String(),
				},
			}}, m{}).Listen(ts.Listener.Addr().String())
		if err == nil {
			t.Errorf("want not nil error")
		}
	})
}
