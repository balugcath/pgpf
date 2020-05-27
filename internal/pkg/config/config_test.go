package config

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"sync"
	"testing"
)

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
				configFile: "",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "test 2",
			args: args{
				configFile: "../../../configs/pgpf.yml",
			},
			want: &Config{
				configFile:           "../../../configs/pgpf.yml",
				Mutex:                &sync.Mutex{},
				etcdAddress:          "",
				etcdKey:              "",
				TimeoutWaitPromote:   timeoutWaitPromote,
				MinVerSQLPromote:     minVerSQLPromote,
				TimeoutHostStatus:    timeoutHostStatus,
				TimeoutCheckMaster:   timeoutCheckMaster,
				TimeoutMasterDial:    timeoutMasterDial,
				Listen:               ":5432",
				ShardListen:          ":5433",
				FailoverTimeout:      2,
				PgUser:               "postgres",
				PgPasswd:             "123",
				PrometheusListenPort: ":9091",
				Servers: map[string]*Server{
					"one": {
						Address: "192.168.1.25:5432", Use: true, Host: "192.168.1.25", Port: "5432",
						PgConn: "user=postgres password=123 dbname=postgres host=192.168.1.25 port=5432 sslmode=disable",
					},
					"two": {
						Address: "192.168.1.26:5432", Use: true, Host: "192.168.1.26", Port: "5432",
						PgConn:             "user=postgres password=123 dbname=postgres host=192.168.1.26 port=5432 sslmode=disable",
						Command:            "ssh -i id_rsa -o StrictHostKeyChecking=no root@192.168.1.26 touch /var/lib/postgresql/12/data/failover_triggerr",
						PostPromoteCommand: "ssh -i id_rsa -o StrictHostKeyChecking=no root@192.168.1.26 /root/change_master.sh {{.Host}} {{.Port}} {{.PgUser}}",
					},
					"three": {
						Address: "192.168.1.27:5432", Use: true, Host: "192.168.1.27", Port: "5432",
						PgConn:             "user=postgres password=123 dbname=postgres host=192.168.1.27 port=5432 sslmode=disable",
						Command:            "ssh -i id_rsa -o StrictHostKeyChecking=no root@192.168.1.27 touch /var/lib/postgresql/12/data/failover_triggerr",
						PostPromoteCommand: "ssh -i id_rsa -o StrictHostKeyChecking=no root@192.168.1.27 /root/change_master.sh {{.Host}} {{.Port}} {{.PgUser}}",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "test 3",
			args: args{
				configFile: "../../../configs/pgpf.ymll",
			},
			want:    nil,
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
			if !assert.Equal(t, got, tt.want) {
				t.Errorf("NewConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_Save(t *testing.T) {
	tests := []struct {
		name       string
		configFile string
		wantErr    bool
	}{
		{
			name:       "test 1",
			configFile: "../../../configs/pgpf.yml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s1, _ := NewConfig(tt.configFile, "", "")
			if err := s1.Save(); (err != nil) != tt.wantErr {
				t.Errorf("Config.Save() error = %v, wantErr %v", err, tt.wantErr)
			}
			s2, _ := NewConfig(tt.configFile, "", "")
			if !reflect.DeepEqual(s1, s2) {
				t.Errorf("NewConfig() = %v, want %v", s2, s1)
			}
		})
	}
}

func TestConfig_chkConfig(t *testing.T) {
	type fields struct {
		Mutex                *sync.Mutex
		etcdAddress          string
		etcdKey              string
		configFile           string
		TimeoutWaitPromote   int
		MinVerSQLPromote     float64
		TimeoutHostStatus    int
		TimeoutCheckMaster   int
		TimeoutMasterDial    int
		Listen               string
		ShardListen          string
		FailoverTimeout      int
		PgUser               string
		PgPasswd             string
		PrometheusListenPort string
		Servers              map[string]*Server
	}
	tests := []struct {
		name    string
		fields  fields
		want    *Config
		wantErr bool
	}{
		{
			name: "test 1",
			fields: fields{
				configFile:           "../../../configs/pgpf.yml",
				Mutex:                &sync.Mutex{},
				etcdAddress:          "",
				etcdKey:              "",
				TimeoutWaitPromote:   timeoutWaitPromote,
				MinVerSQLPromote:     minVerSQLPromote,
				TimeoutHostStatus:    timeoutHostStatus,
				TimeoutCheckMaster:   timeoutCheckMaster,
				TimeoutMasterDial:    timeoutMasterDial,
				Listen:               ":5432",
				ShardListen:          ":5433",
				FailoverTimeout:      2,
				PgUser:               "postgres",
				PgPasswd:             "123",
				PrometheusListenPort: ":9091",
				Servers: map[string]*Server{
					"one": {
						Address: "192.168.1.25:5432", Use: true,
					},
					"two": {
						Address: "192.168.1.26:5432", Use: true,
						Command: "ssh -i id_rsa -o StrictHostKeyChecking=no root@192.168.1.26 touch /var/lib/postgresql/12/data/failover_triggerr",
					},
				},
			},
			want: &Config{
				configFile:           "../../../configs/pgpf.yml",
				Mutex:                &sync.Mutex{},
				etcdAddress:          "",
				etcdKey:              "",
				TimeoutWaitPromote:   timeoutWaitPromote,
				MinVerSQLPromote:     minVerSQLPromote,
				TimeoutHostStatus:    timeoutHostStatus,
				TimeoutCheckMaster:   timeoutCheckMaster,
				TimeoutMasterDial:    timeoutMasterDial,
				Listen:               ":5432",
				ShardListen:          ":5433",
				FailoverTimeout:      2,
				PgUser:               "postgres",
				PgPasswd:             "123",
				PrometheusListenPort: ":9091",
				Servers: map[string]*Server{
					"one": {
						Host: "192.168.1.25", Port: "5432", Address: "192.168.1.25:5432", Use: true,
						PgConn: "user=postgres password=123 dbname=postgres host=192.168.1.25 port=5432 sslmode=disable",
					},
					"two": {
						Host: "192.168.1.26", Port: "5432", Address: "192.168.1.26:5432", Use: true,
						PgConn:  "user=postgres password=123 dbname=postgres host=192.168.1.26 port=5432 sslmode=disable",
						Command: "ssh -i id_rsa -o StrictHostKeyChecking=no root@192.168.1.26 touch /var/lib/postgresql/12/data/failover_triggerr",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "test 2",
			fields: fields{
				configFile:           "../../../configs/pgpf.yml",
				Mutex:                &sync.Mutex{},
				etcdAddress:          "",
				etcdKey:              "",
				TimeoutWaitPromote:   timeoutWaitPromote,
				MinVerSQLPromote:     minVerSQLPromote,
				TimeoutHostStatus:    timeoutHostStatus,
				TimeoutCheckMaster:   timeoutCheckMaster,
				TimeoutMasterDial:    timeoutMasterDial,
				Listen:               ":5432",
				ShardListen:          ":5433",
				FailoverTimeout:      2,
				PgUser:               "postgres",
				PgPasswd:             "123",
				PrometheusListenPort: ":9091",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "test 3",
			fields: fields{
				configFile:           "../../../configs/pgpf.yml",
				Mutex:                &sync.Mutex{},
				etcdAddress:          "",
				etcdKey:              "",
				TimeoutWaitPromote:   timeoutWaitPromote,
				MinVerSQLPromote:     minVerSQLPromote,
				TimeoutHostStatus:    timeoutHostStatus,
				TimeoutCheckMaster:   timeoutCheckMaster,
				TimeoutMasterDial:    timeoutMasterDial,
				Listen:               ":5432",
				ShardListen:          ":5433",
				FailoverTimeout:      2,
				PgPasswd:             "123",
				PrometheusListenPort: ":9091",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "test 4",
			fields: fields{
				configFile:           "../../../configs/pgpf.yml",
				Mutex:                &sync.Mutex{},
				etcdAddress:          "",
				etcdKey:              "",
				TimeoutWaitPromote:   timeoutWaitPromote,
				MinVerSQLPromote:     minVerSQLPromote,
				TimeoutHostStatus:    timeoutHostStatus,
				TimeoutCheckMaster:   timeoutCheckMaster,
				TimeoutMasterDial:    timeoutMasterDial,
				Listen:               ":5432",
				ShardListen:          ":5433",
				FailoverTimeout:      2,
				PgUser:               "postgres",
				PrometheusListenPort: ":9091",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "test 5",
			fields: fields{
				configFile:           "../../../configs/pgpf.yml",
				Mutex:                &sync.Mutex{},
				etcdAddress:          "",
				etcdKey:              "",
				TimeoutWaitPromote:   timeoutWaitPromote,
				MinVerSQLPromote:     minVerSQLPromote,
				TimeoutHostStatus:    timeoutHostStatus,
				TimeoutCheckMaster:   timeoutCheckMaster,
				TimeoutMasterDial:    timeoutMasterDial,
				ShardListen:          ":5433",
				FailoverTimeout:      2,
				PgUser:               "postgres",
				PgPasswd:             "123",
				PrometheusListenPort: ":9091",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "test 6",
			fields: fields{
				configFile:           "../../../configs/pgpf.yml",
				Mutex:                &sync.Mutex{},
				etcdAddress:          "",
				etcdKey:              "",
				TimeoutWaitPromote:   timeoutWaitPromote,
				MinVerSQLPromote:     minVerSQLPromote,
				TimeoutHostStatus:    timeoutHostStatus,
				TimeoutMasterDial:    timeoutMasterDial,
				TimeoutCheckMaster:   timeoutCheckMaster,
				Listen:               ":5432",
				ShardListen:          ":5433",
				FailoverTimeout:      2,
				PgUser:               "postgres",
				PgPasswd:             "123",
				PrometheusListenPort: ":9091",
				Servers: map[string]*Server{
					"one": {
						Address: "192.168.1.25*5432", Use: true,
					},
					"two": {
						Address: "192.168.1.26:5432", Use: true,
						Command: "ssh -i id_rsa -o StrictHostKeyChecking=no root@192.168.1.26 touch /var/lib/postgresql/12/data/failover_triggerr",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Config{
				Mutex:                tt.fields.Mutex,
				etcdAddress:          tt.fields.etcdAddress,
				etcdKey:              tt.fields.etcdKey,
				configFile:           tt.fields.configFile,
				TimeoutWaitPromote:   tt.fields.TimeoutWaitPromote,
				MinVerSQLPromote:     tt.fields.MinVerSQLPromote,
				TimeoutHostStatus:    tt.fields.TimeoutHostStatus,
				TimeoutCheckMaster:   tt.fields.TimeoutCheckMaster,
				TimeoutMasterDial:    tt.fields.TimeoutMasterDial,
				Listen:               tt.fields.Listen,
				ShardListen:          tt.fields.ShardListen,
				FailoverTimeout:      tt.fields.FailoverTimeout,
				PgUser:               tt.fields.PgUser,
				PgPasswd:             tt.fields.PgPasswd,
				PrometheusListenPort: tt.fields.PrometheusListenPort,
				Servers:              tt.fields.Servers,
			}
			got, err := s.chkConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.chkConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Config.chkConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
