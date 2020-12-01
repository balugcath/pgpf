// +build integration_test
// see ../../../build/.drone.yml

package config

import (
	"sync"
	"testing"
)

const (
	etcdAddress = "etcd_service:2379"
)

func TestConfig_SaveEtcd(t *testing.T) {
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
				etcdAddress: "",
			},
		},
		{
			name:    "test 2",
			wantErr: false,
			fields: fields{
				etcdAddress: etcdAddress,
				etcdKey:     "a",
			},
		},
		{
			name:    "test 3",
			wantErr: true,
			fields: fields{
				etcdAddress: etcdAddress,
				etcdKey:     "",
			},
		},
		{
			name:    "test 4",
			wantErr: true,
			fields: fields{
				etcdAddress: ":1234",
				etcdKey:     "aa",
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
		})
	}
}

func TestConfig_loadEtcd(t *testing.T) {
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
				etcdAddress: "",
			},
		},
		{
			name:    "test 2",
			wantErr: false,
			fields: fields{
				etcdAddress: etcdAddress,
				etcdKey:     "a",
			},
		},
		{
			name:    "test 3",
			wantErr: true,
			fields: fields{
				etcdAddress: etcdAddress,
				etcdKey:     "aa",
			},
		},
		{
			name:    "test 4",
			wantErr: true,
			fields: fields{
				etcdAddress: ":1234",
				etcdKey:     "aa",
			},
		},
		{
			name:    "test 5",
			wantErr: true,
			fields: fields{
				etcdAddress: "/",
				etcdKey:     "aa",
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
