package rtmetric

import (
	"errors"
	"testing"
	"time"

	"github.com/balugcath/pgpf/internal/pkg/config"
	"github.com/stretchr/testify/assert"
)

type tr struct{}

func (tr) HostStatus(conn string) (bool, float64, error) {
	switch conn {
	case "one":
		return false, 12, nil
	case "two":
		return true, 12, nil
	default:
		return true, 12, errors.New("err")
	}
}

type m struct {
	r map[string]float64
}

func (s *m) Set(_ string, opts ...interface{}) error {
	k, ok := opts[0].(string)
	if !ok {
		return nil
	}
	v, ok := opts[1].(float64)
	if !ok {
		return nil
	}
	s.r[k] = v
	return nil
}

func (*m) Register(_ int, _, _ string, _ ...string) error { return nil }
func (*m) Add(_ string, _ ...interface{}) error           { return nil }
func (*m) Inc(_ string, _ ...interface{}) error           { return nil }
func (*m) Dec(_ string, _ ...interface{}) error           { return nil }

func TestMetric_Start(t *testing.T) {
	type fields struct {
		Config *config.Config
		m      m
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]float64
	}{
		{
			name: "test 1",
			fields: fields{
				m: m{r: make(map[string]float64)},
				Config: &config.Config{
					TimeoutHostStatus: 1,
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
			},
			want: map[string]float64{"one": 2, "three": 0, "two": 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewHostStatus(tt.fields.Config, &tt.fields.m, tr{})
			s.Start()
			time.Sleep(time.Second)
			if !assert.Equal(t, tt.fields.m.r, tt.want) {
				t.Errorf("TestMetric_Start() = %v, want %v", tt.fields.m.r, tt.want)
			}
		})
	}
}
