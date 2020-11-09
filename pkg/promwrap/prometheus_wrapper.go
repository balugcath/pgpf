package promwrap

import (
	"errors"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Interface ...
type Interface interface {
	Register(kind int, name, help string, opts ...string) error
	Add(name string, opts ...interface{}) error
	Set(name string, opts ...interface{}) error
	Inc(name string, opts ...interface{}) error
	Dec(name string, opts ...interface{}) error
}

const (
	_ = iota
	// Gauge ...
	Gauge
	// GaugeVec ...
	GaugeVec
	// Counter ...
	Counter
	// CounterVec ...
	CounterVec
)

var (
	// ErrCastType ...
	ErrCastType = errors.New("cast type")
	// ErrOperationNotImplemented ...
	ErrOperationNotImplemented = errors.New("operation not implemented")
	// ErrMetricNotExist ...
	ErrMetricNotExist = errors.New("metric not exist")
)

type metric struct {
	kind int
	val  interface{}
}

// Prom ...
type Prom struct {
	port   string
	path   string
	metric map[string]metric
}

// NewProm ...
func NewProm(port, path string) *Prom {
	s := &Prom{
		port:   port,
		path:   path,
		metric: make(map[string]metric),
	}
	return s
}

// Start ...
func (s *Prom) Start() error {
	http.Handle(s.path, promhttp.Handler())
	return http.ListenAndServe(s.port, nil)
}

// Register ...
func (s *Prom) Register(kind int, name, help string, opts ...string) error {
	if _, ok := s.metric[name]; ok {
		return nil
	}
	switch kind {
	default:
		return ErrOperationNotImplemented
	case Gauge:
		m := prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: name,
				Help: help,
			},
		)
		prometheus.MustRegister(m)
		s.metric[name] = metric{kind: kind, val: m}

	case GaugeVec:
		m := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: name,
				Help: help,
			},
			opts,
		)
		prometheus.MustRegister(m)
		s.metric[name] = metric{kind: kind, val: m}

	case CounterVec:
		m := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: name,
				Help: help,
			},
			opts,
		)
		prometheus.MustRegister(m)
		s.metric[name] = metric{kind: kind, val: m}

	case Counter:
		m := prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: name,
				Help: help,
			},
		)
		prometheus.MustRegister(m)
		s.metric[name] = metric{kind: kind, val: m}
	}
	return nil
}

// Add ...
// Gauge, Counter: opts[0] = float64
// GaugeVec, CounterVec: []opts = float64 and strings
func (s *Prom) Add(name string, opts ...interface{}) error {
	defer func() {
		recover()
	}()

	v, ok := s.metric[name]
	if !ok {
		return ErrMetricNotExist
	}

	switch v.kind {
	default:
		return ErrOperationNotImplemented

	case Gauge:
		m, ok := v.val.(prometheus.Gauge)
		if !ok {
			return ErrCastType
		}
		v, ok := opts[0].(float64)
		if !ok {
			return ErrCastType
		}
		m.Add(v)

	case Counter:
		m, ok := v.val.(prometheus.Counter)
		if !ok {
			return ErrCastType
		}
		v, ok := opts[0].(float64)
		if !ok {
			return ErrCastType
		}
		m.Add(v)

	case CounterVec:
		m, ok := v.val.(*prometheus.CounterVec)
		if !ok {
			return ErrCastType
		}
		var (
			newOpts = []string{}
			val     float64
		)

		for i := range opts {
			switch v := opts[i].(type) {
			default:
				continue
			case float64:
				val = v
			case string:
				newOpts = append(newOpts, v)
			}
		}
		m.WithLabelValues(newOpts...).Add(val)

	case GaugeVec:
		m, ok := v.val.(*prometheus.GaugeVec)
		if !ok {
			return ErrCastType
		}
		var (
			newOpts = []string{}
			val     float64
		)

		for i := range opts {
			switch v := opts[i].(type) {
			default:
				continue
			case float64:
				val = v
			case string:
				newOpts = append(newOpts, v)
			}
		}
		m.WithLabelValues(newOpts...).Add(val)
	}

	return nil
}

// Set ...
// Gauge: opts[0] = float64
// GaugeVec: []opts float64 and strings
func (s *Prom) Set(name string, opts ...interface{}) error {
	defer func() {
		recover()
	}()

	v, ok := s.metric[name]
	if !ok {
		return ErrMetricNotExist
	}

	switch v.kind {
	default:
		return ErrOperationNotImplemented

	case Gauge:
		m, ok := v.val.(prometheus.Gauge)
		if !ok {
			return ErrCastType
		}
		v, ok := opts[0].(float64)
		if !ok {
			return ErrCastType
		}
		m.Set(v)

	case GaugeVec:
		m, ok := v.val.(*prometheus.GaugeVec)
		if !ok {
			return ErrCastType
		}
		var (
			newOpts = []string{}
			val     float64
		)

		for i := range opts {
			switch v := opts[i].(type) {
			default:
				continue
			case float64:
				val = v
			case string:
				newOpts = append(newOpts, v)
			}
		}
		m.WithLabelValues(newOpts...).Set(val)
	}

	return nil
}

// Inc ...
// Gauge, Counter: []opts = nil
// GaugeVec, CounterVec: []opts strings
func (s *Prom) Inc(name string, opts ...interface{}) error {
	defer func() {
		recover()
	}()

	v, ok := s.metric[name]
	if !ok {
		return ErrMetricNotExist
	}

	switch v.kind {
	default:
		return ErrOperationNotImplemented

	case Gauge:
		m, ok := v.val.(prometheus.Gauge)
		if !ok {
			return ErrCastType
		}
		m.Inc()

	case Counter:
		m, ok := v.val.(prometheus.Counter)
		if !ok {
			return ErrCastType
		}
		m.Inc()

	case CounterVec:
		m, ok := v.val.(*prometheus.CounterVec)
		if !ok {
			return ErrCastType
		}
		var (
			newOpts = []string{}
		)

		for i := range opts {
			switch v := opts[i].(type) {
			default:
				continue
			case string:
				newOpts = append(newOpts, v)
			}
		}
		m.WithLabelValues(newOpts...).Inc()

	case GaugeVec:
		m, ok := v.val.(*prometheus.GaugeVec)
		if !ok {
			return ErrCastType
		}
		var (
			newOpts = []string{}
		)

		for i := range opts {
			switch v := opts[i].(type) {
			default:
				continue
			case string:
				newOpts = append(newOpts, v)
			}
		}
		m.WithLabelValues(newOpts...).Inc()
	}

	return nil
}

// Dec ...
// Gauge: []opts = nil
// GaugeVec: []opts strings
func (s *Prom) Dec(name string, opts ...interface{}) error {
	defer func() {
		recover()
	}()

	v, ok := s.metric[name]
	if !ok {
		return ErrMetricNotExist
	}

	switch v.kind {
	default:
		return ErrOperationNotImplemented

	case Gauge:
		m, ok := v.val.(prometheus.Gauge)
		if !ok {
			return ErrCastType
		}
		m.Dec()

	case GaugeVec:
		m, ok := v.val.(*prometheus.GaugeVec)
		if !ok {
			return ErrCastType
		}
		var (
			newOpts = []string{}
		)

		for i := range opts {
			switch v := opts[i].(type) {
			default:
				continue
			case string:
				newOpts = append(newOpts, v)
			}
		}
		m.WithLabelValues(newOpts...).Dec()
	}

	return nil
}
