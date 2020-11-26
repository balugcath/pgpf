package rtmetric

import (
	"errors"
	"testing"
	"time"

	"github.com/balugcath/pgpf/internal/pkg/config"
	"github.com/stretchr/testify/assert"
)

var errFoo = errors.New("err foo")

func init() {
	timeUnit = time.Millisecond
}

type trMock struct {
	retOpen       map[string][]error
	retPromote    map[string][]error
	retIsRecovery map[string][]struct {
		f   bool
		err error
	}
	retHostVersion map[string][]struct {
		v   float64
		err error
	}
	key string
}

func (s *trMock) Open(key string) (err error) {
	s.key = key
	err = s.retOpen[s.key][0]
	if len(s.retOpen[s.key]) > 1 {
		s.retOpen[s.key] = s.retOpen[s.key][1:]
	}
	return
}

func (s *trMock) Close() {
	s.key = ""
}

func (s *trMock) IsRecovery() (f bool, err error) {
	f = s.retIsRecovery[s.key][0].f
	err = s.retIsRecovery[s.key][0].err
	if len(s.retIsRecovery[s.key]) > 1 {
		s.retIsRecovery[s.key] = s.retIsRecovery[s.key][1:]
	}
	return
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

func (*m) Register(int, string, string, ...string) error { return nil }
func (*m) Add(string, ...interface{}) error              { return nil }
func (*m) Inc(string, ...interface{}) error              { return nil }
func (*m) Dec(string, ...interface{}) error              { return nil }

func TestMetric_Start(t *testing.T) {
	type fields struct {
		Config    *config.Config
		transport *trMock
		m         *m
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]float64
	}{
		{
			name: "test 1",
			fields: fields{
				m: &m{r: make(map[string]float64)},
				transport: &trMock{
					retOpen: map[string][]error{"one": {nil}, "two": {nil}, "three": {nil}, "four": {errFoo}},
					retIsRecovery: map[string][]struct {
						f   bool
						err error
					}{"one": {{false, nil}}, "two": {{true, nil}}, "three": {{true, errFoo}}},
				},
				Config: &config.Config{
					HostStatusIntervalSec: 1,
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
							Use:    true,
						},
						"five": {
							PgConn: "five",
							Use:    false,
						},
					},
				},
			},
			want: map[string]float64{"one": 2, "three": 0, "two": 1, "four": 0},
		},
		{
			name: "test 2",
			fields: fields{
				m: &m{r: make(map[string]float64)},
				Config: &config.Config{
					HostStatusIntervalSec: 0,
				},
			},
			want: make(map[string]float64),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewHostStatus(tt.fields.Config, tt.fields.m, tt.fields.transport)
			s.Start()
			time.Sleep(time.Millisecond * 10)
			if !assert.Equal(t, tt.want, tt.fields.m.r) {
				t.Errorf("TestMetric_Start() = %v, want %v", tt.fields.m.r, tt.want)
			}
		})
	}
}
