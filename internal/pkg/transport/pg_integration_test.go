// +build integration_test
// see ../../../build/.drone.yml

package transport

import (
	"testing"
)

func init() {
	timeout = 100000000
}

func TestPG_HostStatus(t *testing.T) {
	tests := []struct {
		name  string
		want1 bool
		want2 float64
		want3 error
	}{
		{
			name:  "test 1",
			want1: false,
			want2: 12,
			want3: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got1, got2, got3 := new(PG).HostStatus("user=postgres password=123 dbname=postgres_service host=postgres sslmode=disable")
			if got1 != tt.want1 {
				t.Errorf("PG.HostStatus() = %v, want %v", got1, tt.want1)
			}
			if got2 < tt.want2 {
				t.Errorf("PG.HostStatus() = %v, want %v", got2, tt.want2)
			}
			if got3 != tt.want3 {
				t.Errorf("PG.HostStatus() = %v, want %v", got3, tt.want3)
			}
		})
	}
}
