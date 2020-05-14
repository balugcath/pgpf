package metric

import (
	"net/http"

	"github.com/balugcath/pgpf/internal/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metricer ...
type Metricer interface {
	ClientConnInc(string)
	ClientConnDec(string)
	TransferBytes(string, string, int)
	StatusHost(string, int)
}

const (
	pgpfClientConnections = "pgpf_client_connections"
	pgpfTransferBytes     = "pgpf_transfer_bytes"
	pgpfStatusHosts       = "pgpf_status_hosts"
)

// Metric ...
type Metric struct {
	*config.Config
	clientConn    *prometheus.GaugeVec
	transferBytes *prometheus.CounterVec
	statusHosts   *prometheus.GaugeVec
}

// NewMetric ...
func NewMetric(config *config.Config) *Metric {
	s := &Metric{Config: config}

	s.clientConn = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: pgpfClientConnections,
			Help: "How many client connected, partition by host",
		},
		[]string{"host"},
	)
	prometheus.MustRegister(s.clientConn)

	s.transferBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: pgpfTransferBytes,
			Help: "How many bytes transferred, partition by host",
		},
		[]string{"host", "type"},
	)
	prometheus.MustRegister(s.transferBytes)

	s.statusHosts = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: pgpfStatusHosts,
			Help: "Status hosts",
		},
		[]string{"host"},
	)
	prometheus.MustRegister(s.statusHosts)

	return s
}

// Start ...
func (s *Metric) Start() error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(s.PrometheusListenPort, nil)
}

// ClientConnInc ...
func (s *Metric) ClientConnInc(host string) {
	s.clientConn.WithLabelValues(host).Inc()
}

// ClientConnDec ...
func (s *Metric) ClientConnDec(host string) {
	s.clientConn.WithLabelValues(host).Dec()
}

// TransferBytes ...
func (s *Metric) TransferBytes(host, t string, n int) {
	s.transferBytes.WithLabelValues(host, t).Add(float64(n))
}

// StatusHost ...
func (s *Metric) StatusHost(host string, n int) {
	s.statusHosts.WithLabelValues(host).Set(float64(n))
}
