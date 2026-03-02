package database

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/jhin1m/juiscript/internal/system"
)

// nameRegex validates database and username format.
// Only lowercase letters, digits, underscore. Must start with letter.
var nameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// systemDBs cannot be dropped or modified.
var systemDBs = map[string]bool{
	"information_schema": true,
	"mysql":              true,
	"performance_schema": true,
	"sys":                true,
}

// passwordChars for secure password generation.
const passwordChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"

// DBInfo holds database metadata.
type DBInfo struct {
	Name   string
	SizeMB float64
	Tables int
}

// Manager handles MariaDB database and user operations.
// Uses socket auth (no password) since juiscript runs as root.
type Manager struct {
	executor system.Executor
}

// NewManager creates a database manager with the given executor.
func NewManager(executor system.Executor) *Manager {
	return &Manager{executor: executor}
}

// validateName checks if a database or username is safe for SQL.
func validateName(name string) error {
	if !nameRegex.MatchString(name) {
		return fmt.Errorf("invalid name %q: must match ^[a-z][a-z0-9_]{0,63}$", name)
	}
	return nil
}

// isSystemDB returns true if the name is a protected system database.
func isSystemDB(name string) bool {
	return systemDBs[strings.ToLower(name)]
}

// exec runs a SQL statement via the mysql CLI with socket auth.
func (m *Manager) exec(ctx context.Context, sql string) (string, error) {
	return m.executor.Run(ctx, "mysql", "-u", "root", "-N", "-e", sql)
}

// GeneratePassword creates a cryptographically secure random password.
func GeneratePassword(length int) (string, error) {
	if length < 8 {
		length = 8
	}
	result := make([]byte, length)
	for i := range result {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordChars))))
		if err != nil {
			return "", fmt.Errorf("generate password: %w", err)
		}
		result[i] = passwordChars[idx.Int64()]
	}
	return string(result), nil
}
