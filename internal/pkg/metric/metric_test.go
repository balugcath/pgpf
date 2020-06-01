package metric

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/balugcath/pgpf/internal/pkg/config"
	"github.com/balugcath/pgpf/internal/pkg/transport"
)

func TestMetric_Start(t *testing.T) {
	type fields struct {
		Config      *config.Config
		Transporter transport.Transporter
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "test 1",
			fields: fields{
				Config: &config.Config{
					TimeoutHostStatus:    1,
					PrometheusListenPort: ":9090",
					Servers: map[string]*config.Server{
						"one": {
							PgConn: "one",
							Use:    true,
						},
						"two": {
							PgConn: "two",
							Use:    true,
						},
						"three": {
							PgConn: "three",
							Use:    true,
						},
						"four": {
							PgConn: "four",
							Use:    false,
						},
					},
				},
				Transporter: &transport.Mock{
					FHostStatus: func(h string, p bool) (bool, float64, error) {
						switch h {
						case "one":
							return false, 12, nil
						case "two":
							return true, 12, nil
						default:
							return true, 12, errors.New("err")
						}
					},
				},
			},
			want: []string{
				`pgpf_status_hosts{host="one"} 2`,
				`pgpf_status_hosts{host="three"} 0`,
				`pgpf_status_hosts{host="two"} 1`,
				`pgpf_transfer_bytes{host="one",type="a"} 1`,
				`pgpf_transfer_bytes{host="one",type="b"} 1`,
				`pgpf_client_connections{host="one"} 1`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMetric(tt.fields.Config, tt.fields.Transporter)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			s.Start(ctx)
			s.ClientConnInc("one")
			s.ClientConnInc("one")
			s.ClientConnDec("one")
			s.TransferBytes("one", "a", 1)
			s.TransferBytes("one", "b", 1)
			time.Sleep(time.Second * 5)
			resp, err := new(http.Client).Get("http://localhost" + tt.fields.Config.PrometheusListenPort + "/metrics")
			if err != nil {
				t.Errorf("TestMetric_Start() error = %v", err)
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("TestMetric_Start() error = %v", err)
			}
			resp.Body.Close()
			for _, v := range tt.want {
				matched, err := regexp.Match(v, b)
				if err != nil || !matched {
					t.Errorf("TestMetric_Start() error = %v", err)
				}
			}
			time.Sleep(time.Millisecond * 10)
		})
	}
}
