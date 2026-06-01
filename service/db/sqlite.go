package db

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/v2rayA/v2rayA/conf"
	"github.com/v2rayA/v2rayA/pkg/util/log"
)

var (
	onceDB   sync.Once
	sqlDB    *sql.DB
	dbPath   string
	readOnly bool
	IsNewDB  bool // true if the database was just created (no pre-existing file)
	writeMu  sync.Mutex
)

// ErrNeedMigration is returned when an old BoltDB database is detected,
// indicating that migration to SQLite is required before normal operation.
var ErrNeedMigration = errors.New("bolt.db exists, migration required")

func validateSQLiteDriver() error {
	if sqliteDriverName == "" {
		return errors.New("mips/loong64 build requires CGO_ENABLED=1 to enable sqlite driver")
	}
	return nil
}

// SetReadOnly sets the database to read-only mode.
// In read-only mode, a temporary copy of the database is created.
func SetReadOnly() {
	readOnly = true
}

// initDB opens or creates the SQLite database with WAL mode
func initDB() {
	confPath := conf.GetEnvironmentConfig().Config
	dbPath = filepath.Join(confPath, "v2raya.db")

	if readOnly {
		// For read-only mode, create a temporary copy
		f, err := os.CreateTemp(os.TempDir(), "v2raya_tmp_sqlite_*.db")
		if err != nil {
			panic(err)
		}
		newPath := f.Name()
		f.Close()

		// Copy existing database if it exists
		if _, err := os.Stat(dbPath); err == nil {
			input, err := os.ReadFile(dbPath)
			if err != nil {
				panic(err)
			}
			if err := os.WriteFile(newPath, input, 0600); err != nil {
				panic(err)
			}
		}
		dbPath = newPath
	}

	var err error
	if err = validateSQLiteDriver(); err != nil {
		log.Fatal("SQLite driver is unavailable: %v", err)
	}
	sqlDB, err = sql.Open(sqliteDriverName, sqliteDSN(dbPath))
	if err != nil {
		log.Fatal("sql.Open: %v", err)
	}
	// Keep the pool small: SQLite has a single-writer model, while multiple
	// read connections are still useful and avoid self-deadlocks in tests that
	// compile packages with init-time DB access.
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(4)

	// Configure PRAGMAs for WAL mode and performance
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
		"PRAGMA cache_size=-8000",
	}
	for _, p := range pragmas {
		if _, err := sqlDB.Exec(p); err != nil {
			log.Fatal("Failed to set PRAGMA %s: %v", p, err)
		}
	}

	// Initialize schema
	if err := InitSchema(sqlDB); err != nil {
		log.Fatal("InitSchema: %v", err)
	}
}

func sqliteDSN(path string) string {
	return "file:" + url.PathEscape(path) + "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)"
}

// GetDB returns the singleton SQLite database connection
func GetDB() *sql.DB {
	onceDB.Do(initDB)
	return sqlDB
}

// DBPath returns the path to the SQLite database file
func DBPath() string {
	onceDB.Do(initDB)
	return dbPath
}

// Open initializes the database connection.
// It first checks if an old BoltDB database (bolt.db) exists.
// If bolt.db exists, it returns ErrNeedMigration to signal that
// migration is required before SQLite can be used.
// Otherwise, it creates/opens the SQLite database and initializes the schema.
func Open() error {
	confPath := conf.GetEnvironmentConfig().Config
	boltPath := filepath.Join(confPath, "bolt.db")
	sqlitePath := filepath.Join(confPath, "v2raya.db")

	// Check if old BoltDB database exists
	if _, err := os.Stat(boltPath); err == nil {
		// bolt.db exists — migration is required
		// Do NOT create or initialize SQLite at this point
		return ErrNeedMigration
	}

	// bolt.db does not exist, proceed with normal SQLite initialization
	// Check if SQLite already exists before creating it
	if _, err := os.Stat(sqlitePath); os.IsNotExist(err) {
		// Database file does not exist — this is a fresh install
		IsNewDB = true
		// Create an empty file so sql.Open does not fail
		f, err := os.Create(sqlitePath)
		if err != nil {
			return fmt.Errorf("failed to create SQLite database file: %w", err)
		}
		f.Close()
	} else {
		// Database file already exists — this is a restart
		IsNewDB = false
	}

	GetDB()
	return nil
}

// Close closes the SQLite database connection
func Close() error {
	if sqlDB != nil {
		return sqlDB.Close()
	}
	return nil
}

// IsBusyError reports whether err is a transient SQLite busy/locked error.
func IsBusyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database table is locked") ||
		strings.Contains(msg, "sqlite_busy") ||
		strings.Contains(msg, "sqlite_locked")
}

func retryDelay(attempt int) time.Duration {
	delay := time.Duration(attempt+1) * 100 * time.Millisecond
	if delay > time.Second {
		return time.Second
	}
	return delay
}

func WithBusyRetry(fn func() error) error {
	deadline := time.Now().Add(15 * time.Second)
	var err error
	for attempt := 0; ; attempt++ {
		err = fn()
		if !IsBusyError(err) || time.Now().After(deadline) {
			return err
		}
		time.Sleep(retryDelay(attempt))
	}
}

// ReadModifyWrite executes a function within a read-write transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
func ReadModifyWrite(fn func(tx *sql.Tx) error) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	return WithBusyRetry(func() error {
		db := GetDB()
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		defer func() {
			if p := recover(); p != nil {
				_ = tx.Rollback()
				panic(p)
			}
		}()

		if err := fn(tx); err != nil {
			_ = tx.Rollback()
			return err
		}

		return tx.Commit()
	})
}
