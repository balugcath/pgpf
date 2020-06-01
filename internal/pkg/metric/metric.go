package metric

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/balugcath/pgpf/internal/pkg/config"
	"github.com/balugcath/pgpf/internal/pkg/transport"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metricer ...
type Metricer interface {
	ClientConnInc(string)
	ClientConnDec(string)
	TransferBytes(string, string, int)
}

const (
	pgpfClientConnections = "pgpf_client_connections"
	pgpfTransferBytes     = "pgpf_transfer_bytes"
	pgpfStatusHosts       = "pgpf_status_hosts"
)

const (
	hostDead = iota
	hostStandby
	hostMaster
)

// Metric ...
type Metric struct {
	*config.Config
	transport.Transporter
	clientConn    *prometheus.GaugeVec
	transferBytes *prometheus.CounterVec
	statusHosts   *prometheus.GaugeVec
}

// NewMetric ...
func NewMetric(cfg *config.Config, tr transport.Transporter) *Metric {
	s := &Metric{
		Config:      cfg,
		Transporter: tr,
	}

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
func (s *Metric) Start(doneCtx context.Context) *Metric {
	if s.Config.PrometheusListenPort == "" {
		return s
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Fatalln(http.ListenAndServe(s.Config.PrometheusListenPort, nil))
	}()

	go func() {
		for {
			select {
			case <-doneCtx.Done():
				return
			case <-time.After(time.Second * time.Duration(s.Config.TimeoutHostStatus)):
			}
			for k, v := range s.Config.Servers {
				if !v.Use {
					continue
				}
				isRecovery, _, err := s.Transporter.HostStatus(v.PgConn)
				if err != nil {
					s.statusHosts.WithLabelValues(k).Set(float64(hostDead))
					continue
				}
				if !isRecovery {
					s.statusHosts.WithLabelValues(k).Set(float64(hostMaster))
					continue
				}
				s.statusHosts.WithLabelValues(k).Set(float64(hostStandby))
			}
		}
	}()

	return s
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
