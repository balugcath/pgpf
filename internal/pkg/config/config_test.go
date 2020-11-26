package config

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_checkConfig(t *testing.T) {
	type fields struct {
		Mutex                  *sync.Mutex
		etcdAddress            string
		etcdKey                string
		configFile             string
		MinVerSQLPromote       float64
		TimeoutWaitPromoteSec  int
		HostStatusIntervalSec  int
		CheckMasterIntervalSec int
		FailoverTimeoutSec     int
		TimeoutMasterDialSec   int
		Listen                 string
		ShardListen            string
		PgUser                 string
		PgPasswd               string
		PrometheusListenPort   string
		Servers                map[string]*Server
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "test 1",
			wantErr: true,
		},
		{
			name:    "test 2",
			wantErr: true,
			fields: fields{
				PgUser: "1",
			},
		},
		{
			name:    "test 3",
			wantErr: true,
			fields: fields{
				PgUser:   "1",
				PgPasswd: "2",
			},
		},
		{
			name:    "test 4",
			wantErr: true,
			fields: fields{
				PgUser:   "1",
				PgPasswd: "2",
				Listen:   "3",
			},
		},
		{
			name:    "test 5",
			wantErr: true,
			fields: fields{
				PgUser:      "1",
				PgPasswd:    "2",
				Listen:      "3",
				ShardListen: "3",
			},
		},
		{
			name:    "test 6",
			wantErr: true,
			fields: fields{
				PgUser:      "1",
				PgPasswd:    "2",
				Listen:      "3",
				ShardListen: "4",
			},
		},
		{
			name:    "test 7",
			wantErr: true,
			fields: fields{
				PgUser:                 "1",
				PgPasswd:               "2",
				Listen:                 "3",
				ShardListen:            "4",
				CheckMasterIntervalSec: 5,
			},
		},
		{
			name:    "test 8",
			wantErr: true,
			fields: fields{
				PgUser:                 "1",
				PgPasswd:               "2",
				Listen:                 "3",
				ShardListen:            "4",
				CheckMasterIntervalSec: 5,
				FailoverTimeoutSec:     1,
			},
		},
		{
			name:    "test 9",
			wantErr: true,
			fields: fields{
				PgUser:                 "1",
				PgPasswd:               "2",
				Listen:                 "3",
				ShardListen:            "4",
				CheckMasterIntervalSec: 5,
				FailoverTimeoutSec:     6,
			},
		},
		{
			name:    "test 10",
			wantErr: false,
			fields: fields{
				PgUser:                 "1",
				PgPasswd:               "2",
				Listen:                 "3",
				ShardListen:            "4",
				CheckMasterIntervalSec: 5,
				FailoverTimeoutSec:     6,
				Servers:                map[string]*Server{"one": {}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Config{
				Mutex:                  tt.fields.Mutex,
				etcdAddress:            tt.fields.etcdAddress,
				etcdKey:                tt.fields.etcdKey,
				configFile:             tt.fields.configFile,
				MinVerSQLPromote:       tt.fields.MinVerSQLPromote,
				TimeoutWaitPromoteSec:  tt.fields.TimeoutWaitPromoteSec,
				HostStatusIntervalSec:  tt.fields.HostStatusIntervalSec,
				CheckMasterIntervalSec: tt.fields.CheckMasterIntervalSec,
				FailoverTimeoutSec:     tt.fields.FailoverTimeoutSec,
				TimeoutMasterDialSec:   tt.fields.TimeoutMasterDialSec,
				Listen:                 tt.fields.Listen,
				ShardListen:            tt.fields.ShardListen,
				PgUser:                 tt.fields.PgUser,
				PgPasswd:               tt.fields.PgPasswd,
				PrometheusListenPort:   tt.fields.PrometheusListenPort,
				Servers:                tt.fields.Servers,
			}
			if err := s.checkConfig(); (err != nil) != tt.wantErr {
				t.Errorf("Config.checkConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_prepeareConfig(t *testing.T) {
	type fields struct {
		Mutex                  *sync.Mutex
		etcdAddress            string
		etcdKey                string
		configFile             string
		MinVerSQLPromote       float64
		TimeoutWaitPromoteSec  int
		HostStatusIntervalSec  int
		CheckMasterIntervalSec int
		FailoverTimeoutSec     int
		TimeoutMasterDialSec   int
		Listen                 string
		ShardListen            string
		PgUser                 string
		PgPasswd               string
		PrometheusListenPort   string
		Servers                map[string]*Server
	}
	type args struct {
		pgConnTemplate string
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantErr    bool
		wantConfig Config
	}{
		{
			name:    "test 1",
			args:    args{pgConnTemplate: "{{"},
			wantErr: true,
		},
		{
			name: "test 2",
			args: args{pgConnTemplate: pgConnTemplate},
			fields: fields{
				Servers: map[string]*Server{"one": {
					Address: "qwe",
				}},
			},
			wantErr: true,
		},
		{
			name: "test 3",
			args: args{pgConnTemplate: pgConnTemplate},
			fields: fields{
				Servers: map[string]*Server{"one": {
					Address: "1.1.1.1:1234",
				}},
			},
			wantErr: false,
			wantConfig: Config{
				Servers: map[string]*Server{"one": {
					Address: "1.1.1.1:1234",
					Host:    "1.1.1.1",
					Port:    "1234",
					PgConn:  "user= password= dbname=postgres host=1.1.1.1 port=1234 sslmode=disable",
				}},
			},
		},
		{
			name: "test 4",
			args: args{pgConnTemplate: pgConnTemplate},
			fields: fields{
				PgUser:   "one",
				PgPasswd: "one",
				Servers: map[string]*Server{"one": {
					Address: "1.1.1.1:1234",
				}},
			},
			wantErr: false,
			wantConfig: Config{
				PgUser:   "one",
				PgPasswd: "one",
				Servers: map[string]*Server{"one": {
					Address: "1.1.1.1:1234",
					Host:    "1.1.1.1",
					Port:    "1234",
					PgConn:  "user=one password=one dbname=postgres host=1.1.1.1 port=1234 sslmode=disable",
				}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Config{
				Mutex:                  tt.fields.Mutex,
				etcdAddress:            tt.fields.etcdAddress,
				etcdKey:                tt.fields.etcdKey,
				configFile:             tt.fields.configFile,
				MinVerSQLPromote:       tt.fields.MinVerSQLPromote,
				TimeoutWaitPromoteSec:  tt.fields.TimeoutWaitPromoteSec,
				HostStatusIntervalSec:  tt.fields.HostStatusIntervalSec,
				CheckMasterIntervalSec: tt.fields.CheckMasterIntervalSec,
				FailoverTimeoutSec:     tt.fields.FailoverTimeoutSec,
				TimeoutMasterDialSec:   tt.fields.TimeoutMasterDialSec,
				Listen:                 tt.fields.Listen,
				ShardListen:            tt.fields.ShardListen,
				PgUser:                 tt.fields.PgUser,
				PgPasswd:               tt.fields.PgPasswd,
				PrometheusListenPort:   tt.fields.PrometheusListenPort,
				Servers:                tt.fields.Servers,
			}
			if err := s.prepeareConfig(tt.args.pgConnTemplate); (err != nil) != tt.wantErr {
				t.Errorf("Config.prepeareConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !assert.Equal(t, tt.wantConfig, *s) {
				t.Errorf("Config.prepeareConfig() got = %v, wantErr %v", *s, tt.wantConfig)
			}
		})
	}
}

func TestConfig_Save(t *testing.T) {
	type fields struct {
		Mutex                  *sync.Mutex
		etcdAddress            string
		etcdKey                string
		configFile             string
		MinVerSQLPromote       float64
		TimeoutWaitPromoteSec  int
		HostStatusIntervalSec  int
		CheckMasterIntervalSec int
		FailoverTimeoutSec     int
		TimeoutMasterDialSec   int
		Listen                 string
		ShardListen            string
		PgUser                 string
		PgPasswd               string
		PrometheusListenPort   string
		Servers                map[string]*Server
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "test 1",
			wantErr: true,
			fields: fields{
				configFile: ".",
			},
		},
		{
			name:    "test 2",
			wantErr: false,
			fields: fields{
				configFile: "a",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Config{
				Mutex:                  tt.fields.Mutex,
				etcdAddress:            tt.fields.etcdAddress,
				etcdKey:                tt.fields.etcdKey,
				configFile:             tt.fields.configFile,
				MinVerSQLPromote:       tt.fields.MinVerSQLPromote,
				TimeoutWaitPromoteSec:  tt.fields.TimeoutWaitPromoteSec,
				HostStatusIntervalSec:  tt.fields.HostStatusIntervalSec,
				CheckMasterIntervalSec: tt.fields.CheckMasterIntervalSec,
				FailoverTimeoutSec:     tt.fields.FailoverTimeoutSec,
				TimeoutMasterDialSec:   tt.fields.TimeoutMasterDialSec,
				Listen:                 tt.fields.Listen,
				ShardListen:            tt.fields.ShardListen,
				PgUser:                 tt.fields.PgUser,
				PgPasswd:               tt.fields.PgPasswd,
				PrometheusListenPort:   tt.fields.PrometheusListenPort,
				Servers:                tt.fields.Servers,
			}
			if err := s.Save(); (err != nil) != tt.wantErr {
				t.Errorf("Config.Save() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				os.Remove(tt.fields.configFile)
			}
		})
	}
}

func TestConfig_load(t *testing.T) {
	type fields struct {
		Mutex                  *sync.Mutex
		etcdAddress            string
		etcdKey                string
		configFile             string
		MinVerSQLPromote       float64
		TimeoutWaitPromoteSec  int
		HostStatusIntervalSec  int
		CheckMasterIntervalSec int
		FailoverTimeoutSec     int
		TimeoutMasterDialSec   int
		Listen                 string
		ShardListen            string
		PgUser                 string
		PgPasswd               string
		PrometheusListenPort   string
		Servers                map[string]*Server
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "test 1",
			wantErr: true,
			fields: fields{
				configFile: ".",
			},
		},
		{
			name:    "test 2",
			wantErr: false,
			fields: fields{
				configFile: "../../../configs/pgpf.yml",
			},
		},
		{
			name:    "test 3",
			wantErr: true,
			fields: fields{
				configFile: "config.go",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Config{
				Mutex:                  tt.fields.Mutex,
				etcdAddress:            tt.fields.etcdAddress,
				etcdKey:                tt.fields.etcdKey,
				configFile:             tt.fields.configFile,
				MinVerSQLPromote:       tt.fields.MinVerSQLPromote,
				TimeoutWaitPromoteSec:  tt.fields.TimeoutWaitPromoteSec,
				HostStatusIntervalSec:  tt.fields.HostStatusIntervalSec,
				CheckMasterIntervalSec: tt.fields.CheckMasterIntervalSec,
				FailoverTimeoutSec:     tt.fields.FailoverTimeoutSec,
				TimeoutMasterDialSec:   tt.fields.TimeoutMasterDialSec,
				Listen:                 tt.fields.Listen,
				ShardListen:            tt.fields.ShardListen,
				PgUser:                 tt.fields.PgUser,
				PgPasswd:               tt.fields.PgPasswd,
				PrometheusListenPort:   tt.fields.PrometheusListenPort,
				Servers:                tt.fields.Servers,
			}
			if err := s.load(); (err != nil) != tt.wantErr {
				t.Errorf("Config.load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewConfig(t *testing.T) {
	type args struct {
		configFile  string
		etcdAddress string
		etcdKey     string
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		{
			name: "test 1",
			args: args{
				configFile: "../../../configs/pgpf_minimal.yml",
			},
			want: &Config{
				configFile:             "../../../configs/pgpf_minimal.yml",
				Listen:                 ":5432",
				PgUser:                 "postgres",
				PgPasswd:               "123",
				Mutex:                  &sync.Mutex{},
				MinVerSQLPromote:       12,
				TimeoutWaitPromoteSec:  60,
				HostStatusIntervalSec:  10,
				TimeoutMasterDialSec:   1,
				CheckMasterIntervalSec: 1,
				FailoverTimeoutSec:     8,
				Servers: map[string]*Server{
					"one": {
						Address: "192.168.1.25:5432",
						Use:     true,
						Host:    "192.168.1.25",
						Port:    "5432",
						PgConn:  "user=postgres password=123 dbname=postgres host=192.168.1.25 port=5432 sslmode=disable",
					},
					"two": {
						Address: "192.168.1.26:5432",
						Use:     true,
						Host:    "192.168.1.26",
						Port:    "5432",
						PgConn:  "user=postgres password=123 dbname=postgres host=192.168.1.26 port=5432 sslmode=disable",
					},
				},
			},
		},
		{
			name: "test 2",
			args: args{
				configFile: "n/a",
			},
			wantErr: true,
		},
		{
			name: "test 3",
			args: args{
				configFile: "../../../configs/pgpf_bad1.yml",
			},
			wantErr: true,
		},
		{
			name: "test 4",
			args: args{
				configFile: "../../../configs/pgpf_bad2.yml",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewConfig(tt.args.configFile, tt.args.etcdAddress, tt.args.etcdKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !assert.Equal(t, tt.want, got) {
				t.Errorf("NewConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
