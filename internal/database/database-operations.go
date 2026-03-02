package database

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// CreateDB creates a new MariaDB database with utf8mb4 charset.
func (m *Manager) CreateDB(ctx context.Context, name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	sql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", name)
	_, err := m.exec(ctx, sql)
	return err
}

// DropDB drops a database after checking it's not a system database.
func (m *Manager) DropDB(ctx context.Context, name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	if isSystemDB(name) {
		return fmt.Errorf("cannot drop system database: %s", name)
	}
	sql := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", name)
	_, err := m.exec(ctx, sql)
	return err
}

// ListDBs returns all non-system databases with size and table count.
func (m *Manager) ListDBs(ctx context.Context) ([]DBInfo, error) {
	// Query database sizes and table counts from information_schema
	sql := `SELECT s.schema_name, IFNULL(ROUND(SUM(t.data_length + t.index_length) / 1024 / 1024, 2), 0) AS size_mb, COUNT(t.table_name) AS tables FROM information_schema.schemata s LEFT JOIN information_schema.tables t ON s.schema_name = t.table_schema GROUP BY s.schema_name ORDER BY s.schema_name`

	output, err := m.exec(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("list databases: %w", err)
	}

	var dbs []DBInfo
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		name := fields[0]
		// Skip system databases
		if isSystemDB(name) {
			continue
		}

		sizeMB, _ := strconv.ParseFloat(fields[1], 64)
		tables, _ := strconv.Atoi(fields[2])

		dbs = append(dbs, DBInfo{
			Name:   name,
			SizeMB: sizeMB,
			Tables: tables,
		})
	}

	return dbs, nil
}
