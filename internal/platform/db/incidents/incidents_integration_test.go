//go:build integration

package incidentsdb

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/m1ll3r1337/geo-notifications-service/internal/domain/incidents"
	"github.com/m1ll3r1337/geo-notifications-service/internal/errs"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/db"
)

var (
	testDB      *sqlx.DB
	testDBURL   string
	terminateFn func(context.Context) error
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var err error
	testDB, testDBURL, terminateFn, err = setupDB(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "integration setup failed:", err)
		os.Exit(1)
	}

	code := m.Run()

	_ = testDB.Close()

	if terminateFn != nil {
		tdCtx, tdCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer tdCancel()
		_ = terminateFn(tdCtx)
	}

	os.Exit(code)
}

func setupDB(ctx context.Context) (*sqlx.DB, string, func(context.Context) error, error) {
	if dsn := os.Getenv("TEST_DB_URL"); dsn != "" {
		dbx, err := db.Open(ctx, db.Config{URL: dsn, PingTimeout: 10 * time.Second})
		if err != nil {
			return nil, "", nil, err
		}
		if err := applyMigrations(dsn); err != nil {
			_ = dbx.Close()
			return nil, "", nil, err
		}
		if err := db.StatusCheck(ctx, dbx); err != nil {
			_ = dbx.Close()
			return nil, "", nil, err
		}

		return dbx, dsn, nil, nil
	}

	req := testcontainers.ContainerRequest{
		Image:        "postgis/postgis:16-3.4",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "postgres",
			"POSTGRES_USER":     "postgres",
			"POSTGRES_DB":       "geo_test",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(90 * time.Second),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", nil, fmt.Errorf("start container: %w", err)
	}

	host, err := c.Host(ctx)
	if err != nil {
		_ = c.Terminate(context.Background())
		return nil, "", nil, fmt.Errorf("container host: %w", err)
	}
	port, err := c.MappedPort(ctx, "5432/tcp")
	if err != nil {
		_ = c.Terminate(context.Background())
		return nil, "", nil, fmt.Errorf("container port: %w", err)
	}

	dsn := fmt.Sprintf("postgres://postgres:postgres@%s:%s/geo_test?sslmode=disable", host, port.Port())

	dbx, err := db.Open(ctx, db.Config{URL: dsn, PingTimeout: 15 * time.Second})
	if err != nil {
		_ = c.Terminate(context.Background())
		return nil, "", nil, err
	}

	if err := db.StatusCheck(ctx, dbx); err != nil {
		_ = dbx.Close()
		_ = c.Terminate(context.Background())
		return nil, "", nil, err
	}

	if err := applyMigrations(dsn); err != nil {
		_ = dbx.Close()
		_ = c.Terminate(context.Background())
		return nil, "", nil, err
	}

	terminateFn := func(ctx context.Context) error {
		return c.Terminate(ctx)
	}
	return dbx, dsn, terminateFn, nil
}

func applyMigrations(dbURL string) error {
	migrationsDir, err := findMigrationsDir()
	if err != nil {
		return err
	}

	srcURL := "file://" + filepath.ToSlash(migrationsDir)

	m, err := migrate.New(srcURL, dbURL)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

func findMigrationsDir() (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}

	dir := filepath.Dir(thisFile)
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "..", "..", "..", "..", "migrations")
		candidate = filepath.Clean(candidate)

		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate, nil
		}

		dir = filepath.Dir(dir)
	}
	return "", fmt.Errorf("migrations dir not found")
}

func withTx(t *testing.T) (context.Context, *sqlx.Tx) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	tx, err := testDB.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	return ctx, tx
}

func TestRepository_Create_And_GetByID(t *testing.T) {
	ctx, tx := withTx(t)
	repo := New(tx)

	created, err := repo.Create(ctx, incidents.CreateIncident{
		Title:       "test",
		Description: "desc",
		Center:      incidents.Point{Lat: 55.75, Lon: 37.61},
		Radius:      500,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID <= 0 {
		t.Fatalf("expected ID > 0, got %d", created.ID)
	}

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != created.ID || got.Title != created.Title || got.Radius != created.Radius {
		t.Fatalf("mismatch: got=%+v created=%+v", got, created)
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	ctx, tx := withTx(t)
	repo := New(tx)

	_, err := repo.GetByID(ctx, 999999)
	if err == nil {
		t.Fatalf("expected error")
	}

	e, ok := errs.As(err)
	if !ok || e.Kind != errs.KindNotFound {
		t.Fatalf("expected kind=%s, got %T: %v", errs.KindNotFound, err, err)
	}
}

func TestRepository_Create_CheckViolation_MappedToInvalid(t *testing.T) {
	ctx, tx := withTx(t)
	repo := New(tx)

	_, err := repo.Create(ctx, incidents.CreateIncident{
		Title:       "bad-radius",
		Description: "",
		Center:      incidents.Point{Lat: 10, Lon: 10},
		Radius:      0,
	})
	if err == nil {
		t.Fatalf("expected error")
	}

	e, ok := errs.As(err)
	if !ok {
		t.Fatalf("expected *errs.Error, got %T: %v", err, err)
	}
	if e.Kind != errs.KindInvalid {
		t.Fatalf("expected kind=%s, got %s (code=%s op=%s)", errs.KindInvalid, e.Kind, e.Code, e.Op)
	}
	if e.Code != "CHECK_VIOLATION" {
		t.Fatalf("expected code=CHECK_VIOLATION, got %s", e.Code)
	}
}

func TestRepository_List_ActiveOnly(t *testing.T) {
	ctx, tx := withTx(t)
	repo := New(tx)

	a, err := repo.Create(ctx, incidents.CreateIncident{
		Title:       "a",
		Description: "",
		Center:      incidents.Point{Lat: 1, Lon: 1},
		Radius:      100,
	})
	if err != nil {
		t.Fatalf("Create a: %v", err)
	}

	b, err := repo.Create(ctx, incidents.CreateIncident{
		Title:       "b",
		Description: "",
		Center:      incidents.Point{Lat: 2, Lon: 2},
		Radius:      200,
	})
	if err != nil {
		t.Fatalf("Create b: %v", err)
	}

	if err := repo.Deactivate(ctx, a.ID); err != nil {
		t.Fatalf("Deactivate a: %v", err)
	}

	items, err := repo.List(ctx, incidents.ListFilter{Limit: 50, Offset: 0, ActiveOnly: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 active incident, got %d", len(items))
	}
	if items[0].ID != b.ID {
		t.Fatalf("expected active id=%d, got id=%d", b.ID, items[0].ID)
	}
}

func TestRepository_Update_Partial(t *testing.T) {
	ctx, tx := withTx(t)
	repo := New(tx)

	created, err := repo.Create(ctx, incidents.CreateIncident{
		Title:       "old",
		Description: "old",
		Center:      incidents.Point{Lat: 10, Lon: 20},
		Radius:      100,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	newTitle := "new"
	newRadius := 999
	updated, err := repo.Update(ctx, created.ID, incidents.UpdateIncident{
		Title:  &newTitle,
		Radius: &newRadius,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != newTitle || updated.Radius != newRadius {
		t.Fatalf("unexpected result: %+v", updated)
	}
	if updated.Center != created.Center {
		t.Fatalf("center should be unchanged: got=%+v want=%+v", updated.Center, created.Center)
	}
}

func TestRepository_Deactivate_NotFound_WhenAlreadyInactive(t *testing.T) {
	ctx, tx := withTx(t)
	repo := New(tx)

	created, err := repo.Create(ctx, incidents.CreateIncident{
		Title:       "x",
		Description: "",
		Center:      incidents.Point{Lat: 3, Lon: 3},
		Radius:      100,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Deactivate(ctx, created.ID); err != nil {
		t.Fatalf("Deactivate #1: %v", err)
	}

	err = repo.Deactivate(ctx, created.ID)
	if err == nil {
		t.Fatalf("expected error")
	}

	e, ok := errs.As(err)
	if !ok || e.Kind != errs.KindNotFound {
		t.Fatalf("expected not_found, got %T: %v", err, err)
	}
}
