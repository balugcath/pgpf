package transport

import (
	"errors"
	"fmt"
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

			if err := s.Promote(); (err != nil) != tt.wantErr {
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
		arg     string
	}{
		{
			name:    "test 1",
			wantErr: false,
			want:    11,
			arg:     "11",
		},
		{
			name:    "test 2",
			wantErr: true,
			want:    0,
			err:     errors.New("err"),
		},
		{
			name:    "test 3",
			wantErr: false,
			want:    11,
			arg:     "11 12",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()
			mock.ExpectQuery("show server_version").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(tt.arg)).WillReturnError(tt.err)

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

func TestPG_ChangePrimaryConn(t *testing.T) {
	type args struct {
		host string
		port string
		user string
	}
	tests := []struct {
		name    string
		wantErr bool
		args    args
	}{
		{
			name:    "test 1",
			wantErr: false,
			args: args{
				host: "1",
				port: "2",
				user: "3",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer db.Close()

			mock.ExpectExec("alter system set primary_conninfo to " +
				fmt.Sprintf("'user=%s host=%s port=%s sslmode=prefer sslcompression=1 target_session_attrs=any'",
					tt.args.user, tt.args.host, tt.args.port)).WillReturnResult(sqlmock.NewResult(0, 0))

			s := &PG{db: db}

			if err := s.ChangePrimaryConn(tt.args.host, tt.args.user, tt.args.port); (err != nil) != tt.wantErr {
				t.Errorf("PG.ChangePrimaryConn() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}

		})
	}
}
