package database

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"github.com/fathimasithara01/tradeverse/internal/config"
)

func ConnectDB(ctx context.Context, cfg *config.Config) (*sqlx.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.Name, cfg.Database.SSLMode)

	log.Info().Msgf("Connecting to database: host=%s port=%s user=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Name, cfg.Database.SSLMode)

	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	// Set connection pool limits
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	log.Info().Msg("Database connected and pinged successfully")
	return db, nil
}

func RunMigrations(db *sqlx.DB, migrationsDir string) error {
	log.Info().Msgf("Running database migrations from directory: %s", migrationsDir)

	// Create schema_migrations table if not exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			dirty BOOLEAN NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations tracker table: %w", err)
	}

	// Read migration files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	type migrationFile struct {
		version  int64
		name     string
		filepath string
	}

	var migrationUpFiles []migrationFile

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if strings.HasSuffix(name, ".up.sql") {
			parts := strings.SplitN(name, "_", 2)
			if len(parts) < 2 {
				continue
			}
			version, err := strconv.ParseInt(parts[0], 10, 64)
			if err != nil {
				log.Warn().Msgf("Skipping migration file %s with invalid version suffix", name)
				continue
			}
			migrationUpFiles = append(migrationUpFiles, migrationFile{
				version:  version,
				name:     name,
				filepath: filepath.Join(migrationsDir, name),
			})
		}
	}

	// Sort files by version ascending
	sort.Slice(migrationUpFiles, func(i, j int) bool {
		return migrationUpFiles[i].version < migrationUpFiles[j].version
	})

	for _, m := range migrationUpFiles {
		// Check if migration has already been applied and is not dirty
		var isApplied bool
		var isDirty bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1 AND dirty = false), COALESCE((SELECT dirty FROM schema_migrations WHERE version = $1), false)", m.version).Scan(&isApplied, &isDirty)
		if err != nil {
			return fmt.Errorf("failed to check migration status for version %d: %w", m.version, err)
		}

		if isDirty {
			return fmt.Errorf("database schema is in a dirty state for migration version %d. Please resolve manually", m.version)
		}

		if isApplied {
			log.Debug().Msgf("Migration %s already applied", m.name)
			continue
		}

		log.Info().Msgf("Applying migration %s...", m.name)

		// Read SQL content
		sqlContent, err := os.ReadFile(m.filepath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", m.name, err)
		}

		// Run migration inside transaction
		tx, err := db.Beginx()
		if err != nil {
			return fmt.Errorf("failed to start migration transaction: %w", err)
		}

		// Insert migration as dirty first
		_, err = tx.Exec("INSERT INTO schema_migrations (version, dirty) VALUES ($1, true) ON CONFLICT (version) DO UPDATE SET dirty = true", m.version)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to mark migration %d as dirty: %w", m.version, err)
		}

		// Execute schema SQL queries
		if _, err := tx.Exec(string(sqlContent)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration script %s: %w", m.name, err)
		}

		// Mark migration as clean (not dirty)
		_, err = tx.Exec("UPDATE schema_migrations SET dirty = false WHERE version = $1", m.version)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to mark migration %d as clean: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration transaction for %s: %w", m.name, err)
		}

		log.Info().Msgf("Successfully applied migration %s", m.name)
	}

	log.Info().Msg("All migrations applied successfully")
	return nil
}

// EnsureDirectoryExists helper tool for verifying paths during application initialization.
func EnsureDirectoryExists(dirPath string) error {
	return os.MkdirAll(dirPath, fs.ModePerm)
}
