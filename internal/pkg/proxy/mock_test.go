package proxy

import (
	"context"
	"io"
	"net"
	"reflect"
	"testing"
	"time"
)

func Test_echoServer(t *testing.T) {
	type args struct {
		address string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		buf     []byte
	}{
		{
			name:    "test 1",
			args:    args{address: ":1234"},
			wantErr: false,
			buf:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8},
		},
		{
			name:    "test 2",
			args:    args{address: ":1234"},
			wantErr: false,
			buf:     []byte{1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				conn net.Conn
				err  error
			)

			b := make([]byte, 65535)

			go func() {
				time.Sleep(time.Millisecond * 10)
				conn, err = net.Dial("tcp", tt.args.address)
				if err != nil {
					t.Fatal(err)
				}
				conn.Write(tt.buf)
			}()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, err = echoServer(ctx, tt.args.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("echoServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			n, err := conn.(io.ReadWriter).Read(b)
			conn.Close()
			cancel()

			if !reflect.DeepEqual(b[:n], tt.buf) {
				t.Errorf("echoServer() = %v, want %v", b[:n], tt.buf)
			}
			time.Sleep(time.Millisecond * 10)
		})
	}
}
