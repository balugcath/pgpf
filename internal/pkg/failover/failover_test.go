package failover

import (
	"context"
	"errors"
	"net"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/balugcath/pgpf/internal/pkg/config"
)

var errFoo = errors.New("err foo")

func init() {
	timeUnit = time.Millisecond
}

type trMock struct {
	retOpen              map[string][]error
	retPromote           map[string][]error
	retChangePrimaryConn map[string][]error
	retIsRecovery        map[string][]struct {
		f   bool
		err error
	}
	retHostVersion map[string][]struct {
		v   float64
		err error
	}
	key string
}

func (s *trMock) Open(key string) (err error) {
	s.key = key
	err = s.retOpen[s.key][0]
	if len(s.retOpen[s.key]) > 1 {
		s.retOpen[s.key] = s.retOpen[s.key][1:]
	}
	return
}

func (s *trMock) Close() {
	s.key = ""
}

func (s *trMock) IsRecovery() (f bool, err error) {
	f = s.retIsRecovery[s.key][0].f
	err = s.retIsRecovery[s.key][0].err
	if len(s.retIsRecovery[s.key]) > 1 {
		s.retIsRecovery[s.key] = s.retIsRecovery[s.key][1:]
	}
	return
}

func (s *trMock) HostVersion() (v float64, err error) {
	v = s.retHostVersion[s.key][0].v
	err = s.retHostVersion[s.key][0].err
	if len(s.retHostVersion[s.key]) > 1 {
		s.retHostVersion[s.key] = s.retHostVersion[s.key][1:]
	}
	return
}

func (s *trMock) Promote() (err error) {
	err = s.retPromote[s.key][0]
	if len(s.retPromote[s.key]) > 1 {
		s.retPromote[s.key] = s.retPromote[s.key][1:]
	}
	return
}

func (s *trMock) ChangePrimaryConn(string, string, string) (err error) {
	err = s.retChangePrimaryConn[s.key][0]
	if len(s.retChangePrimaryConn[s.key]) > 1 {
		s.retChangePrimaryConn[s.key] = s.retChangePrimaryConn[s.key][1:]
	}
	return
}

func TestFailover_checkMaster1(t *testing.T) {
	type fields struct {
		transporter transporter
		*config.Config
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr error
	}{
		{
			name: "test 1",
			fields: fields{
				transporter: &trMock{
					retOpen: map[string][]error{"one": {errFoo}},
				},
			},
			wantErr: errFoo,
		},
		{
			name: "test 2",
			fields: fields{
				Config: &config.Config{
					FailoverTimeoutSec:     10,
					CheckMasterIntervalSec: 1,
				},
				transporter: &trMock{
					retOpen: map[string][]error{"one": {nil}},
					retIsRecovery: map[string][]struct {
						f   bool
						err error
					}{"one": {{false, nil}, {false, errFoo}}},
				},
			},
			wantErr: ErrCheckHostFail,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Failover{
				transporter: tt.fields.transporter,
				Config:      tt.fields.Config,
			}
			doneCtx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if err := s.checkMaster(doneCtx, "one"); err != tt.wantErr {
				t.Errorf("Failover.checkMaster1() got = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestFailover_checkMaster2(t *testing.T) {
	type fields struct {
		transporter transporter
		*config.Config
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
					FailoverTimeoutSec:     10,
					CheckMasterIntervalSec: 1,
				},
				transporter: &trMock{
					retOpen: map[string][]error{"one": {nil}},
					retIsRecovery: map[string][]struct {
						f   bool
						err error
					}{"one": {{false, nil}}},
				},
			},
			wantErr: ErrTerminate,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Failover{
				transporter: tt.fields.transporter,
				Config:      tt.fields.Config,
			}
			doneCtx, cancel := context.WithCancel(context.Background())
			go func() {
				cancel()
			}()

			if err := s.checkMaster(doneCtx, "one"); err != tt.wantErr {
				t.Errorf("Failover.checkMaster2() got = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func Test_execCommand(t *testing.T) {
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
			if err := execCommand(tt.args.cmd); (err != nil) != tt.wantErr {
				t.Errorf("execCommand() got = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestFailover_findServer(t *testing.T) {
	type fields struct {
		transporter transporter
		*config.Config
	}
	tests := []struct {
		name       string
		fields     fields
		wantErr    error
		wantRet    string
		isRecovery bool
	}{
		{
			name: "test 1",
			fields: fields{
				Config: &config.Config{
					FailoverTimeoutSec:     10,
					CheckMasterIntervalSec: 1,
					Servers: map[string]*config.Server{
						"one": {
							PgConn: "one",
							Use:    true,
						},
						"two": {
							PgConn: "two",
							Use:    false,
						},
						"three": {
							PgConn: "three",
							Use:    true,
						},
						"four": {
							PgConn: "four",
							Use:    true,
						},
					},
				},
				transporter: &trMock{
					retOpen: map[string][]error{"one": {errFoo}, "two": {nil}, "three": {nil}, "four": {nil}},
					retIsRecovery: map[string][]struct {
						f   bool
						err error
					}{"one": {{false, nil}}, "two": {{false, nil}}, "three": {{true, errFoo}}, "four": {{true, nil}}},
				},
			},
			wantErr:    nil,
			wantRet:    "four",
			isRecovery: true,
		},
		{
			name: "test 2",
			fields: fields{
				Config: &config.Config{
					FailoverTimeoutSec:     10,
					CheckMasterIntervalSec: 1,
					Servers: map[string]*config.Server{
						"one": {
							PgConn: "one",
							Use:    true,
						},
						"two": {
							PgConn: "two",
							Use:    false,
						},
						"three": {
							PgConn: "three",
							Use:    true,
						},
						"four": {
							PgConn: "four",
							Use:    true,
						},
					},
				},
				transporter: &trMock{
					retOpen: map[string][]error{"one": {errFoo}, "two": {nil}, "three": {nil}, "four": {nil}},
					retIsRecovery: map[string][]struct {
						f   bool
						err error
					}{"one": {{false, nil}}, "two": {{false, errFoo}}, "three": {{true, errFoo}}, "four": {{false, nil}}},
				},
			},
			wantErr:    ErrServerNotFound,
			wantRet:    "",
			isRecovery: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Failover{
				transporter: tt.fields.transporter,
				Config:      tt.fields.Config,
			}
			r, err := s.findServer(tt.isRecovery)
			if err != tt.wantErr {
				t.Errorf("Failover.findServer() got = %v, want %v", err, tt.wantErr)
			}
			if r != tt.wantRet {
				t.Errorf("Failover.findServer() got = %v, want %v", r, tt.wantRet)
			}
		})
	}
}

func TestFailover_makeMaster1(t *testing.T) {
	type fields struct {
		transporter transporter
		*config.Config
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr error
		wantRet string
	}{
		{
			name: "test 1",
			fields: fields{
				Config: &config.Config{
					Servers: map[string]*config.Server{
						"one": {
							PgConn: "one",
							Use:    false,
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
							Use:    true,
						},
						"five": {
							PgConn: "five",
							Use:    true,
						},
						"six": {
							PgConn: "six",
							Use:    true,
						},
						"seven": {
							PgConn: "seven",
							Use:    true,
						},
					},
				},
				transporter: &trMock{
					retOpen: map[string][]error{"two": {errFoo}, "three": {nil},
						"four": {nil}, "five": {nil}, "six": {nil}, "seven": {nil}},
					retIsRecovery: map[string][]struct {
						f   bool
						err error
					}{"three": {{true, errFoo}}, "four": {{false, nil}}, "five": {{true, nil}},
						"six": {{true, nil}}, "seven": {{true, nil}, {true, errFoo}}},
					retHostVersion: map[string][]struct {
						v   float64
						err error
					}{"five": {{-1, errFoo}}, "six": {{12, nil}}, "seven": {{12, nil}}},
					retPromote: map[string][]error{"six": {errFoo}, "seven": {nil}},
				},
			},
			wantErr: ErrServerNotFound,
			wantRet: "",
		},
		{
			name: "test 2",
			fields: fields{
				Config: &config.Config{
					TimeoutWaitPromoteSec: 4,
					Servers: map[string]*config.Server{
						"one": {
							PgConn: "one",
							Use:    false,
						},
						"two": {
							PgConn: "two",
							Use:    true,
						},
					},
				},
				transporter: &trMock{
					retOpen: map[string][]error{"two": {nil}},
					retIsRecovery: map[string][]struct {
						f   bool
						err error
					}{"two": {{true, nil}, {true, nil}, {false, nil}}},
					retHostVersion: map[string][]struct {
						v   float64
						err error
					}{"two": {{12, nil}}},
					retPromote: map[string][]error{"two": {nil}},
				},
			},
			wantErr: nil,
			wantRet: "two",
		},
		{
			name: "test 3",
			fields: fields{
				Config: &config.Config{
					TimeoutWaitPromoteSec: 4,
					MinVerSQLPromote:      12,
					Servers: map[string]*config.Server{
						"one": {
							PgConn:  "one",
							Use:     true,
							Command: "lsls",
						},
						"two": {
							PgConn:  "two",
							Use:     true,
							Command: "ls",
						},
					},
				},
				transporter: &trMock{
					retOpen: map[string][]error{"one": {nil}, "two": {nil}},
					retIsRecovery: map[string][]struct {
						f   bool
						err error
					}{"one": {{true, nil}}, "two": {{true, nil}, {true, nil}, {false, nil}}},
					retHostVersion: map[string][]struct {
						v   float64
						err error
					}{"one": {{10, nil}}, "two": {{10, nil}}},
				},
			},
			wantErr: nil,
			wantRet: "two",
		},
		{
			name: "test 4",
			fields: fields{
				Config: &config.Config{
					TimeoutWaitPromoteSec: 4,
					MinVerSQLPromote:      12,
					Servers: map[string]*config.Server{
						"one": {
							PgConn:             "one",
							Use:                true,
							PostPromoteCommand: "ls",
							Host:               "a",
							Port:               "b",
						},
						"two": {
							PgConn:             "two",
							Use:                true,
							PostPromoteCommand: "ls",
						},
						"three": {
							PgConn:             "three",
							Use:                true,
							PostPromoteCommand: "lsls",
						},
					},
				},
				transporter: &trMock{
					retOpen: map[string][]error{"one": {nil}, "two": {errFoo}, "three": {errFoo}},
					retIsRecovery: map[string][]struct {
						f   bool
						err error
					}{"one": {{true, nil}, {false, nil}}},
					retHostVersion: map[string][]struct {
						v   float64
						err error
					}{"one": {{12, nil}}},
					retPromote: map[string][]error{"one": {nil}},
				},
			},
			wantErr: nil,
			wantRet: "one",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Failover{
				transporter: tt.fields.transporter,
				Config:      tt.fields.Config,
			}
			r, err := s.makeMaster()
			if err != tt.wantErr {
				t.Errorf("Failover.makeMaster1() got = %v, want %v", err, tt.wantErr)
			}
			if r != tt.wantRet {
				t.Errorf("Failover.makeMaster1() got = %v, want %v", r, tt.wantRet)
			}
			time.Sleep(time.Second)
		})
	}
}

type m struct {
}

func (m) Register(_ int, _, _ string, _ ...string) error { return nil }
func (m) Add(_ string, _ ...interface{}) error           { return nil }
func (m) Set(_ string, _ ...interface{}) error           { return nil }
func (m) Inc(_ string, _ ...interface{}) error           { return nil }
func (m) Dec(_ string, _ ...interface{}) error           { return nil }

func TestFailover_start1(t *testing.T) {
	type fields struct {
		Config      *config.Config
		transporter transporter
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
					Mutex:                  &sync.Mutex{},
					TimeoutWaitPromoteSec:  4,
					FailoverTimeoutSec:     10,
					CheckMasterIntervalSec: 1,
					MinVerSQLPromote:       12,
					Listen:                 ":5051",
					ShardListen:            ":5052",
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
					},
				},
				transporter: &trMock{
					retOpen: map[string][]error{"one": {nil}, "two": {nil}, "three": {nil}},
					retIsRecovery: map[string][]struct {
						f   bool
						err error
					}{"one": {{false, nil}}, "two": {{true, nil}}, "three": {{true, nil}}},
					retHostVersion: map[string][]struct {
						v   float64
						err error
					}{"one": {{12, nil}}, "two": {{12, nil}}, "three": {{12, nil}}},
				},
			},
			wantErr: ErrTerminate,
		},
		{
			name: "test 2",
			fields: fields{
				Config: &config.Config{
					Mutex:                  &sync.Mutex{},
					TimeoutWaitPromoteSec:  4,
					FailoverTimeoutSec:     10,
					CheckMasterIntervalSec: 1,
					MinVerSQLPromote:       12,
					Listen:                 ":5051",
					ShardListen:            ":5052",
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
					},
				},
				transporter: &trMock{
					retOpen: map[string][]error{"one": {errFoo}, "two": {errFoo}, "three": {errFoo}},
				},
			},
			wantErr: ErrServerNotFound,
		},
		{
			name: "test 3",
			fields: fields{
				Config: &config.Config{
					Mutex:                  &sync.Mutex{},
					TimeoutWaitPromoteSec:  4,
					FailoverTimeoutSec:     10,
					CheckMasterIntervalSec: 1,
					MinVerSQLPromote:       12,
					Listen:                 ":5051",
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
					},
				},
				transporter: &trMock{
					retOpen: map[string][]error{"one": {nil}, "two": {nil}, "three": {nil}},
					retIsRecovery: map[string][]struct {
						f   bool
						err error
					}{"one": {{false, nil}}, "two": {{true, nil}}, "three": {{true, nil}}},
					retHostVersion: map[string][]struct {
						v   float64
						err error
					}{"one": {{12, nil}}, "two": {{12, nil}}, "three": {{12, nil}}},
				},
			},
			wantErr: ErrTerminate,
		},
		{
			name: "test 4",
			fields: fields{
				Config: &config.Config{
					Mutex:                  &sync.Mutex{},
					TimeoutWaitPromoteSec:  4,
					FailoverTimeoutSec:     10,
					CheckMasterIntervalSec: 1,
					MinVerSQLPromote:       12,
					Listen:                 ":5051",
					ShardListen:            ":5052",
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
					},
				},
				transporter: &trMock{
					retOpen: map[string][]error{"one": {nil}, "two": {nil}, "three": {nil}},
					retIsRecovery: map[string][]struct {
						f   bool
						err error
					}{"one": {{false, nil}, {false, errFoo}}, "two": {{true, nil}, {false, nil}}, "three": {{true, nil}, {false, nil}}},
					retHostVersion: map[string][]struct {
						v   float64
						err error
					}{"one": {{12, nil}}, "two": {{12, nil}}, "three": {{12, nil}}},
				},
			},
			wantErr: ErrTerminate,
		},
		{
			name: "test 5",
			fields: fields{
				Config: &config.Config{
					Mutex:                  &sync.Mutex{},
					TimeoutWaitPromoteSec:  4,
					FailoverTimeoutSec:     10,
					CheckMasterIntervalSec: 1,
					MinVerSQLPromote:       12,
					Listen:                 ":5051",
					ShardListen:            ":5052",
					Servers: map[string]*config.Server{
						"one": {
							PgConn: "one",
							Use:    true,
						},
					},
				},
				transporter: &trMock{
					retOpen: map[string][]error{"one": {nil}, "two": {nil}, "three": {nil}},
					retIsRecovery: map[string][]struct {
						f   bool
						err error
					}{"one": {{false, nil}}, "two": {{true, nil}, {false, nil}}, "three": {{true, nil}, {false, nil}}},
					retHostVersion: map[string][]struct {
						v   float64
						err error
					}{"one": {{12, nil}}, "two": {{12, nil}}, "three": {{12, nil}}},
				},
			},
			wantErr: ErrTerminate,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			ch := make(chan struct{})
			s := NewFailover(tt.fields.Config, tt.fields.transporter, m{})
			doneCtx, cancel := context.WithCancel(context.Background())
			go func() {
				err = s.Start(doneCtx)
				ch <- struct{}{}
			}()
			time.Sleep(time.Second)
			cancel()
			<-ch
			if err != tt.wantErr {
				t.Errorf("Failover.start1() got = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestFailover_Start2(t *testing.T) {
	type fields struct {
		Config      *config.Config
		transporter transporter
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
					Mutex:                  &sync.Mutex{},
					TimeoutWaitPromoteSec:  4,
					FailoverTimeoutSec:     10,
					CheckMasterIntervalSec: 1,
					MinVerSQLPromote:       12,
					Listen:                 ":5051",
					ShardListen:            ":5052",
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
					},
				},
				transporter: &trMock{
					retOpen: map[string][]error{"one": {nil}, "two": {nil}, "three": {nil}},
					retIsRecovery: map[string][]struct {
						f   bool
						err error
					}{"one": {{false, nil}}, "two": {{true, nil}}, "three": {{true, nil}}},
					retHostVersion: map[string][]struct {
						v   float64
						err error
					}{"one": {{12, nil}}, "two": {{12, nil}}, "three": {{12, nil}}},
				},
			},
			wantErr: ErrServerNotFound,
		},
		{
			name: "test 2",
			fields: fields{
				Config: &config.Config{
					Mutex:                  &sync.Mutex{},
					TimeoutWaitPromoteSec:  4,
					FailoverTimeoutSec:     10,
					CheckMasterIntervalSec: 1,
					MinVerSQLPromote:       12,
					Listen:                 ":5052",
					ShardListen:            ":5051",
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
					},
				},
				transporter: &trMock{
					retOpen: map[string][]error{"one": {nil}, "two": {nil}, "three": {nil}},
					retIsRecovery: map[string][]struct {
						f   bool
						err error
					}{"one": {{false, nil}}, "two": {{true, nil}}, "three": {{true, nil}}},
					retHostVersion: map[string][]struct {
						v   float64
						err error
					}{"one": {{12, nil}}, "two": {{12, nil}}, "three": {{12, nil}}},
				},
			},
			wantErr: ErrTerminate,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			ch := make(chan struct{})
			net.Listen("tcp", ":5051")

			s := NewFailover(tt.fields.Config, tt.fields.transporter, m{})
			doneCtx, cancel := context.WithCancel(context.Background())
			go func() {
				err = s.Start(doneCtx)
				ch <- struct{}{}
			}()
			time.Sleep(time.Second)
			cancel()
			<-ch
			if err == nil {
				t.Errorf("Failover.start2() got = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestFailover_makePostPromoteCmd(t *testing.T) {
	type fields struct {
		Config *config.Config
	}
	type args struct {
		newMaster string
		host      string
		port      string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []string
	}{
		{
			name: "test 1",
			fields: fields{
				Config: &config.Config{
					PgUser: "postgres",
					Servers: map[string]*config.Server{
						"one": {
							Use: true,
						},
						"two": {
							Use:                true,
							PostPromoteCommand: "sh2 {{.Host}} {{.Port}} {{.PgUser}}",
						},
						"three": {
							Use:                true,
							PostPromoteCommand: "sh3 {{.Host}} {{.Port}} {{.PgUser}}",
						},
						"four": {
							Use:                false,
							PostPromoteCommand: "sh4 {{.Host}} {{.Port}} {{.PgUser}}",
						},
						"five": {
							Use:                true,
							PostPromoteCommand: "sh5 {{.Host}} {{.Port}} {{.PgUser}}",
						},
						"six": {
							Use:                true,
							PostPromoteCommand: "sh5 {{{.Host}} {{.Port}} {{.PgUser}}",
						},
						"seven": {
							Use:                true,
							PostPromoteCommand: "",
						},
					},
				},
			},
			args: args{
				newMaster: "two",
				host:      "1.1.1.2",
				port:      "1234",
			},
			want: []string{"sh3 1.1.1.2 1234 postgres", "sh5 1.1.1.2 1234 postgres"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Failover{
				Config: tt.fields.Config,
			}
			got := s.makePostPromoteCmd(tt.args.newMaster, tt.args.host, tt.args.port)
			if len(got) != len(tt.want) {
				t.Errorf("Failover.makePostPromoteCmds() got = %v, want %v", got, tt.want)
			}
			sort.Strings(tt.want)
			sort.Strings(got)
			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("Failover.makePostPromoteCmds() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFailover_makePostPromoteExec(t *testing.T) {
	type fields struct {
		Config      *config.Config
		transporter transporter
	}
	type args struct {
		newMaster string
		host      string
		port      string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []string
	}{
		{
			name: "test 1",
			fields: fields{
				Config: &config.Config{
					PgUser: "postgres",
					Servers: map[string]*config.Server{
						"one": {
							PgConn: "one",
							Use:    true,
						},
						"two": {
							PgConn: "two",
							Use:    false,
						},
						"three": {
							PgConn: "three",
							Use:    true,
						},
						"four": {
							PgConn: "four",
							Use:    true,
						},
						"five": {
							PgConn: "five",
							Use:    true,
						},
					},
				},
				transporter: &trMock{
					retOpen:              map[string][]error{"one": {nil}, "two": {nil}, "three": {errFoo}, "four": {nil}},
					retChangePrimaryConn: map[string][]error{"one": {nil}, "two": {nil}, "three": {nil}, "four": {errFoo}},
				},
			},
			args: args{
				newMaster: "five",
				host:      "1.1.1.2",
				port:      "1234",
			},
			want: []string{"one", "four"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Failover{
				Config:      tt.fields.Config,
				transporter: tt.fields.transporter,
			}
			got := s.makePostPromoteExec(tt.args.newMaster, tt.args.host, tt.args.port)
			if len(got) != len(tt.want) {
				t.Errorf("Failover.makePostPromoteExec() got = %v, want %v", got, tt.want)
			}
			sort.Strings(tt.want)
			sort.Strings(got)
			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("Failover.makePostPromoteExec() got = %v, want %v", got, tt.want)
			}
		})
	}
}
