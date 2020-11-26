package rtmetric

import (
	"time"

	"github.com/balugcath/pgpf/internal/pkg/config"
	"github.com/balugcath/pgpf/pkg/promwrap"
	log "github.com/sirupsen/logrus"
)

const (
	pgpfStatusHostsMetricName = "pgpf_status_hosts"
	pgpfStatusHostsMetricHelp = "status hosts"
)

const (
	hostDead = iota
	hostStandby
	hostMaster
)

type transporter interface {
	IsRecovery() (bool, error)
	Open(string) error
	Close()
}

// HostStatus ...
type HostStatus struct {
	metric promwrap.Interface
	*config.Config
	transporter
}

var timeUnit = time.Second

// NewHostStatus ...
func NewHostStatus(config *config.Config, metric promwrap.Interface, transporter transporter) *HostStatus {
	s := HostStatus{Config: config, metric: metric, transporter: transporter}
	s.metric.Register(promwrap.GaugeVec, pgpfStatusHostsMetricName, pgpfStatusHostsMetricHelp, []string{"node"}...)

	return &s
}

// Start ...
func (s *HostStatus) Start() {
	if s.Config.HostStatusIntervalSec == 0 {
		return
	}
	go func() {
		for {
			for k, v := range s.Config.Servers {
				log.Debugf("rtmetric.Start() host %s check", k)
				if !v.Use {
					log.Debugf("rtmetric.Start() host %s skip", k)
					continue
				}
				if err := s.transporter.Open(v.PgConn); err != nil {
					s.metric.Set(pgpfStatusHostsMetricName, []interface{}{k, float64(hostDead)}...)
					log.Errorf("rtmetric.Start() host %s %s", k, err)
					continue
				}
				isRecovery, err := s.transporter.IsRecovery()
				if err != nil {
					log.Errorf("rtmetric.Start() host %s %s", k, err)
					s.metric.Set(pgpfStatusHostsMetricName, []interface{}{k, float64(hostDead)}...)
					s.transporter.Close()
					continue
				}
				if !isRecovery {
					s.metric.Set(pgpfStatusHostsMetricName, []interface{}{k, float64(hostMaster)}...)
					s.transporter.Close()
					continue
				}
				s.metric.Set(pgpfStatusHostsMetricName, []interface{}{k, float64(hostStandby)}...)
				s.transporter.Close()
			}
			<-time.After(time.Duration(s.Config.HostStatusIntervalSec) * timeUnit)
		}
	}()
}
