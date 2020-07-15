package transport

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMock_Open(t *testing.T) {
	type fields struct {
		FIsRecovery func() (bool, error)
		FOpen       func(string) error
		FPromote    func(string) error
		FHostStatus func(string, bool) (bool, float64, error)
		PromoteDone map[string]bool
	}
	type args struct {
		host string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test1",
			fields: fields{
				FOpen: func(string) error {
					return nil
				},
			},
			wantErr: false,
			args:    args{host: ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Mock{
				FIsRecovery: tt.fields.FIsRecovery,
				FOpen:       tt.fields.FOpen,
				FPromote:    tt.fields.FPromote,
				FHostStatus: tt.fields.FHostStatus,
				PromoteDone: tt.fields.PromoteDone,
			}
			if err := s.Open(tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("Mock.Open() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMock_Promote(t *testing.T) {
	type fields struct {
		FIsRecovery func() (bool, error)
		FOpen       func(string) error
		FPromote    func(string) error
		FHostStatus func(string, bool) (bool, float64, error)
		PromoteDone map[string]bool
	}
	type args struct {
		host string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test1",
			fields: fields{
				FPromote: func(string) error {
					return nil
				},
				PromoteDone: make(map[string]bool),
			},
			wantErr: false,
			args:    args{host: "one"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Mock{
				FIsRecovery: tt.fields.FIsRecovery,
				FOpen:       tt.fields.FOpen,
				FPromote:    tt.fields.FPromote,
				FHostStatus: tt.fields.FHostStatus,
				PromoteDone: tt.fields.PromoteDone,
			}
			if err := s.Promote(tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("Mock.Promote() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !assert.Equal(t, s.PromoteDone, map[string]bool{"one": true}) {
				t.Errorf("Mock.Promote() got = %v, want %v", s.PromoteDone, map[string]bool{"one": true})
			}

		})
	}
}

func TestMock_HostStatus(t *testing.T) {
	type fields struct {
		FIsRecovery func() (bool, error)
		FOpen       func(string) error
		FPromote    func(string) error
		FHostStatus func(string, bool) (bool, float64, error)
		PromoteDone map[string]bool
	}
	type args struct {
		host string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		want1   float64
		wantErr bool
	}{
		{
			name: "test1",
			fields: fields{
				FPromote: func(string) error {
					return nil
				},
				PromoteDone: make(map[string]bool),
				FHostStatus: func(_ string, isPromote bool) (bool, float64, error) {
					return isPromote, 10, nil
				},
			},
			wantErr: false,
			args:    args{host: "one"},
			want1:   10,
			want:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Mock{
				FIsRecovery: tt.fields.FIsRecovery,
				FOpen:       tt.fields.FOpen,
				FPromote:    tt.fields.FPromote,
				FHostStatus: tt.fields.FHostStatus,
				PromoteDone: tt.fields.PromoteDone,
			}
			if err := s.Promote(tt.args.host); (err != nil) != tt.wantErr {
				t.Errorf("Mock.Promote() error = %v, wantErr %v", err, tt.wantErr)
			}

			got, got1, err := s.HostStatus(tt.args.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("Mock.HostStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Mock.HostStatus() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Mock.HostStatus() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestMock_IsRecovery(t *testing.T) {
	type fields struct {
		FIsRecovery func() (bool, error)
		FOpen       func(string) error
		FPromote    func(string) error
		FHostStatus func(string, bool) (bool, float64, error)
		PromoteDone map[string]bool
	}
	tests := []struct {
		name    string
		fields  fields
		want    bool
		wantErr bool
	}{
		{
			name: "test1",
			fields: fields{
				FIsRecovery: func() (bool, error) {
					return true, nil
				},
			},
			wantErr: false,
			want:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Mock{
				FIsRecovery: tt.fields.FIsRecovery,
				FOpen:       tt.fields.FOpen,
				FPromote:    tt.fields.FPromote,
				FHostStatus: tt.fields.FHostStatus,
				PromoteDone: tt.fields.PromoteDone,
			}
			got, err := s.IsRecovery()
			if (err != nil) != tt.wantErr {
				t.Errorf("Mock.IsRecovery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Mock.IsRecovery() = %v, want %v", got, tt.want)
			}
		})
	}
}
