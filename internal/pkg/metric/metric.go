package metric

import (
	"context"
	"net/http"
	"time"

	"github.com/balugcath/pgpf/internal/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

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

type transporter interface {
	HostStatus(string) (bool, float64, error)
}

// Metric ...
type Metric struct {
	*config.Config
	transporter
	clientConn    *prometheus.GaugeVec
	transferBytes *prometheus.CounterVec
	statusHosts   *prometheus.GaugeVec
}

// NewMetric ...
func NewMetric(cfg *config.Config, tranporter transporter) *Metric {
	s := &Metric{
		Config:      cfg,
		transporter: tranporter,
	}
	return s
}

// Start ...
func (s *Metric) Start(doneCtx context.Context) *Metric {
	if s.Config.PrometheusListenPort == "" {
		return s
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
				log.Debugf("check host %s", k)
				if !v.Use {
					log.Debugf("skip host %s", k)
					continue
				}
				isRecovery, _, err := s.transporter.HostStatus(v.PgConn)
				if err != nil {
					s.statusHosts.WithLabelValues(k).Set(float64(hostDead))
					log.Errorf("check host error %s", err)
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
	if s.clientConn != nil {
		s.clientConn.WithLabelValues(host).Inc()
	}
}

// ClientConnDec ...
func (s *Metric) ClientConnDec(host string) {
	if s.clientConn != nil {
		s.clientConn.WithLabelValues(host).Dec()
	}
}

// TransferBytes ...
func (s *Metric) TransferBytes(host, t string, n int) {
	if s.transferBytes != nil {
		s.transferBytes.WithLabelValues(host, t).Add(float64(n))
	}
}
