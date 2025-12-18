package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// PostgresStore implements the Persistence interface
type PostgresStore struct {
	db        *sql.DB
	tableName string
}

// NewPostgresStore initializes PostgreSQL storage.
// connStr: Connection string
// tablePrefix: Table prefix (defaults to "scanner_") -> Resulting table is prefix + "checkpoints"
func NewPostgresStore(connStr string, tablePrefix string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if tablePrefix == "" {
		tablePrefix = "scanner_"
	}
	tableName := tablePrefix + "checkpoints"

	store := &PostgresStore{
		db:        db,
		tableName: tableName,
	}
	
	if err := store.initTable(); err != nil {
		return nil, err
	}

	return store, nil
}

// initTable automatically creates the scan progress table
func (p *PostgresStore) initTable() error {
	query := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		task_key VARCHAR(255) PRIMARY KEY,
		block_height BIGINT NOT NULL,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`, p.tableName)
	_, err := p.db.Exec(query)
	return err
}

func (p *PostgresStore) LoadCursor(key string) (uint64, error) {
	var height uint64
	query := fmt.Sprintf("SELECT block_height FROM %s WHERE task_key = $1", p.tableName)
	err := p.db.QueryRow(query, key).Scan(&height)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return height, nil
}

func (p *PostgresStore) SaveCursor(key string, height uint64) error {
	// Upsert using Postgres ON CONFLICT syntax
	query := fmt.Sprintf(`
	INSERT INTO %s (task_key, block_height, updated_at)
	VALUES ($1, $2, NOW())
	ON CONFLICT (task_key) 
	DO UPDATE SET block_height = EXCLUDED.block_height, updated_at = NOW();
	`, p.tableName)
	_, err := p.db.Exec(query, key, height)
	return err
}

func (p *PostgresStore) Close() error {
	return p.db.Close()
}