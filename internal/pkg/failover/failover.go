package failover

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/balugcath/pgpf/internal/pkg/config"
	"github.com/balugcath/pgpf/internal/pkg/proxy"
	"github.com/balugcath/pgpf/pkg/prom_wrap"
	log "github.com/sirupsen/logrus"
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

type transporter interface {
	HostStatus(string) (bool, float64, error)
	IsRecovery() (bool, error)
	Close()
	Open(string) error
	Promote(string) error
}

// Failover ...
type Failover struct {
	*config.Config
	transporter
	metric promwrap.Interface
}

// NewFailover ...
func NewFailover(cfg *config.Config, transporter transporter, metric promwrap.Interface) *Failover {
	s := &Failover{
		Config:      cfg,
		transporter: transporter,
		metric:      metric,
	}
	return s
}

// Start ...
func (s *Failover) Start(doneCtx context.Context) *Failover {
	go func() {
		log.Fatalln(s.start(doneCtx))
	}()
	return s
}

func (s *Failover) start(doneCtx context.Context) error {
	var wg sync.WaitGroup
	defer wg.Wait()
	log.Debugln("failover start")
	defer log.Debugln("failover exit")

	for {
		s.Config.Lock()

		terminateCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		masterName, err := s.makeMaster()
		if err != nil {
			return err
		}
		log.Infof("use master %s", masterName)

		masterProxy, err := proxy.NewProxy(s.Config, s.metric).Listen(s.Config.Listen)
		if err != nil {
			return err
		}

		if s.Config.ShardListen == "" {
			log.Debugln("shard port not set, skiping shard server")
		}
		for s.Config.ShardListen != "" {
			standbyName, err := s.findStandby()
			if err != nil {
				log.Errorln(err)
				break
			}
			shardProxy, err := proxy.NewProxy(s.Config, s.metric).Listen(s.Config.ShardListen)
			if err != nil {
				log.Errorln(err)
				break
			}
			log.Infof("use standby %s", standbyName)
			wg.Add(1)
			go func() {
				defer wg.Done()
				log.Errorf("standby proxy %s exit %s", standbyName, shardProxy.Serve(terminateCtx, standbyName))
			}()
			break
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancel()
			log.Errorf("check master %s exit %s", masterName, s.checkMaster(terminateCtx, s.Config.Servers[masterName].PgConn))
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Errorf("proxy %s exit %s", masterName, masterProxy.Serve(terminateCtx, masterName))
		}()

		s.Config.Unlock()

		select {
		case <-doneCtx.Done():
			return ErrTerminate
		case <-terminateCtx.Done():
			wg.Wait()
			s.Config.Servers[masterName].Use = false
			log.Infof("set master server %s as dead", masterName)
			s.Config.Save()
		}
	}
}

func (s *Failover) makeMaster() (string, error) {
	for k, v := range s.Config.Servers {
		log.Debugf("check server %s", k)
		if !v.Use {
			log.Debugf("skip %s server", k)
			continue
		}
		isRecovery, _, err := s.transporter.HostStatus(v.PgConn)
		if err != nil {
			log.Errorln(err)
			continue
		}
		if !isRecovery {
			return k, nil
		}
		log.Debugf("server %s in recovery mode", k)
	}

	log.Debugln("no master server found, try search standby server")

	for k, v := range s.Config.Servers {
		log.Debugf("check server %s", k)
		if !v.Use {
			log.Debugf("skip %s server", k)
			continue
		}
		isRecovery, ver, err := s.transporter.HostStatus(v.PgConn)
		if err != nil {
			log.Errorln(err)
			continue
		}
		if !isRecovery {
			log.Debugf("server %s is not in recovery mode", k)
			continue
		}

		log.Debugf("found server %s version %+v", k, ver)
		if ver >= s.Config.MinVerSQLPromote {
			err = s.transporter.Promote(v.PgConn)
		} else {
			err = execCommand(v.Command)
		}
		if err != nil {
			log.Errorf("promote error %s, skiping %s", err, k)
			continue
		}
		for y := 0; y <= s.Config.TimeoutWaitPromote; y++ {
			time.Sleep(time.Second)
			log.Infof("waiting for promote %s %d", k, y+1)
			isRecovery, _, err := s.transporter.HostStatus(v.PgConn)
			if err != nil {
				log.Errorf("promote error %s, skiping %s", err, k)
				break
			}
			if !isRecovery {
				go func() {
					for _, cmd := range s.makePostPromoteCmds(k, v.Host, v.Port) {
						log.Debugf("exec post promote command %s", cmd)
						if err := execCommand(cmd); err != nil {
							log.Errorln(err)
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
		isRecovery, _, err := s.transporter.HostStatus(v.PgConn)
		if err != nil {
			log.Errorln(err)
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
			log.Errorf("error parse template %s %s", v.PostPromoteCommand, err)
			continue
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
			log.Errorf("error execute template %s %s", v.PostPromoteCommand, err)
			continue
		}
		out = append(out, str.String())
	}
	return out
}

func (s *Failover) checkMaster(doneCtx context.Context, pgConn string) error {
	if err := s.transporter.Open(pgConn); err != nil {
		return err
	}
	defer s.transporter.Close()

	failureCount := 0
	for {
		_, err := s.transporter.IsRecovery()
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
