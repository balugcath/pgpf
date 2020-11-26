package config

import (
	"context"
	"errors"
	"io/ioutil"
	"net"
	"strings"
	"sync"
	"text/template"
	"time"

	"go.etcd.io/etcd/clientv3"
	"gopkg.in/yaml.v2"
)

const (
	pgConnTemplate = "user={{.PGUser}} password={{.PGPasswd}} dbname=postgres host={{.Host}} port={{.Port}} sslmode=disable"
)

var connTimeout = time.Second * 8

// Server ...
type Server struct {
	Address            string `yaml:"address"`
	Command            string `yaml:"command,omitempty"`
	PostPromoteCommand string `yaml:"post_promote_command,omitempty"`
	PgConn             string `yaml:"-"`
	Use                bool   `yaml:"use"`
	Host               string `yaml:"-"`
	Port               string `yaml:"-"`
}

// Config ...
type Config struct {
	*sync.Mutex `yaml:"-"`

	etcdAddress            string
	etcdKey                string
	configFile             string
	MinVerSQLPromote       float64            `yaml:"-"`
	TimeoutWaitPromoteSec  int                `yaml:"-"`
	HostStatusIntervalSec  int                `yaml:"host_status_interval"`
	CheckMasterIntervalSec int                `yaml:"check_master_interval"`
	FailoverTimeoutSec     int                `yaml:"failover_timeout"`
	TimeoutMasterDialSec   int                `yaml:"master_dial_timeout"`
	Listen                 string             `yaml:"listen"`
	ShardListen            string             `yaml:"shard_listen"`
	PgUser                 string             `yaml:"pg_user"`
	PgPasswd               string             `yaml:"pg_passwd"`
	PrometheusListenPort   string             `yaml:"prometheus_listen_port"`
	Servers                map[string]*Server `yaml:"servers,omitempty"`
}

var config = Config{
	Mutex:            &sync.Mutex{},
	MinVerSQLPromote: 12,
	// https://www.postgresql.org/docs/12/functions-admin.html#FUNCTIONS-RECOVERY-CONTROL
	TimeoutWaitPromoteSec:  60,
	HostStatusIntervalSec:  10,
	TimeoutMasterDialSec:   1,
	CheckMasterIntervalSec: 1,
	FailoverTimeoutSec:     8,
}

// NewConfig ...
func NewConfig(configFile, etcdAddress, etcdKey string) (*Config, error) {
	s := config
	s.configFile = configFile
	s.etcdAddress = etcdAddress
	s.etcdKey = etcdKey

	if err := s.load(); err != nil {
		return nil, err
	}
	if err := s.checkConfig(); err != nil {
		return nil, err
	}
	if err := s.prepeareConfig(pgConnTemplate); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *Config) checkConfig() error {
	if s.PgUser == "" {
		return errors.New("pg user not defined")
	}
	if s.PgPasswd == "" {
		return errors.New("pg password not defined")
	}
	if s.Listen == "" {
		return errors.New("listen port not defined")
	}
	if s.Listen == s.ShardListen {
		return errors.New("listen port equal shard port")
	}
	if s.CheckMasterIntervalSec == 0 {
		return errors.New("check master interval zero")
	}
	if s.FailoverTimeoutSec == 0 {
		return errors.New("failover timeout interval zero")
	}
	if s.FailoverTimeoutSec < s.CheckMasterIntervalSec {
		return errors.New("failover timeout interval must be a greater than check master interval")
	}
	if len(s.Servers) == 0 {
		return errors.New("server hot defined")
	}
	return nil
}

func (s *Config) prepeareConfig(pgConnTemplate string) error {
	t, err := template.New("t").Parse(pgConnTemplate)
	if err != nil {
		return err
	}

	for k, v := range s.Servers {
		v.Host, v.Port, err = net.SplitHostPort(v.Address)
		if err != nil {
			return err
		}

		var str strings.Builder
		if err := t.Execute(&str,
			struct {
				PGUser   string
				PGPasswd string
				Host     string
				Port     string
			}{
				s.PgUser,
				s.PgPasswd,
				v.Host,
				v.Port,
			}); err != nil {
			return err
		}
		v.PgConn = str.String()
		v.PostPromoteCommand = strings.TrimSpace(s.Servers[k].PostPromoteCommand)
		s.Servers[k] = v
	}

	return nil
}

// Save ...
func (s *Config) Save() error {
	b, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	switch {
	case s.configFile != "":
		if err := ioutil.WriteFile(s.configFile, b, 0644); err != nil {
			return err
		}
	case s.etcdAddress != "" && s.etcdKey != "":
		cli, err := clientv3.New(clientv3.Config{
			Endpoints: []string{s.etcdAddress},
		})
		defer cli.Close()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), connTimeout)
		defer cancel()
		_, err = cli.Put(ctx, s.etcdKey, string(b))
		if err != nil {
			return err
		}
	default:
		return errors.New("config not defined")
	}
	return nil
}

func (s *Config) load() error {

	var (
		b   []byte
		err error
	)

	switch {
	case s.configFile != "":
		b, err = ioutil.ReadFile(s.configFile)
		if err != nil {
			return err
		}

	case s.etcdAddress != "" && s.etcdKey != "":
		cli, err := clientv3.New(clientv3.Config{
			Endpoints: []string{s.etcdAddress},
		})
		if err != nil {
			return err
		}
		defer cli.Close()

		ctx, cancel := context.WithTimeout(context.Background(), connTimeout)
		defer cancel()
		resp, err := cli.Get(ctx, s.etcdKey)
		if err != nil {
			return err
		}

		if len(resp.Kvs) == 0 {
			return errors.New("error reading config")
		}
		b = resp.Kvs[0].Value

	default:
		return errors.New("config not defined")
	}

	if err := yaml.Unmarshal(b, s); err != nil {
		return err
	}
	return nil
}
