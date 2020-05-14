package transport

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/lib/pq"
)

func TestPG_Promote(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "test 1",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()
			mock.ExpectExec("select pg_promote\\(false\\)").WillReturnResult(sqlmock.NewResult(1, 1))

			s := &PG{db: db}

			if err := s.promote(); (err != nil) != tt.wantErr {
				t.Errorf("PG.Promote() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}

		})
	}
}

func TestPG_IsRecovery(t *testing.T) {
	tests := []struct {
		name    string
		want    bool
		wantErr bool
	}{
		{
			name:    "test 1",
			wantErr: false,
			want:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()
			mock.ExpectQuery("select pg_is_in_recovery\\(\\)").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(true))

			s := &PG{db: db}

			got, err := s.IsRecovery()
			if (err != nil) != tt.wantErr {
				t.Errorf("PG.IsRecovery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PG.IsRecovery() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPG_HostVersion(t *testing.T) {
	tests := []struct {
		name    string
		want    float64
		wantErr bool
		err     error
	}{
		{
			name:    "test 1",
			wantErr: false,
			want:    11,
		},
		{
			name:    "test 2",
			wantErr: true,
			want:    0,
			err:     errors.New("err"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()
			mock.ExpectQuery("show server_version").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(11)).WillReturnError(tt.err)

			s := &PG{db: db}

			got, err := s.HostVersion()
			if (err != nil) != tt.wantErr {
				t.Errorf("PG.HostVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PG.HostVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPG_hostStatus(t *testing.T) {
	tests := []struct {
		name    string
		want    bool
		want1   float64
		wantErr bool
		err     error
	}{
		{
			name:    "test 1",
			want:    true,
			want1:   11,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()
			mock.ExpectQuery("select pg_is_in_recovery\\(\\)").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(true))
			mock.ExpectQuery("show server_version").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(11)).WillReturnError(tt.err)

			s := &PG{db: db}

			got, got1, err := s.hostStatus()
			if (err != nil) != tt.wantErr {
				t.Errorf("PG.hostStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PG.hostStatus() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("PG.hostStatus() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
