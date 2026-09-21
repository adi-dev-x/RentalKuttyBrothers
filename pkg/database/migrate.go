package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"myproject/pkg/config"

	_ "github.com/lib/pq"
)

// schemaFiles are applied, in this order, every time the app boots.
//
// schema.sql creates the full schema on a fresh database (CREATE ... IF NOT
// EXISTS everywhere, seed rows use ON CONFLICT DO NOTHING), so on an existing
// database it just no-ops on tables that already exist.
//
// migration_product_code.sql upgrades an existing (pre product_code) database
// — ADD COLUMN IF NOT EXISTS / DROP COLUMN IF EXISTS — so once a database is
// already current it also no-ops.
//
// migration_transaction_invoice.sql drops transaction's relation to
// main_transaction and adds invoice_id/from_date/to_date/invoice_type — same
// ADD/DROP ... IF [NOT] EXISTS idempotency.
//
// Running all of these on every `go run`/`go build` startup is therefore
// safe against a brand-new database, an already-migrated one, and an old
// production one.
var schemaFiles = []string{
	"schema.sql",
	"migration_product_code.sql",
	"migration_transaction_invoice.sql",
}

// RunSchemaMigrations applies schemaFiles against the configured database.
// It opens its own short-lived connection via lib/pq (registered under the
// "postgres" driver name) rather than reusing the gorm pool, because lib/pq
// supports running a whole ;-separated SQL file in one Exec call — each file
// runs as a single implicit transaction, so a failing statement rolls back
// that file's changes instead of leaving it half-applied.
func RunSchemaMigrations(cnf config.Config) error {
	dsn := fmt.Sprintf(
		"host=%v user=%v password=%v dbname=%v port=%v sslmode=%v",
		cnf.PGHost, cnf.PGUserName, cnf.PGPassword, cnf.PGDBName, cnf.PgPort, cnf.PgSSLMode,
	)

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("migrate: failed to open connection: %w", err)
	}
	defer conn.Close()

	if err := conn.Ping(); err != nil {
		return fmt.Errorf("migrate: failed to reach database: %w", err)
	}

	for _, file := range schemaFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("migrate: failed to read %s: %w", file, err)
		}

		if _, err := conn.Exec(string(content)); err != nil {
			return fmt.Errorf("migrate: failed to apply %s: %w", file, err)
		}

		log.Printf("migrate: applied %s", file)
	}

	return nil
}
