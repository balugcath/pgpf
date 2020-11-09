package promwrap

import (
	"io/ioutil"
	"net/http"
	"regexp"
	"testing"
)

const (
	path = "/metric"
	port = ":8080"
)

var s *Prom

func init() {
	s = NewProm(port, path)
	go func() {
		s.Start()
	}()
}

func TestProm_Add1(t *testing.T) {
	type metric struct {
		kind int
		name string
		help string
		opts []string
		data []float64
	}
	type args struct {
		metric []metric
	}
	tests := []struct {
		name string
		want []string
		args args
	}{
		{
			name: "test 1",
			args: args{
				metric: []metric{
					{
						kind: Gauge,
						name: "gauge_add",
						data: []float64{10, 11, 12},
					},
					{
						kind: GaugeVec,
						name: "gauge_vec_add",
						opts: []string{"one", "two"},
						data: []float64{10, 33},
					},
					{
						kind: Counter,
						name: "counter_add",
						opts: []string{"one", "two"},
						data: []float64{30, 33},
					},
					{
						kind: CounterVec,
						name: "counter_vec_add",
						opts: []string{"one", "two"},
						data: []float64{30, 33},
					},
				},
			},
			want: []string{
				`gauge_add 33`,
				`counter_add 63`,
				`gauge_vec_add{one="one",two="two"} 43`,
				`counter_vec_add{one="one",two="two"} 63`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			for _, v := range tt.args.metric {
				err := s.Register(v.kind, v.name, v.help, v.opts...)
				if err != nil {
					t.Errorf("TestProm_Add1() error = %v", err)
				}
			}

			for _, v := range tt.args.metric {
				var err error
				switch v.kind {
				case Gauge, Counter:
					for i := range v.data {
						err = s.Add(v.name, v.data[i])
					}
				case GaugeVec, CounterVec:
					for i := range v.data {
						err = s.Add(v.name, []interface{}{v.opts[0], v.opts[1], 1, v.data[i]}...)
					}
				}
				if err != nil {
					t.Errorf("TestProm_Add1() error = %v", err)
				}
			}

			resp, err := new(http.Client).Get("http://localhost" + port + path)
			if err != nil {
				t.Errorf("TestProm_Add1() error = %v", err)
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("TestProm_Add1() error = %v", err)
			}
			resp.Body.Close()
			for _, v := range tt.want {
				matched, err := regexp.Match(v, b)
				if err != nil || !matched {
					t.Errorf("TestProm_Add1() error = %v", err)
				}
			}
		})
	}
}

func TestProm_Inc1(t *testing.T) {
	type metric struct {
		kind int
		name string
		help string
		opts []string
	}
	type args struct {
		metric []metric
	}
	tests := []struct {
		name string
		want []string
		args args
	}{
		{
			name: "test 1",
			args: args{
				metric: []metric{
					{
						kind: Gauge,
						name: "gauge_inc",
					},
					{
						kind: GaugeVec,
						name: "gauge_vec_inc",
						opts: []string{"one", "two"},
					},
					{
						kind: Counter,
						name: "counter_inc",
					},
					{
						kind: CounterVec,
						name: "counter_vec_inc",
						opts: []string{"one", "two"},
					},
				},
			},
			want: []string{
				`gauge_inc 1`,
				`counter_inc 1`,
				`gauge_vec_inc{one="one",two="two"} 1`,
				`counter_vec_inc{one="one",two="two"} 1`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			for _, v := range tt.args.metric {
				err := s.Register(v.kind, v.name, v.help, v.opts...)
				if err != nil {
					t.Errorf("TestProm_Inc1() error = %v", err)
				}
			}

			for _, v := range tt.args.metric {
				var err error
				switch v.kind {
				case Gauge, Counter:
					err = s.Inc(v.name)
				case GaugeVec, CounterVec:
					err = s.Inc(v.name, []interface{}{v.opts[0], 1, v.opts[1]}...)
				}
				if err != nil {
					t.Errorf("TestProm_Inc1() error = %v", err)
				}
			}

			resp, err := new(http.Client).Get("http://localhost" + port + path)
			if err != nil {
				t.Errorf("TestProm_Inc1() error = %v", err)
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("TestProm_Inc1() error = %v", err)
			}
			resp.Body.Close()
			for _, v := range tt.want {
				matched, err := regexp.Match(v, b)
				if err != nil || !matched {
					t.Errorf("TestProm_Inc1() error = %v", err)
				}
			}
		})
	}
}

func TestProm_Dec1(t *testing.T) {
	type metric struct {
		kind int
		name string
		help string
		opts []string
	}
	type args struct {
		metric []metric
	}
	tests := []struct {
		name string
		want []string
		args args
	}{
		{
			name: "test 1",
			args: args{
				metric: []metric{
					{
						kind: Gauge,
						name: "gauge_dec",
					},
					{
						kind: GaugeVec,
						name: "gauge_vec_dec",
						opts: []string{"one", "two"},
					},
				},
			},
			want: []string{
				`gauge_dec -1`,
				`gauge_vec_dec{one="one",two="two"} -1`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			for _, v := range tt.args.metric {
				err := s.Register(v.kind, v.name, v.help, v.opts...)
				if err != nil {
					t.Errorf("TestProm_Dec1() error = %v", err)
				}
			}

			for _, v := range tt.args.metric {
				var err error
				switch v.kind {
				case Gauge:
					err = s.Dec(v.name)
				case GaugeVec:
					err = s.Dec(v.name, []interface{}{v.opts[0], 1, v.opts[1]}...)
				}
				if err != nil {
					t.Errorf("TestProm_Dec1() error = %v", err)
				}
			}

			resp, err := new(http.Client).Get("http://localhost" + port + path)
			if err != nil {
				t.Errorf("TestProm_Dec1() error = %v", err)
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("TestProm_Dec1() error = %v", err)
			}
			resp.Body.Close()
			// log.Printf(string(b))
			for _, v := range tt.want {
				matched, err := regexp.Match(v, b)
				if err != nil || !matched {
					t.Errorf("TestProm_Dec1() error = %v", err)
				}
			}
		})
	}
}

func TestProm_Set1(t *testing.T) {
	type metric struct {
		kind int
		name string
		help string
		opts []string
		data []float64
	}
	type args struct {
		metric []metric
	}
	tests := []struct {
		name string
		want []string
		args args
	}{
		{
			name: "test 1",
			args: args{
				metric: []metric{
					{
						kind: Gauge,
						name: "gauge_set",
						data: []float64{30, 33},
					},
					{
						kind: GaugeVec,
						name: "gauge_vec_set",
						opts: []string{"one", "two"},
						data: []float64{40, 43},
					},
				},
			},
			want: []string{
				`gauge_set 33`,
				`gauge_vec_set{one="one",two="two"} 43`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			for _, v := range tt.args.metric {
				err := s.Register(v.kind, v.name, v.help, v.opts...)
				if err != nil {
					t.Errorf("TestProm_Set1() error = %v", err)
				}
			}

			for _, v := range tt.args.metric {
				var err error
				switch v.kind {
				case Gauge, Counter:
					for i := range v.data {
						err = s.Set(v.name, v.data[i])
					}
				case GaugeVec, CounterVec:
					for i := range v.data {
						err = s.Set(v.name, []interface{}{v.opts[0], 1, v.opts[1], v.data[i]}...)
					}
				}
				if err != nil {
					t.Errorf("TestProm_Set1() error = %v", err)
				}
			}

			resp, err := new(http.Client).Get("http://localhost" + port + path)
			if err != nil {
				t.Errorf("TestProm_Set1() error = %v", err)
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("TestProm_Set1() error = %v", err)
			}
			resp.Body.Close()
			t.Log(string(b))
			for _, v := range tt.want {
				matched, err := regexp.Match(v, b)
				if err != nil || !matched {
					t.Errorf("TestProm_Set1() error = %v", err)
				}
			}
		})
	}
}

func TestProm_ErrMetricNotExist(t *testing.T) {
	err := s.Add("v.name", 1)
	if err != ErrMetricNotExist {
		t.Errorf("TestProm_ErrMetricNotExist() error = %v", err)
	}
	err = s.Inc("v.name")
	if err != ErrMetricNotExist {
		t.Errorf("TestProm_ErrMetricNotExist() error = %v", err)
	}
	err = s.Dec("v.name")
	if err != ErrMetricNotExist {
		t.Errorf("TestProm_ErrMetricNotExist() error = %v", err)
	}
	err = s.Set("v.name")
	if err != ErrMetricNotExist {
		t.Errorf("TestProm_ErrMetricNotExist() error = %v", err)
	}
}

func TestProm_ErrOperationNotImplemented(t *testing.T) {
	s := NewProm("foo", "bar")
	s.Register(Counter, "one", "test")
	err := s.Register(Counter, "one", "test")
	if err != nil {
		t.Errorf("TestProm_ErrOperationNotImplemented() error = %v", err)
	}
	err = s.Register(0, "onee", "test")
	if err != ErrOperationNotImplemented {
		t.Errorf("TestProm_ErrOperationNotImplemented() error = %v", err)
	}

	err = s.Dec("one")
	if err != ErrOperationNotImplemented {
		t.Errorf("TestProm_ErrOperationNotImplemented() error = %v", err)
	}
	err = s.Set("one")
	if err != ErrOperationNotImplemented {
		t.Errorf("TestProm_ErrOperationNotImplemented() error = %v", err)
	}

	s.metric["t"] = metric{}
	err = s.Add("t")
	if err != ErrOperationNotImplemented {
		t.Errorf("TestProm_ErrOperationNotImplemented() error = %v", err)
	}
	err = s.Inc("t")
	if err != ErrOperationNotImplemented {
		t.Errorf("TestProm_ErrOperationNotImplemented() error = %v", err)
	}

}

func TestProm_ErrCastType(t *testing.T) {
	s := NewProm("foo", "bar")

	s.metric["1"] = metric{kind: Gauge}
	err := s.Dec("1")
	if err != ErrCastType {
		t.Errorf("TestProm_ErrCastType() error = %v", err)
	}
	err = s.Inc("1")
	if err != ErrCastType {
		t.Errorf("TestProm_ErrCastType() error = %v", err)
	}
	err = s.Set("1")
	if err != ErrCastType {
		t.Errorf("TestProm_ErrCastType() error = %v", err)
	}
	err = s.Add("1")
	if err != ErrCastType {
		t.Errorf("TestProm_ErrCastType() error = %v", err)
	}

	s.metric["1"] = metric{kind: GaugeVec}
	err = s.Dec("1")
	if err != ErrCastType {
		t.Errorf("TestProm_ErrCastType() error = %v", err)
	}
	err = s.Inc("1")
	if err != ErrCastType {
		t.Errorf("TestProm_ErrCastType() error = %v", err)
	}
	err = s.Set("1")
	if err != ErrCastType {
		t.Errorf("TestProm_ErrCastType() error = %v", err)
	}
	err = s.Add("1")
	if err != ErrCastType {
		t.Errorf("TestProm_ErrCastType() error = %v", err)
	}

	s.metric["1"] = metric{kind: Counter}
	err = s.Inc("1")
	if err != ErrCastType {
		t.Errorf("TestProm_ErrCastType() error = %v", err)
	}
	err = s.Add("1")
	if err != ErrCastType {
		t.Errorf("TestProm_ErrCastType() error = %v", err)
	}

	s.metric["1"] = metric{kind: CounterVec}
	err = s.Inc("1")
	if err != ErrCastType {
		t.Errorf("TestProm_ErrCastType() error = %v", err)
	}
	err = s.Add("1")
	if err != ErrCastType {
		t.Errorf("TestProm_ErrCastType() error = %v", err)
	}

	s.Register(Counter, "err1", "test")
	s.Register(Gauge, "err2", "test")
	err = s.Add("err1", "one")
	if err != ErrCastType {
		t.Errorf("TestProm_ErrCastType() error = %v", err)
	}
	err = s.Add("err2", "two")
	if err != ErrCastType {
		t.Errorf("TestProm_ErrCastType() error = %v", err)
	}
	err = s.Set("err2", "two")
	if err != ErrCastType {
		t.Errorf("TestProm_ErrCastType() error = %v", err)
	}
}
