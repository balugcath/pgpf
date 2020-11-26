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
	"github.com/balugcath/pgpf/pkg/promwrap"
	log "github.com/sirupsen/logrus"
)

var (
	// ErrCheckHostFail ...
	ErrCheckHostFail = errors.New("check host failed")
	// ErrTerminate ...
	ErrTerminate = errors.New("terminate")
	// ErrServerNotFound ...
	ErrServerNotFound = errors.New("server not found")
)

type transporter interface {
	Open(string) error
	Close()
	HostVersion() (float64, error)
	IsRecovery() (bool, error)
	Promote() error
	ChangePrimaryConn(string, string, string) error
}

var timeUnit = time.Second

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
func (s *Failover) Start(doneCtx context.Context) error {
	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		s.Config.Lock()

		terminateCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		masterName, err := s.findServer(false)
		if err != nil {
			masterName, err = s.makeMaster()
			if err != nil {
				return err
			}
		}

		masterProxy, err := proxy.NewProxy(s.Config, s.metric).Listen(s.Config.Listen)
		if err != nil {
			return err
		}

		for s.Config.ShardListen != "" {
			standbyName, err := s.findServer(true)
			if err != nil {
				log.Errorf("failover.Start() %s", err)
				break
			}
			shardProxy, err := proxy.NewProxy(s.Config, s.metric).Listen(s.Config.ShardListen)
			if err != nil {
				log.Errorf("failover.Start() %s", err)
				break
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				log.Debugf("failover.Start() host %s use as standby", standbyName)
				log.Errorf("failover.Start() host %s %s", standbyName, shardProxy.Serve(terminateCtx, standbyName))
			}()
			break
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancel()
			log.Errorf("failover.Start() host %s %s", masterName, s.checkMaster(terminateCtx, s.Config.Servers[masterName].PgConn))
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Debugf("failover.Start() host %s use as master", masterName)
			log.Errorf("failover.Start() host %s %s", masterName, masterProxy.Serve(terminateCtx, masterName))
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
	log.Debugln("failover.makeMaster() start")

	for k, v := range s.Config.Servers {
		if !v.Use {
			log.Debugf("failover.makeMaster() host %s skip", k)
			continue
		}
		if err := s.transporter.Open(v.PgConn); err != nil {
			log.Errorf("failover.makeMaster() %s", err)
			continue
		}
		isRecovery, err := s.transporter.IsRecovery()
		if err != nil {
			log.Errorf("failover.makeMaster() %s", err)
			s.transporter.Close()
			continue
		}
		if !isRecovery {
			s.transporter.Close()
			continue
		}
		ver, err := s.transporter.HostVersion()
		if err != nil {
			log.Errorf("failover.makeMaster() %s", err)
			s.transporter.Close()
			continue
		}
		log.Debugf("failover.makeMaster() host %s try promote", k)
		if ver >= s.Config.MinVerSQLPromote {
			err = s.transporter.Promote()
		} else {
			err = execCommand(v.Command)
		}
		if err != nil {
			log.Errorf("failover.makeMaster() %s", err)
			s.transporter.Close()
			continue
		}
		for y := 0; y <= s.Config.TimeoutWaitPromoteSec; y++ {
			time.Sleep(time.Second)
			isRecovery, err := s.transporter.IsRecovery()
			if err != nil {
				log.Errorf("failover.makeMaster() %s", err)
				s.transporter.Close()
				break
			}
			if !isRecovery {
				go func() {
					// check on 13 ver
					// if ver >= s.Config.MinVerSQLPromote {
					// s.makePostPromoteExec(k, v.Host, v.Port)
					// } else {
					// s.makePostPromoteCmd(k, v.Host, v.Port)
					// }
					s.makePostPromoteCmd(k, v.Host, v.Port)
				}()
				log.Debugf("failover.makeMaster() host %s promoted", k)
				return k, nil
			}
		}
	}
	return "", ErrServerNotFound
}

func (s *Failover) findServer(isRecovery bool) (string, error) {
	log.Debugf("failover.findServer() recovery %+v start", isRecovery)

	for k, v := range s.Config.Servers {
		if !v.Use {
			continue
		}
		if err := s.transporter.Open(v.PgConn); err != nil {
			log.Errorf("failover.findServer() %s", err)
			continue
		}
		ret, err := s.transporter.IsRecovery()
		s.transporter.Close()
		if err != nil {
			log.Errorf("failover.findServer() %s", err)
			continue
		}
		if ret == isRecovery {
			log.Debugf("failover.findServer() found host %s recovery %+v", k, isRecovery)
			return k, nil
		}
	}
	log.Debugf("failover.findServer() recovery %+v %s", isRecovery, ErrServerNotFound)
	return "", ErrServerNotFound
}

func (s *Failover) makePostPromoteExec(newMaster, host, port string) (ret []string) {
	for k, v := range s.Config.Servers {
		if !v.Use || k == newMaster {
			continue
		}
		if err := s.transporter.Open(v.PgConn); err != nil {
			log.Errorf("failover.makePostPromoteExec() %s", err)
			continue
		}
		if err := s.transporter.ChangePrimaryConn(v.Host, s.Config.PgUser, v.Port); err != nil {
			log.Errorf("failover.makePostPromoteExec() %s", err)
		}
		s.transporter.Close()
		ret = append(ret, k)
	}
	return ret
}

func (s *Failover) makePostPromoteCmd(newMaster, host, port string) (ret []string) {
	for k, v := range s.Config.Servers {
		if !v.Use || k == newMaster || v.PostPromoteCommand == "" {
			continue
		}

		t, err := template.New("t").Parse(v.PostPromoteCommand)
		if err != nil {
			log.Errorf("failover.makePostPromoteCmd() %s %s", v.PostPromoteCommand, err)
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
			log.Errorf("failover.makePostPromoteCmd() %s %s", v.PostPromoteCommand, err)
			continue
		}

		ret = append(ret, str.String())

		if err := execCommand(str.String()); err != nil {
			log.Errorf("failover.makePostPromoteCmd() %s %s", str.String(), err)
		}
	}
	return ret
}

func (s *Failover) checkMaster(doneCtx context.Context, pgConn string) error {
	log.Debugf("failover.checkMaster() start")
	defer log.Debugf("failover.checkMaster() exit")

	if err := s.transporter.Open(pgConn); err != nil {
		return err
	}
	defer s.transporter.Close()

	leftTime := time.Duration(s.Config.FailoverTimeoutSec) * timeUnit
	for {
		_, err := s.transporter.IsRecovery()
		if err != nil {
			leftTime -= time.Duration(s.Config.CheckMasterIntervalSec) * timeUnit
			log.Errorf("failover.checkMaster() %s", err)
		} else {
			leftTime = time.Duration(s.Config.FailoverTimeoutSec) * timeUnit
		}

		select {
		case <-doneCtx.Done():
			return ErrTerminate
		case <-time.After(leftTime):
			return ErrCheckHostFail
		case <-time.After(time.Duration(s.Config.CheckMasterIntervalSec) * timeUnit):
		}
	}
}

func execCommand(cmd string) error {
	c := strings.Split(cmd, " ")
	out, err := exec.Command(c[0], c[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failover.execCommand() %s %s", string(out), err)
	}
	return nil
}
