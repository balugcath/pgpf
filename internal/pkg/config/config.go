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

const (
	// https://www.postgresql.org/docs/12/functions-admin.html#FUNCTIONS-RECOVERY-CONTROL
	timeoutWaitPromote = 60
	minVerSQLPromote   = 12
	timeoutHostStatus  = 8
	timeoutCheckMaster = 1
	timeoutMasterDial  = 8
)

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
	*sync.Mutex          `yaml:"-"`
	etcdAddress          string             `yaml:"-"`
	etcdKey              string             `yaml:"-"`
	configFile           string             `yaml:"-"`
	TimeoutWaitPromote   int                `yaml:"-"`
	MinVerSQLPromote     float64            `yaml:"-"`
	TimeoutHostStatus    int                `yaml:"-"`
	TimeoutCheckMaster   int                `yaml:"-"`
	TimeoutMasterDial    int                `yaml:"-"`
	Listen               string             `yaml:"listen"`
	ShardListen          string             `yaml:"shard_listen"`
	FailoverTimeout      int                `yaml:"failover_timeout"`
	PgUser               string             `yaml:"pg_user"`
	PgPasswd             string             `yaml:"pg_passwd"`
	PrometheusListenPort string             `yaml:"prometheus_listen_port"`
	Servers              map[string]*Server `yaml:"servers"`
}

// NewConfig ...
func NewConfig(configFile, etcdAddress, etcdKey string) (*Config, error) {

	var (
		b   []byte
		err error
	)

	switch {
	case configFile != "":
		b, err = ioutil.ReadFile(configFile)
		if err != nil {
			return nil, err
		}

	case etcdAddress != "" && etcdKey != "":
		cli, err := clientv3.New(clientv3.Config{
			Endpoints:   []string{etcdAddress},
			DialTimeout: 2 * time.Second,
		})
		if err != nil {
			return nil, err
		}
		defer cli.Close()

		resp, err := cli.Get(context.Background(), etcdKey)
		if err != nil {
			return nil, err
		}

		if len(resp.Kvs) == 0 {
			return nil, errors.New("error reading config")
		}
		b = resp.Kvs[0].Value

	default:
		return nil, errors.New("config not defined")
	}

	s := &Config{
		Mutex:              &sync.Mutex{},
		configFile:         configFile,
		etcdAddress:        etcdAddress,
		etcdKey:            etcdKey,
		TimeoutWaitPromote: timeoutWaitPromote,
		MinVerSQLPromote:   minVerSQLPromote,
		TimeoutHostStatus:  timeoutHostStatus,
		TimeoutMasterDial:  timeoutMasterDial,
		TimeoutCheckMaster: timeoutCheckMaster,
	}
	if err := yaml.Unmarshal(b, s); err != nil {
		return nil, err
	}
	return s.chkConfig()
}

func (s *Config) chkConfig() (*Config, error) {
	t, err := template.New("t").Parse(pgConnTemplate)
	if err != nil {
		return nil, err
	}

	if s.PgUser == "" {
		return nil, errors.New("pg user not defined")
	}
	if s.PgPasswd == "" {
		return nil, errors.New("pg password not defined")
	}
	if s.Listen == "" {
		return nil, errors.New("listen port not defined")
	}

	for k, v := range s.Servers {
		host, port, err := net.SplitHostPort(v.Address)
		if err != nil {
			return nil, err
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
				host,
				port,
			}); err != nil {
			return nil, err
		}
		s.Servers[k].Host = host
		s.Servers[k].Port = port
		s.Servers[k].PgConn = str.String()
		s.Servers[k].PostPromoteCommand = strings.TrimSpace(s.Servers[k].PostPromoteCommand)
	}

	if len(s.Servers) == 0 {
		return nil, errors.New("server hot defined")
	}

	return s, nil
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
			Endpoints:   []string{s.etcdAddress},
			DialTimeout: 2 * time.Second,
		})
		defer cli.Close()
		if err != nil {
			return err
		}
		_, err = cli.Put(context.Background(), s.etcdKey, string(b))
		if err != nil {
			return err
		}
	default:
		return errors.New("config not defined")
	}
	return nil
}
