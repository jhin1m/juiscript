package database

import (
	"context"
	"fmt"
)

// CreateUser creates a MariaDB user with full privileges on a specific database.
// Returns the generated password.
func (m *Manager) CreateUser(ctx context.Context, username, dbName string) (string, error) {
	if err := validateName(username); err != nil {
		return "", fmt.Errorf("invalid username: %w", err)
	}
	if err := validateName(dbName); err != nil {
		return "", fmt.Errorf("invalid database name: %w", err)
	}

	password, err := GeneratePassword(24)
	if err != nil {
		return "", err
	}

	// Create user, grant privileges, flush in one batch
	sql := fmt.Sprintf(
		"CREATE USER IF NOT EXISTS '%s'@'localhost' IDENTIFIED BY '%s'; "+
			"GRANT ALL ON `%s`.* TO '%s'@'localhost'; "+
			"FLUSH PRIVILEGES",
		username, password, dbName, username,
	)

	if _, err := m.exec(ctx, sql); err != nil {
		return "", fmt.Errorf("create user %s: %w", username, err)
	}

	return password, nil
}

// DropUser revokes all privileges and drops a database user.
func (m *Manager) DropUser(ctx context.Context, username string) error {
	if err := validateName(username); err != nil {
		return fmt.Errorf("invalid username: %w", err)
	}

	sql := fmt.Sprintf(
		"REVOKE ALL PRIVILEGES, GRANT OPTION FROM '%s'@'localhost'; "+
			"DROP USER IF EXISTS '%s'@'localhost'; "+
			"FLUSH PRIVILEGES",
		username, username,
	)

	_, err := m.exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("drop user %s: %w", username, err)
	}
	return nil
}

// ResetPassword sets a new password for an existing database user.
// Returns the new password.
func (m *Manager) ResetPassword(ctx context.Context, username string) (string, error) {
	if err := validateName(username); err != nil {
		return "", fmt.Errorf("invalid username: %w", err)
	}

	password, err := GeneratePassword(24)
	if err != nil {
		return "", err
	}

	sql := fmt.Sprintf(
		"ALTER USER '%s'@'localhost' IDENTIFIED BY '%s'; FLUSH PRIVILEGES",
		username, password,
	)

	if _, err := m.exec(ctx, sql); err != nil {
		return "", fmt.Errorf("reset password for %s: %w", username, err)
	}

	return password, nil
}
