package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/albertwidi/go_project_example/internal/pkg/log/logger"
	"github.com/jmoiron/sqlx"
)

// list of error
var (
	errConfigNil = errors.New("sqldb: config is nil")
)

// DB struct to hold all database connections
type DB struct {
	driver   string
	leader   *sqlx.DB
	follower *sqlx.DB

	logger logger.Logger
}

// New sqldb wrapper object
func New(ctx context.Context, leader, follower *sqlx.DB) (*DB, error) {
	if leader.DriverName() != follower.DriverName() {
		return nil, fmt.Errorf("sqldb: leader and follower driver is not match. leader = %s follower = %s", leader.DriverName(), follower.DriverName())
	}

	db := DB{
		driver:   leader.DriverName(),
		leader:   leader,
		follower: follower,
	}

	return &db, nil
}

// ConnectOptions to list options when connect to the db
type ConnectOptions struct {
	Retry                 int
	MaxOpenConnections    int
	MaxIdleConnections    int
	ConnectionMaxLifetime time.Duration
}

// Connect to a new database
func Connect(ctx context.Context, driver, dsn string, connOpts *ConnectOptions) (*sqlx.DB, error) {
	opts := connOpts
	if opts == nil {
		opts = &ConnectOptions{}
	}

	db, err := connectWithRetry(ctx, driver, dsn, opts.Retry)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(opts.MaxOpenConnections)
	db.SetMaxIdleConns(opts.MaxIdleConnections)
	db.SetConnMaxLifetime(opts.ConnectionMaxLifetime)
	return db, nil
}

func connect(ctx context.Context, driver, dsn string) (*sqlx.DB, error) {
	sqlxdb, err := sqlx.ConnectContext(ctx, driver, dsn)
	if err != nil {
		return nil, err
	}
	return sqlxdb, err
}

func connectWithRetry(ctx context.Context, driver, dsn string, retry int) (*sqlx.DB, error) {
	var (
		sqlxdb *sqlx.DB
		err    error
	)

	if retry == 0 {
		sqlxdb, err = connect(ctx, driver, dsn)
		return nil, err
	}

	for x := 0; x < retry; x++ {
		sqlxdb, err = connect(ctx, driver, dsn)
		if err == nil {
			break
		}

		// db.logger.Warnf("sqldb: failed to connect to %s with error %s", dsn, err.Error())
		// db.logger.Warnf("sqldb: retrying to connect to %s. Retry: %d", dsn, x+1)

		if x+1 == retry && err != nil {
			// db.logger.Errorf("sqldb: retry time exhausted, cannot connect to database: %s", err.Error())
			return nil, fmt.Errorf("sqldb: failed connect to database: %s", err.Error())
		}
		time.Sleep(time.Second * 3)
	}
	return sqlxdb, err
}

// Leader return leader database connection
func (db *DB) Leader() *sqlx.DB {
	return db.leader
}

// Follower return follower database connection
func (db *DB) Follower() *sqlx.DB {
	return db.follower
}

// SetMaxIdleConns to sql database
func (db *DB) SetMaxIdleConns(n int) {
	db.Leader().SetMaxIdleConns(n)
	db.Follower().SetMaxIdleConns(n)
}

// SetMaxOpenConns to sql database
func (db *DB) SetMaxOpenConns(n int) {
	db.Leader().SetMaxOpenConns(n)
	db.Follower().SetMaxOpenConns(n)
}

// SetConnMaxLifetime to sql database
func (db *DB) SetConnMaxLifetime(t time.Duration) {
	db.Leader().SetConnMaxLifetime(t)
	db.Follower().SetConnMaxLifetime(t)
}

// Get return one value in destination using relfection
func (db *DB) Get(dest interface{}, query string, args ...interface{}) error {
	return db.follower.Get(dest, query, args...)
}

// Select return more than one value in destintion using reflection
func (db *DB) Select(dest interface{}, query string, args ...interface{}) error {
	return db.follower.Select(dest, query, args...)
}

// Query function
func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.follower.Query(query, args...)
}

// NamedQuery function
func (db *DB) NamedQuery(query string, arg interface{}) (*sqlx.Rows, error) {
	return db.follower.NamedQuery(query, arg)
}

// QueryRow function
func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.follower.QueryRow(query, args...)
}

// Exec function
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.leader.Exec(query, args...)
}

// NamedExec execute query with named parameter
func (db *DB) NamedExec(query string, arg interface{}) (sql.Result, error) {
	return db.leader.NamedExec(query, arg)
}

// Begin return sql transaction object, begin a transaction
func (db *DB) Begin() (*sql.Tx, error) {
	return db.leader.Begin()
}

// Beginx return sqlx transaction object, begin a transaction
func (db *DB) Beginx() (*sqlx.Tx, error) {
	return db.leader.Beginx()
}

// Rebind query
func (db *DB) Rebind(query string) string {
	return sqlx.Rebind(sqlx.BindType(db.driver), query)
}

// Named return named query and parameters
func (db *DB) Named(query string, arg interface{}) (string, interface{}, error) {
	return sqlx.Named(query, arg)
}

// BindNamed return named query wrapped with bind
func (db *DB) BindNamed(query string, arg interface{}) (string, interface{}, error) {
	return sqlx.BindNamed(sqlx.BindType(db.driver), query, arg)
}