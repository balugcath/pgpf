package failover

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/balugcath/pgpf/internal/pkg/config"
	"github.com/balugcath/pgpf/internal/pkg/metric"
	"github.com/balugcath/pgpf/internal/pkg/transport"
)

func TestFailover_checkMaster(t *testing.T) {
	type fields struct {
		Config      *config.Config
		Transporter transport.Transporter
	}
	type args struct {
		pgConn string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		err    error
	}{
		{
			name: "test 1",
			fields: fields{
				Config: &config.Config{
					FailoverTimeout:    2,
					TimeoutCheckMaster: 1,
				},
				Transporter: &transport.Mock{
					FOpen:       func(string) error { return nil },
					FIsRecovery: func() (bool, error) { return false, nil },
				},
			},
			args: args{pgConn: "one"},
			err:  ErrTerminate,
		},
		{
			name: "test 2",
			fields: fields{
				Config: &config.Config{
					FailoverTimeout:    2,
					TimeoutCheckMaster: 1,
				},
				Transporter: &transport.Mock{
					FOpen:       func(string) error { return errors.New("err") },
					FIsRecovery: func() (bool, error) { return false, nil },
				},
			},
			args: args{pgConn: "one"},
			err:  errors.New("err"),
		},
		{
			name: "test 3",
			fields: fields{
				Config: &config.Config{
					FailoverTimeout:    2,
					TimeoutCheckMaster: 1,
				},
				Transporter: &transport.Mock{
					FOpen:       func(string) error { return nil },
					FIsRecovery: func() (bool, error) { return false, errors.New("err") },
				},
			},
			args: args{pgConn: "one"},
			err:  ErrCheckHostFail,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Failover{
				Config:      tt.fields.Config,
				Transporter: tt.fields.Transporter,
			}
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(time.Second * 4)
				defer cancel()
			}()
			if err := s.checkMaster(ctx, tt.args.pgConn); !reflect.DeepEqual(err, tt.err) {
				t.Errorf("Failover.checkMaster() error = %v, wantErr %v", err, tt.err)
			}
		})
	}
}

func Test_wakeupSlave(t *testing.T) {
	type args struct {
		cmd string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test 1",
			args: args{
				cmd: "ls -al /",
			},
			wantErr: false,
		},
		{
			name: "test 2",
			args: args{
				cmd: "lslsls -al /",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := wakeupSlave(tt.args.cmd); (err != nil) != tt.wantErr {
				t.Errorf("wakeupSlave() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFailover_makeMaster(t *testing.T) {
	type fields struct {
		Config      *config.Config
		Transporter transport.Transporter
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr error
	}{
		{
			name: "test 1",
			fields: fields{
				Config: &config.Config{
					TimeoutWaitPromote: 2,
					MinVerSQLPromote:   12,
					Servers: map[string]*config.Server{
						"one": {
							PgConn: "one",
							Use:    true,
						},
						"two": {
							PgConn: "two",
							Use:    true,
						},
					},
				},
				Transporter: &transport.Mock{
					FHostStatus: func(h string, p bool) (bool, float64, error) {
						if h == "one" {
							return false, 12, nil
						}
						return true, 12, nil
					},
				},
			},
			want:    "one",
			wantErr: nil,
		},
		{
			name: "test 2",
			fields: fields{
				Config: &config.Config{
					TimeoutWaitPromote: 2,
					MinVerSQLPromote:   12,
					Servers: map[string]*config.Server{
						"one": {
							PgConn: "one",
							Use:    true,
						},
						"two": {
							PgConn: "two",
							Use:    true,
						},
					},
				},
				Transporter: &transport.Mock{
					PromoteDone: make(map[string]bool),
					FHostStatus: func(h string, p bool) (bool, float64, error) {
						if h == "one" {
							return true, 12, nil
						}
						return !p, 12, nil
					},
					FPromote: func(h string) error {
						if h == "one" {
							return errors.New("err")
						}
						return nil
					},
				},
			},
			want:    "two",
			wantErr: nil,
		},
		{
			name: "test 3",
			fields: fields{
				Config: &config.Config{
					TimeoutWaitPromote: 2,
					MinVerSQLPromote:   12,
					Servers: map[string]*config.Server{
						"one": {
							PgConn: "one",
							Use:    true,
						},
						"two": {
							PgConn: "two",
							Use:    true,
						},
					},
				},
				Transporter: &transport.Mock{
					PromoteDone: make(map[string]bool),
					FHostStatus: func(h string, p bool) (bool, float64, error) {
						return true, 12, errors.New("err")
					},
					FPromote: func(h string) error {
						return errors.New("err")
					},
				},
			},
			want:    "",
			wantErr: ErrNoMasterFound,
		},
		{
			name: "test 4",
			fields: fields{
				Config: &config.Config{
					TimeoutWaitPromote: 2,
					MinVerSQLPromote:   12,
					Servers: map[string]*config.Server{
						"one": {
							PgConn: "one",
							Use:    true,
						},
						"two": {
							PgConn: "two",
							Use:    true,
						},
					},
				},
				Transporter: &transport.Mock{
					PromoteDone: make(map[string]bool),
					FHostStatus: func(h string, p bool) (bool, float64, error) {
						return true, 10, nil
					},
					FPromote: func(h string) error {
						return nil
					},
				},
			},
			want:    "",
			wantErr: ErrNoMasterFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Failover{
				Config:      tt.fields.Config,
				Transporter: tt.fields.Transporter,
			}
			got, err := s.makeMaster()
			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("Failover.makeMaster() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Failover.makeMaster() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFailover_findStandby(t *testing.T) {
	type fields struct {
		Config      *config.Config
		Transporter transport.Transporter
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr error
	}{
		{
			name: "test 1",
			fields: fields{
				Config: &config.Config{
					TimeoutWaitPromote: 2,
					MinVerSQLPromote:   12,
					Servers: map[string]*config.Server{
						"one": {
							PgConn: "one",
							Use:    true,
						},
						"two": {
							PgConn: "two",
							Use:    true,
						},
					},
				},
				Transporter: &transport.Mock{
					FHostStatus: func(h string, p bool) (bool, float64, error) {
						if h == "one" {
							return false, 12, nil
						}
						return true, 12, nil
					},
				},
			},
			want:    "two",
			wantErr: nil,
		},
		{
			name: "test 2",
			fields: fields{
				Config: &config.Config{
					TimeoutWaitPromote: 2,
					MinVerSQLPromote:   12,
					Servers: map[string]*config.Server{
						"one": {
							PgConn: "one",
							Use:    true,
						},
						"two": {
							PgConn: "two",
							Use:    true,
						},
					},
				},
				Transporter: &transport.Mock{
					FHostStatus: func(h string, p bool) (bool, float64, error) {
						return false, 12, nil
					},
				},
			},
			want:    "",
			wantErr: ErrNoStandbyFound,
		},
		{
			name: "test 3",
			fields: fields{
				Config: &config.Config{
					TimeoutWaitPromote: 2,
					MinVerSQLPromote:   12,
					Servers: map[string]*config.Server{
						"one": {
							PgConn: "one",
							Use:    true,
						},
						"two": {
							PgConn: "two",
							Use:    true,
						},
					},
				},
				Transporter: &transport.Mock{
					FHostStatus: func(h string, p bool) (bool, float64, error) {
						return false, 12, errors.New("err")
					},
				},
			},
			want:    "",
			wantErr: ErrNoStandbyFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Failover{
				Config:      tt.fields.Config,
				Transporter: tt.fields.Transporter,
			}
			got, err := s.findStandby()
			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("Failover.findStandby() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Failover.findStandby() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFailover_Start(t *testing.T) {
	type fields struct {
		Config      *config.Config
		Transporter transport.Transporter
		Metricer    metric.Metricer
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr error
	}{
		{
			name: "test 1",
			fields: fields{
				Config: &config.Config{
					Mutex:              &sync.Mutex{},
					TimeoutWaitPromote: 1,
					TimeoutCheckMaster: 1,
					TimeoutHostStatus:  1,
					MinVerSQLPromote:   12,
					Listen:             ":5432",
					ShardListen:        ":5433",
					Servers: map[string]*config.Server{
						"one": {
							PgConn: "one",
							Use:    true,
						},
						"two": {
							PgConn: "two",
							Use:    true,
						},
					},
				},
				Transporter: &transport.Mock{
					PromoteDone: make(map[string]bool),
					FOpen:       func(string) error { return nil },
					FIsRecovery: func() (bool, error) { return false, errors.New("err") },
					FHostStatus: func(h string, p bool) (bool, float64, error) {
						if h == "one" {
							return false, 12, nil
						}
						return false, 12, nil
					},
					FPromote: func(h string) error {
						if h == "one" {
							return errors.New("err")
						}
						return nil
					},
				},
				Metricer: metric.NewMock(),
			},
			wantErr: ErrTerminate,
		},
		{
			name: "test 2",
			fields: fields{
				Config: &config.Config{
					Mutex:              &sync.Mutex{},
					TimeoutWaitPromote: 1,
					TimeoutCheckMaster: 1,
					TimeoutHostStatus:  1,
					MinVerSQLPromote:   12,
					Listen:             ":5432",
					ShardListen:        ":5433",
					Servers: map[string]*config.Server{
						"one": {
							PgConn: "one",
							Use:    true,
						},
						"two": {
							PgConn: "two",
							Use:    true,
						},
					},
				},
				Transporter: &transport.Mock{
					PromoteDone: make(map[string]bool),
					FOpen:       func(string) error { return errors.New("err") },
					FIsRecovery: func() (bool, error) { return false, errors.New("err") },
					FHostStatus: func(h string, p bool) (bool, float64, error) {
						if h == "one" {
							return false, 12, nil
						}
						return true, 12, nil
					},
					FPromote: func(h string) error {
						if h == "one" {
							return errors.New("err")
						}
						return nil
					},
				},
				Metricer: metric.NewMock(),
			},
			wantErr: ErrNoMasterFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewFailover(tt.fields.Config)
			s.Transporter = tt.fields.Transporter
			s.Metricer = tt.fields.Metricer
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(time.Second * 2)
				defer cancel()
			}()
			if err := s.startFailover(ctx); !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("Failover.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFailover_startChkSatusHosts(t *testing.T) {
	type fields struct {
		Config   *config.Config
		Metricer *metric.Mock
	}
	type args struct {
		tr transport.Transporter
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string]int
	}{
		{
			name: "test 1",
			fields: fields{
				Config: &config.Config{
					TimeoutHostStatus: 1,
					Servers: map[string]*config.Server{
						"one": {
							PgConn: "one",
							Use:    true,
						},
						"two": {
							PgConn: "two",
							Use:    true,
						},
						"three": {
							PgConn: "three",
							Use:    true,
						},
						"four": {
							PgConn: "four",
							Use:    false,
						},
					},
				},
				Metricer: metric.NewMock(),
			},
			args: args{
				tr: &transport.Mock{
					FHostStatus: func(h string, p bool) (bool, float64, error) {
						switch h {
						case "one":
							return false, 12, nil
						case "two":
							return true, 12, nil
						default:
							return true, 12, errors.New("err")
						}
					},
				},
			},
			want: map[string]int{"one": hostMaster, "two": hostStandby, "three": hostDead},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Failover{
				Config:   tt.fields.Config,
				Metricer: tt.fields.Metricer,
			}
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(time.Second * 2)
				defer cancel()
			}()
			s.startChkSatusHosts(ctx, tt.args.tr)

			if !reflect.DeepEqual(tt.fields.Metricer.HostStatusData, tt.want) {
				t.Errorf("Failover.makeMaster() error = %v, wantErr %v", tt.fields.Metricer.HostStatusData, tt.want)
			}
		})
	}
}
