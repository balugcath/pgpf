// +build integration_test
// see ../../../build/.drone.yml

package config

import (
	"reflect"
	"testing"
)

func TestConfig_NewConfigSave(t *testing.T) {
	tests := []struct {
		name        string
		configFile  string
		etcdAddress string
		etcdKey     string
		wantErr     bool
	}{
		{
			name:        "test 1",
			etcdAddress: "etcd_service:2379",
			etcdKey:     "pgpf",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s1, err := NewConfig(tt.configFile, tt.etcdAddress, tt.etcdKey)
			if err != nil {
				t.Fatalf("Config.NewConfig() error = %v", err)
			}
			s1.FailoverTimeout = 4
			if err := s1.Save(); (err != nil) != tt.wantErr {
				t.Fatalf("Config.Save() error = %v, wantErr %v", err, tt.wantErr)
			}
			s2, err := NewConfig(tt.configFile, tt.etcdAddress, tt.etcdKey)
			if err != nil {
				t.Fatalf("Config.NewConfig() error = %v", err)
			}
			if !reflect.DeepEqual(s1, s2) {
				t.Fatalf("NewConfig() = %v, want %v", s2, s1)
			}
		})
	}
}
