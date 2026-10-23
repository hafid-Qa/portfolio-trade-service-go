package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	"app/config"
	"app/internal/sqlc"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

var (
	testDB    *sql.DB
	testStore sqlc.Store
)

// TestMain runs migrations against the test database itself (rather than
// assuming they were applied beforehand) and truncates afterward, so every
// test run starts from a known-clean schema and no data -- including the
// very first run against a fresh db_test container with no schema at all.
func TestMain(m *testing.M) {
	ctx := context.Background()
	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}

	testDB, err = sql.Open(cfg.DBDriver, cfg.TestDBUrl)
	if err != nil {
		log.Fatalf("cannot connect to test db: %v", err)
	}
	testStore = *sqlc.NewStore(testDB)

	runMigrations(cfg.TestDBUrl)
	truncateAllTables(testDB)

	os.Exit(m.Run())
}

// seedUser inserts a minimal users row for userID, needed since portfolios
// and trades now carry a foreign key to users -- ON CONFLICT DO NOTHING so
// callers seeding multiple trades/portfolios for the same userID don't collide.
func seedUser(t *testing.T, userID int64) {
	t.Helper()
	_, err := testDB.Exec(
		`INSERT INTO users (id, full_name, username, email, password_hash)
		 VALUES ($1, 'Test User', $2, $3, 'x') ON CONFLICT (id) DO NOTHING`,
		userID, fmt.Sprintf("test_user_%d", userID), fmt.Sprintf("test%d@example.com", userID),
	)
	if err != nil {
		t.Fatalf("seeding user %d: %v", userID, err)
	}
}

func runMigrations(dbURL string) {
	m, err := migrate.New("file://../../sqlc/migrations", dbURL)
	if err != nil {
		log.Fatalf("cannot create migrate instance: %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("cannot apply migrations: %v", err)
	}
}

func truncateAllTables(db *sql.DB) {
	const q = `
DO $$
DECLARE r record;
BEGIN
  FOR r IN
    (SELECT tablename FROM pg_tables
     WHERE schemaname='public' AND tablename <> 'schema_migrations')
  LOOP
    EXECUTE 'TRUNCATE TABLE ' || quote_ident(r.tablename) || ' RESTART IDENTITY CASCADE';
  END LOOP;
END $$;`
	if _, err := db.Exec(q); err != nil {
		log.Fatalf("cannot truncate tables: %v", err)
	}
}
