package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"
)

// Simple helper to run Postgres scenarios and print resulting SQLSTATE codes.
// Usage examples:
//   go run ./tools/pg_scenarios --dsn "postgres://user:pass@localhost:5432/db?sslmode=disable" --scenario unique_violation
//   go run ./tools/pg_scenarios --dsn "postgres://bad:wrong@localhost:5432/db?sslmode=disable" --scenario bad_password

func main() {
	dsn := flag.String("dsn", "", "Postgres DSN, e.g., postgres://user:pass@localhost:5432/db?sslmode=disable")
	scenario := flag.String("scenario", "", "Scenario to run: bad_password|missing_db|permission_denied|unique_violation|statement_timeout")
	flag.Parse()

	if *scenario == "bad_password" {
		// For bad password, attempting to open with wrong creds will return an error before db.Ping
		runBadPassword(*dsn)
		return
	}

	if *dsn == "" {
		log.Fatal("--dsn is required for this scenario")
	}

	switch *scenario {
	case "missing_db":
		runMissingDB(*dsn)
	case "permission_denied":
		runPermissionDenied(*dsn)
	case "unique_violation":
		runUniqueViolation(*dsn)
	case "statement_timeout":
		runStatementTimeout(*dsn)
	default:
		log.Fatalf("unknown scenario: %s", *scenario)
	}
}

func runBadPassword(dsn string) {
	// Expect pq: password authentication failed for user ... (SQLSTATE 28P01)
	db, err := sql.Open("postgres", dsn)
	if err == nil {
		// Force a round-trip
		err = db.Ping()
	}
	report("bad_password", err)
}

func runMissingDB(dsn string) {
	db, err := sql.Open("postgres", dsn)
	if err == nil {
		err = db.Ping()
	}
	report("missing_db", err)
}

func runPermissionDenied(dsn string) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		report("permission_denied", err)
		return
	}
	defer db.Close()

	// Try selecting from a table likely to exist in public; if none, create temp and revoke in advance in your env.
	_, err = db.Exec("SELECT * FROM information_schema.tables WHERE table_schema='restricted_schema'")
	report("permission_denied", err)
}

func runUniqueViolation(dsn string) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		report("unique_violation", err)
		return
	}
	defer db.Close()

	// Prepare a temp table with a unique constraint and violate it
	_, err = db.Exec(`
        CREATE TEMP TABLE IF NOT EXISTS t_unique(id INT PRIMARY KEY);
        INSERT INTO t_unique(id) VALUES (1);
        INSERT INTO t_unique(id) VALUES (1);
    `)
	report("unique_violation", err)
}

func runStatementTimeout(dsn string) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		report("statement_timeout", err)
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Set a small statement_timeout and run a long sleep
	_, err = db.ExecContext(ctx, `SET statement_timeout = '500ms'`)
	if err == nil {
		_, err = db.ExecContext(ctx, `SELECT pg_sleep(5)`) // expect 57014
	}
	report("statement_timeout", err)
}

func report(name string, err error) {
	if err == nil {
		fmt.Printf("%s: OK (no error)\n", name)
		return
	}
	// Try to extract SQLSTATE from lib/pq errors
	var pqErr interface {
		Code() string
		Error() string
	}
	if errors.As(err, &pqErr) {
		fmt.Printf("%s: error code=%s msg=%s\n", name, pqErr.Code(), pqErr.Error())
		return
	}
	fmt.Printf("%s: error=%v\n", name, err)
}
