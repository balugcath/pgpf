// +build integration_test
// see ../../../build/.drone.yml

package transport

import (
	"testing"
)

const (
	dbConn = "user=postgres password=123 dbname=postgres host=postgres_service sslmode=disable"
)

func TestPG_HostVersion1(t *testing.T) {
	tests := []struct {
		name    string
		want    float64
		wantErr bool
		err     error
	}{
		{
			name:    "test 1",
			wantErr: false,
			want:    12,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			s := new(PG)
			err := s.Open(dbConn)
			if err != nil {
				t.Fatalf("PG.HostVersion1() got error = %v", err)
			}
			defer s.Close()

			got, err := s.HostVersion()
			if (err != nil) != tt.wantErr {
				t.Errorf("PG.HostVersion1() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want > got {
				t.Errorf("PG.HostVersion1() = %v, want %v", got, tt.want)
			}
		})
	}
}
