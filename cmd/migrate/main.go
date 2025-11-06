package main

import (
	"bufio"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

const migrationsDir = "migrations"

type migrationFile struct {
	version int
	name    string
	path    string
	kind    string // up or down
}

func main() {
	mode := flag.String("mode", "up", "migration mode: up or down")
	flag.Parse()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	if err := ensureSchemaMigrations(db); err != nil {
		log.Fatalf("failed to ensure schema_migrations: %v", err)
	}

	files, err := loadMigrationFiles(migrationsDir)
	if err != nil {
		log.Fatalf("failed to load migrations: %v", err)
	}

	switch strings.ToLower(*mode) {
	case "up":
		if err := applyUp(db, files); err != nil {
			log.Fatalf("migration up failed: %v", err)
		}
		log.Println("Migration up completed successfully")
	case "down":
		if err := applyDown(db, files); err != nil {
			log.Fatalf("migration down failed: %v", err)
		}
		log.Println("Migration down completed successfully")
	default:
		log.Fatalf("unknown mode: %s", *mode)
	}
}

func ensureSchemaMigrations(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	return err
}

func loadMigrationFiles(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []migrationFile
	for _, e := range entries {
		if e.IsDir() { continue }
		name := e.Name()
		lower := strings.ToLower(name)
		if !strings.HasSuffix(lower, ".sql") { continue }

		kind := "up"
		if strings.HasSuffix(lower, ".down.sql") {
			kind = "down"
		} else if strings.HasSuffix(lower, ".up.sql") {
			kind = "up"
		}

		ver, migName, err := parseVersionAndName(name)
		if err != nil {
			// skip files without numeric prefix
			log.Printf("skip migration without version prefix: %s", name)
			continue
		}

		files = append(files, migrationFile{
			version: ver,
			name:    migName,
			path:    filepath.Join(dir, name),
			kind:    kind,
		})
	}

	// sort by version asc for up, desc for down will be handled later
	sort.Slice(files, func(i, j int) bool { return files[i].version < files[j].version })
	return files, nil
}

func parseVersionAndName(filename string) (int, string, error) {
	// expected: 001_create_users_table.up.sql or 001_create_auth_tables.sql
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) < 2 { return 0, "", errors.New("invalid filename") }
	verStr := parts[0]
	var ver int
	for _, r := range verStr {
		if r < '0' || r > '9' { return 0, "", errors.New("invalid version") }
	}
	fmt.Sscanf(verStr, "%d", &ver)
	return ver, parts[1], nil
}

func alreadyApplied(db *sql.DB, version int) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)", version).Scan(&exists)
	return exists, err
}

func markApplied(db *sql.DB, version int, name string) error {
	_, err := db.Exec("INSERT INTO schema_migrations(version, name, applied_at) VALUES($1,$2,$3)", version, name, time.Now())
	return err
}

func unmarkApplied(db *sql.DB, version int) error {
	_, err := db.Exec("DELETE FROM schema_migrations WHERE version=$1", version)
	return err
}

func applyUp(db *sql.DB, files []migrationFile) error {
	for _, f := range files {
		if f.kind != "up" { continue }
		applied, err := alreadyApplied(db, f.version)
		if err != nil { return err }
		if applied { continue }

		log.Printf("Applying up %03d: %s", f.version, f.name)
		if err := execSQLFile(db, f.path); err != nil {
			return fmt.Errorf("failed applying %s: %w", f.path, err)
		}
		if err := markApplied(db, f.version, f.name); err != nil { return err }
	}
	return nil
}

func applyDown(db *sql.DB, files []migrationFile) error {
	// collect down files and sort desc
	var downs []migrationFile
	for _, f := range files { if f.kind == "down" { downs = append(downs, f) } }
	sort.Slice(downs, func(i, j int) bool { return downs[i].version > downs[j].version })

	for _, f := range downs {
		applied, err := alreadyApplied(db, f.version)
		if err != nil { return err }
		if !applied { continue }

		log.Printf("Reverting down %03d: %s", f.version, f.name)
		if err := execSQLFile(db, f.path); err != nil {
			return fmt.Errorf("failed reverting %s: %w", f.path, err)
		}
		if err := unmarkApplied(db, f.version); err != nil { return err }
	}
	return nil
}

func execSQLFile(db *sql.DB, path string) error {
	f, err := os.Open(path)
	if err != nil { return err }
	defer f.Close()
	var b strings.Builder
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for s.Scan() {
		b.WriteString(s.Text())
		b.WriteString("\n")
	}
	if err := s.Err(); err != nil { return err }
	_, err = db.Exec(b.String())
	return err
}