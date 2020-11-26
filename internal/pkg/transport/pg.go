package transport

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	//
	_ "github.com/lib/pq"
)

var queryTimeout = 2 * time.Second

// PG ...
type PG struct {
	db *sql.DB
}

// Open ...
func (s *PG) Open(pgConn string) error {
	db, err := sql.Open("postgres", pgConn)
	s.db = db
	return err
}

// Close ...
func (s *PG) Close() {
	s.db.Close()
}

// HostVersion ...
func (s *PG) HostVersion() (float64, error) {
	var ver string
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	if err := s.db.QueryRowContext(ctx, "show server_version").Scan(&ver); err != nil {
		return 0, err
	}
	return strconv.ParseFloat(strings.Split(ver, " ")[0], 32)
}

// IsRecovery ...
func (s *PG) IsRecovery() (bool, error) {
	var isRecovery bool
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	err := s.db.QueryRowContext(ctx, "select pg_is_in_recovery()").Scan(&isRecovery)
	defer cancel()
	return isRecovery, err
}

// Promote ...
func (s *PG) Promote() error {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	_, err := s.db.ExecContext(ctx, "select pg_promote(false)")
	return err
}

// ChangePrimaryConn ...
func (s *PG) ChangePrimaryConn(host, user, port string) error {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	_, err := s.db.ExecContext(ctx, "alter system set primary_conninfo to "+
		fmt.Sprintf("'user=%s host=%s port=%s sslmode=prefer sslcompression=1 target_session_attrs=any'", user, host, port))
	return err
}
