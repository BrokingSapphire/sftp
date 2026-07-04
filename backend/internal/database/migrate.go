package database

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// migrationRecord tracks applied migrations.
type migrationRecord struct {
	Version string `gorm:"primaryKey;size:255"`
}

func (migrationRecord) TableName() string { return "schema_migrations" }

// Migrate applies all *.up.sql files from the given filesystem in
// lexical order, skipping those already recorded. Migrations are never
// auto-generated from models; the SQL files are the source of truth.
func Migrate(db *gorm.DB, fsys fs.FS, log *zap.Logger) error {
	if err := db.AutoMigrate(&migrationRecord{}); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	files, err := collectUpMigrations(fsys)
	if err != nil {
		return err
	}

	var applied []migrationRecord
	if err := db.Find(&applied).Error; err != nil {
		return fmt.Errorf("load applied migrations: %w", err)
	}
	done := make(map[string]bool, len(applied))
	for _, a := range applied {
		done[a.Version] = true
	}

	count := 0
	for _, f := range files {
		version := migrationVersion(f)
		if done[version] {
			continue
		}
		content, err := fs.ReadFile(fsys, f)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(content)).Error; err != nil {
				return err
			}
			return tx.Create(&migrationRecord{Version: version}).Error
		})
		if err != nil {
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
		log.Info("migration applied", zap.String("version", version))
		count++
	}

	log.Info("migrations complete", zap.Int("applied", count), zap.Int("total", len(files)))
	return nil
}

func collectUpMigrations(fsys fs.FS) ([]string, error) {
	var files []string
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".up.sql") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk migrations: %w", err)
	}
	sort.Strings(files)
	return files, nil
}

func migrationVersion(path string) string {
	base := path
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	return strings.TrimSuffix(base, ".up.sql")
}
