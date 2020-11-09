package rtmetric

import (
	"time"

	"github.com/balugcath/pgpf/internal/pkg/config"
	"github.com/balugcath/pgpf/pkg/promwrap"
	log "github.com/sirupsen/logrus"
)

const (
	pgpfStatusHostsMetricName = "pgpf_status_hosts"
	pgpfStatusHostsMetricHelp = "pgpf_status_hosts"
)

const (
	hostDead = iota
	hostStandby
	hostMaster
)

type transporter interface {
	HostStatus(string) (bool, float64, error)
}

// HostStatus ...
type HostStatus struct {
	metric promwrap.Interface
	*config.Config
	transporter
}

// NewHostStatus ...
func NewHostStatus(config *config.Config, metric promwrap.Interface, transporter transporter) *HostStatus {
	s := HostStatus{Config: config, metric: metric, transporter: transporter}
	s.metric.Register(promwrap.GaugeVec, pgpfStatusHostsMetricName, pgpfStatusHostsMetricHelp, []string{"node"}...)

	return &s
}

// Start ...
func (s *HostStatus) Start() {
	go func() {
		for {
			for k, v := range s.Config.Servers {
				log.Debugf("check host %s", k)
				if !v.Use {
					log.Debugf("skip host %s", k)
					continue
				}
				isRecovery, _, err := s.transporter.HostStatus(v.PgConn)
				if err != nil {
					s.metric.Set(pgpfStatusHostsMetricName, []interface{}{k, float64(hostDead)}...)
					log.Errorf("check host error %s", err)
					continue
				}
				if !isRecovery {
					s.metric.Set(pgpfStatusHostsMetricName, []interface{}{k, float64(hostMaster)}...)
					continue
				}
				s.metric.Set(pgpfStatusHostsMetricName, []interface{}{k, float64(hostStandby)}...)
			}
			<-time.After(time.Second * time.Duration(s.Config.TimeoutHostStatus))
		}
	}()
}
