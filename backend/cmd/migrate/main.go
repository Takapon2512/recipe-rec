package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var (
		path      = flag.String("path", "migrations", "migration files directory (migrations or seeds)")
		direction = flag.String("direction", "up", "up or down")
		steps     = flag.Int("steps", 0, "number of steps (0 = all)")
		table     = flag.String("table", "", "schema_migrations table name (default: schema_migrations, seeds: schema_seeds)")
	)
	flag.Parse()

	if *table == "" {
		if *path == "seeds" {
			*table = "schema_seeds"
		} else {
			*table = "schema_migrations"
		}
	}

	dsn := buildDSN(*table)
	m, err := migrate.New("file://"+*path, "mysql://"+dsn)
	if err != nil {
		log.Fatalf("migrate.New: %v", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			log.Printf("migrate close source: %v", srcErr)
		}
		if dbErr != nil {
			log.Printf("migrate close db: %v", dbErr)
		}
	}()

	if err := run(m, *direction, *steps); err != nil {
		log.Fatalf("migrate %s: %v", *direction, err)
	}

	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		log.Fatalf("migrate.Version: %v", err)
	}
	fmt.Printf("done: version=%d dirty=%v\n", version, dirty)
}

func run(m *migrate.Migrate, direction string, steps int) error {
	switch direction {
	case "up":
		if steps > 0 {
			return m.Steps(steps)
		}
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
	case "down":
		if steps > 0 {
			return m.Steps(-steps)
		}
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
	default:
		return fmt.Errorf("unknown direction: %s (use up or down)", direction)
	}
	return nil
}

func buildDSN(table string) string {
	host := getenv("DB_HOST", "localhost")
	port := getenv("DB_PORT", "3306")
	user := getenv("DB_USER", "app")
	password := os.Getenv("DB_PASSWORD")
	// パスワードはハードコードから除外
	if password == "" {
		log.Fatal("DB_PASSWORD environment variable is required")
	}
	dbname := getenv("DB_NAME", "app_db")
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?multiStatements=true&parseTime=true&x-migrations-table=%s",
		user, password, host, port, dbname, table)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
