package transport

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	//
	_ "github.com/lib/pq"
)

const (
	timeout = 4
)

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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*timeout)
	defer cancel()
	if err := s.db.QueryRowContext(ctx, "show server_version").Scan(&ver); err != nil {
		return 0, err
	}
	return strconv.ParseFloat(strings.Split(ver, " ")[0], 32)
}

// IsRecovery ...
func (s *PG) IsRecovery() (bool, error) {
	var isRecovery bool
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*timeout)
	err := s.db.QueryRowContext(ctx, "select pg_is_in_recovery()").Scan(&isRecovery)
	defer cancel()
	return isRecovery, err
}

// HostStatus ...
func (s *PG) HostStatus(pgConn string) (bool, float64, error) {
	if err := s.Open(pgConn); err != nil {
		return false, 0, err
	}
	defer s.Close()
	return s.hostStatus()
}

// Promote ...
func (s *PG) Promote(pgConn string) error {
	if err := s.Open(pgConn); err != nil {
		return err
	}
	defer s.Close()
	return s.promote()
}

func (s *PG) promote() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*timeout)
	defer cancel()
	_, err := s.db.ExecContext(ctx, "select pg_promote(false)")
	return err
}

func (s *PG) hostStatus() (bool, float64, error) {
	isRecovery, err := s.IsRecovery()
	if err != nil {
		return false, 0, err
	}
	ver, err := s.HostVersion()
	return isRecovery, ver, err
}
