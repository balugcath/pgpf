package failover

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/balugcath/pgpf/internal/pkg/config"
	"github.com/balugcath/pgpf/internal/pkg/metric"
	"github.com/balugcath/pgpf/internal/pkg/proxy"
	"github.com/balugcath/pgpf/internal/pkg/transport"
)

var (
	// ErrCheckHostFail ...
	ErrCheckHostFail = errors.New("check host failed")
	// ErrTerminate ...
	ErrTerminate = errors.New("terminate")
	// ErrNoMasterFound ...
	ErrNoMasterFound = errors.New("no master found")
	// ErrNoStandbyFound ...
	ErrNoStandbyFound = errors.New("no standby found")
)

// Failover ...
type Failover struct {
	*config.Config
	transport.Transporter
	metric.Metricer
}

// NewFailover ...
func NewFailover(cfg *config.Config) *Failover {
	s := &Failover{
		Config: cfg,
	}
	return s
}

// Start ...
func (s *Failover) Start(doneCtx context.Context) {
	mtrc := metric.NewMetric(s.Config)
	go func() {
		if s.Config.PrometheusListenPort == "" {
			return
		}
		log.Fatalln(mtrc.Start())
	}()

	s.Transporter = &transport.PG{}
	s.Metricer = mtrc

	go func() {
		log.Fatalln(s.startFailover(doneCtx))
	}()

	go func() {
		s.startChkSatusHosts(doneCtx, &transport.PG{})
	}()
}

func (s *Failover) startFailover(doneCtx context.Context) error {
	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		s.Config.Lock()

		terminateCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		masterName, err := s.makeMaster()
		if err != nil {
			return err
		}
		log.Printf("use master %s\n", masterName)

		masterProxy, err := proxy.NewProxy(s.Config, s.Metricer).Listen(s.Config.Listen)
		if err != nil {
			return err
		}

		for s.Config.ShardListen != "" {
			standbyName, err := s.findStandby()
			if err != nil {
				log.Println(err)
				break
			}
			shardProxy, err := proxy.NewProxy(s.Config, s.Metricer).Listen(s.Config.ShardListen)
			if err != nil {
				log.Println(err)
				break
			}
			log.Printf("use standby %s\n", standbyName)
			wg.Add(1)
			go func() {
				defer wg.Done()
				log.Printf("standby proxy %s exit %s\n", standbyName, shardProxy.Serve(terminateCtx, standbyName))
			}()
			break
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancel()
			log.Printf("check master %s exit %s\n", masterName, s.checkMaster(terminateCtx, s.Config.Servers[masterName].PgConn))
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf("proxy %s exit %s\n", masterName, masterProxy.Serve(terminateCtx, masterName))
		}()

		s.Config.Unlock()

		select {
		case <-doneCtx.Done():
			return ErrTerminate
		case <-terminateCtx.Done():
			wg.Wait()
			s.Config.Servers[masterName].Use = false
			s.Config.Save()
		}
	}
}

func (s *Failover) makeMaster() (string, error) {
	for k, v := range s.Config.Servers {
		if !v.Use {
			continue
		}
		isRecovery, _, err := s.Transporter.HostStatus(v.PgConn)
		if err != nil {
			log.Println(err)
			continue
		}
		if !isRecovery {
			return k, nil
		}
	}
	for k, v := range s.Config.Servers {
		if !v.Use {
			continue
		}
		log.Printf("try promote %s\n", k)
		isRecovery, ver, err := s.Transporter.HostStatus(v.PgConn)
		if err != nil {
			log.Println(err)
			continue
		}
		if !isRecovery {
			continue
		}
		if ver >= s.Config.MinVerSQLPromote {
			err = s.Transporter.Promote(v.PgConn)
		} else {
			err = execCommand(v.Command)
		}
		if err != nil {
			log.Println(err)
			continue
		}
		for y := 0; y <= s.Config.TimeoutWaitPromote; y++ {
			time.Sleep(time.Second)
			log.Printf("wait promote %s %d\n", k, y+1)
			isRecovery, _, err := s.Transporter.HostStatus(v.PgConn)
			if err != nil {
				log.Println(err)
				break
			}
			if !isRecovery {
				go func() {
					for _, cmd := range s.makePostPromoteCmds(k, v.Host, v.Port) {
						if err := execCommand(cmd); err != nil {
							log.Println(err)
							continue
						}
					}
				}()
				return k, nil
			}
		}
	}
	return "", ErrNoMasterFound
}

func (s *Failover) findStandby() (string, error) {
	for k, v := range s.Config.Servers {
		if !v.Use {
			continue
		}
		isRecovery, _, err := s.Transporter.HostStatus(v.PgConn)
		if err != nil {
			log.Println(err)
			continue
		}
		if isRecovery {
			return k, nil
		}
	}
	return "", ErrNoStandbyFound
}

func (s *Failover) makePostPromoteCmds(newMaster, host, port string) []string {
	out := make([]string, 0)
	for k, v := range s.Config.Servers {
		if !v.Use || k == newMaster || v.PostPromoteCommand == "" {
			continue
		}

		t, err := template.New("t").Parse(v.PostPromoteCommand)
		if err != nil {
			log.Printf("error parse template %s %s\n", v.PostPromoteCommand, err)
		}

		var str strings.Builder
		if err := t.Execute(&str,
			struct {
				Host   string
				Port   string
				PgUser string
			}{
				host,
				port,
				s.Config.PgUser,
			}); err != nil {
			log.Printf("error build template %s %s\n", v.PostPromoteCommand, err)
		}
		out = append(out, str.String())
	}
	return out
}

func (s *Failover) checkMaster(doneCtx context.Context, pgConn string) error {
	if err := s.Transporter.Open(pgConn); err != nil {
		return err
	}
	defer s.Transporter.Close()

	failureCount := 0
	for {
		_, err := s.Transporter.IsRecovery()
		if err != nil {
			failureCount++
		} else {
			failureCount = 0
		}

		select {
		case <-doneCtx.Done():
			return ErrTerminate
		case <-time.After(time.Second * time.Duration(s.Config.TimeoutCheckMaster)):
			if failureCount > s.Config.FailoverTimeout {
				return ErrCheckHostFail
			}
		}
	}
}

func execCommand(cmd string) error {
	c := strings.Split(cmd, " ")
	out, err := exec.Command(c[0], c[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w %s", err, string(out))
	}
	return nil
}

const (
	hostDead = iota
	hostStandby
	hostMaster
)

func (s *Failover) startChkSatusHosts(doneCtx context.Context, tr transport.Transporter) {
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
			isRecovery, _, err := tr.HostStatus(v.PgConn)
			if err != nil {
				s.Metricer.StatusHost(k, hostDead)
				continue
			}
			if !isRecovery {
				s.Metricer.StatusHost(k, hostMaster)
				continue
			}
			s.Metricer.StatusHost(k, hostStandby)
		}
	}
}
