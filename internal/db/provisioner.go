package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type Provisioner struct {
	host     string
	port     int
	user     string
	password string
	adminDB  string
}

func NewProvisioner(host string, port int, user, pass, adminDB string) *Provisioner {
	return &Provisioner{
		host:     host,
		port:     port,
		user:     user,
		password: pass,
		adminDB:  adminDB,
	}
}

func (p *Provisioner) connect() (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		p.host, p.port, p.user, p.password, p.adminDB)
	return sql.Open("postgres", connStr)
}

func (p *Provisioner) CreateDatabase(dbName string) error {
	db, err := p.connect()
	if err != nil {
		return fmt.Errorf("connection failed: %v", err)
	}
	defer db.Close()

	// Check if database exists
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)", dbName).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("database %s already exists", dbName)
	}

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	return err
}

func (p *Provisioner) DropDatabase(dbName string) error {
	db, err := p.connect()
	if err != nil {
		return err
	}
	defer db.Close()

	// Terminate existing connections
	_, err = db.Exec(fmt.Sprintf(`
        SELECT pg_terminate_backend(pg_stat_activity.pid)
        FROM pg_stat_activity
        WHERE pg_stat_activity.datname = '%s'
        AND pid <> pg_backend_pid()`, dbName))
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	return err
}

func (p *Provisioner) TestConnection(dbName string) error {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		p.host, p.port, p.user, p.password, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Ping()
}
