package database

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// dbOpTimeout for large import/export operations (10 minutes).
const dbOpTimeout = 10 * time.Minute

// safePathRegex rejects shell metacharacters in file paths.
// Allows: letters, digits, slash, dot, dash, underscore.
var safePathRegex = regexp.MustCompile(`^[a-zA-Z0-9/_.\-]+$`)

// validatePath checks a file path is safe for shell interpolation.
func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	if !safePathRegex.MatchString(path) {
		return fmt.Errorf("unsafe file path %q: contains shell metacharacters", path)
	}
	return nil
}

// Import loads a SQL file into a database. Supports .sql and .sql.gz files.
// Uses streaming via shell pipe to handle large files without memory issues.
func (m *Manager) Import(ctx context.Context, dbName, filePath string) error {
	if err := validateName(dbName); err != nil {
		return err
	}
	if err := validatePath(filePath); err != nil {
		return err
	}

	// Verify file exists
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("import file not found: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, dbOpTimeout)
	defer cancel()

	// Detect gzipped file and build appropriate command
	var cmd string
	if strings.HasSuffix(filePath, ".gz") {
		cmd = fmt.Sprintf("gunzip -c %s | mysql -u root %s", filePath, dbName)
	} else {
		cmd = fmt.Sprintf("mysql -u root %s < %s", dbName, filePath)
	}

	_, err := m.executor.Run(ctx, "bash", "-c", cmd)
	if err != nil {
		return fmt.Errorf("import into %s: %w", dbName, err)
	}
	return nil
}

// Export dumps a database to a SQL file using mysqldump.
// Uses --single-transaction for consistent export without locking.
func (m *Manager) Export(ctx context.Context, dbName, outputPath string) error {
	if err := validateName(dbName); err != nil {
		return err
	}
	if err := validatePath(outputPath); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, dbOpTimeout)
	defer cancel()

	var cmd string
	if strings.HasSuffix(outputPath, ".gz") {
		cmd = fmt.Sprintf("mysqldump -u root --single-transaction --quick %s | gzip > %s", dbName, outputPath)
	} else {
		cmd = fmt.Sprintf("mysqldump -u root --single-transaction --quick %s > %s", dbName, outputPath)
	}

	_, err := m.executor.Run(ctx, "bash", "-c", cmd)
	if err != nil {
		return fmt.Errorf("export %s: %w", dbName, err)
	}
	return nil
}
