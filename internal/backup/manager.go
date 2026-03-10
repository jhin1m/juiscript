package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/jhin1m/juiscript/internal/config"
	"github.com/jhin1m/juiscript/internal/database"
	"github.com/jhin1m/juiscript/internal/site"
	"github.com/jhin1m/juiscript/internal/system"
)

// Timeout for backup/restore operations (large sites may take a while).
const opTimeout = 15 * time.Minute

// BackupType identifies what a backup contains.
type BackupType string

const (
	BackupFull     BackupType = "full"
	BackupFiles    BackupType = "files"
	BackupDatabase BackupType = "database"
)

// BackupInfo holds metadata about a single backup archive.
type BackupInfo struct {
	Path      string
	Domain    string
	Type      BackupType
	Size      int64
	CreatedAt time.Time
}

// Metadata is stored inside the backup archive as metadata.toml.
// Contains enough info to restore on a different server.
type Metadata struct {
	Domain      string    `toml:"domain"`
	Type        string    `toml:"type"`
	ProjectType string    `toml:"project_type"`
	PHPVersion  string    `toml:"php_version"`
	DBName      string    `toml:"db_name"`
	DBUser      string    `toml:"db_user"`
	SiteUser    string    `toml:"site_user"`
	CreatedAt   time.Time `toml:"created_at"`
}

// Options configures a backup creation request.
type Options struct {
	Domain string
	Type   BackupType
}

// Manager handles backup and restore operations.
type Manager struct {
	config   *config.Config
	executor system.Executor
	files    system.FileManager
	db       *database.Manager
}

// NewManager creates a backup manager with required dependencies.
func NewManager(cfg *config.Config, executor system.Executor, files system.FileManager, db *database.Manager) *Manager {
	return &Manager{
		config:   cfg,
		executor: executor,
		files:    files,
		db:       db,
	}
}

// backupDir returns the configured backup directory as an absolute path.
func (m *Manager) backupDir() string {
	dir, err := filepath.Abs(m.config.Backup.Dir)
	if err != nil {
		return m.config.Backup.Dir
	}
	return dir
}

// isInsideBackupDir checks that a path is contained within the backup directory.
// Prevents path traversal attacks where crafted paths could delete arbitrary files.
func (m *Manager) isInsideBackupDir(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	dir := m.backupDir()
	if !strings.HasPrefix(abs, dir+"/") {
		return fmt.Errorf("path %q is outside backup directory", path)
	}
	return nil
}

// cronScheduleRegex validates each field of a 5-field cron schedule.
// Allows: *, digits, commas, dashes, slashes (e.g., "0 2 * * *").
var cronScheduleRegex = regexp.MustCompile(`^(\*|[0-9,\-*/]+)(\s+(\*|[0-9,\-*/]+)){4}$`)

// safeNameRegex prevents path traversal in domain/backup names.
var safeNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._\-]+$`)

// validateDomain checks that a domain name is safe for file paths.
func validateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}
	if !safeNameRegex.MatchString(domain) {
		return fmt.Errorf("invalid domain %q: contains unsafe characters", domain)
	}
	return nil
}

// backupFilename generates a filename like domain_20260302_150405.tar.gz
func backupFilename(domain string, t time.Time) string {
	return fmt.Sprintf("%s_%s.tar.gz", domain, t.Format("20060102_150405"))
}

// parseBackupFilename extracts domain and timestamp from a backup filename.
func parseBackupFilename(name string) (domain string, createdAt time.Time, ok bool) {
	// Expected: domain_20060102_150405.tar.gz
	if !strings.HasSuffix(name, ".tar.gz") {
		return "", time.Time{}, false
	}
	name = strings.TrimSuffix(name, ".tar.gz")
	if name == "" {
		return "", time.Time{}, false
	}

	// Find the last two underscore-separated segments (date_time)
	parts := strings.Split(name, "_")
	if len(parts) < 3 {
		return "", time.Time{}, false
	}

	timePart := parts[len(parts)-2] + "_" + parts[len(parts)-1]
	domainPart := strings.Join(parts[:len(parts)-2], "_")

	t, err := time.Parse("20060102_150405", timePart)
	if err != nil {
		return "", time.Time{}, false
	}

	return domainPart, t, true
}

// Create creates a backup archive for the specified site.
func (m *Manager) Create(ctx context.Context, opts Options) (*BackupInfo, error) {
	if err := validateDomain(opts.Domain); err != nil {
		return nil, err
	}

	// Load site metadata to get DB name, user, etc.
	s, err := site.LoadMetadata(m.config.SitesPath(), opts.Domain)
	if err != nil {
		return nil, fmt.Errorf("load site metadata: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	// Ensure backup directory exists
	backupDir := m.backupDir()
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return nil, fmt.Errorf("create backup dir: %w", err)
	}

	now := time.Now()
	tmpDir, err := os.MkdirTemp("", "juiscript-backup-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Step 1: Backup database if needed (full or database type)
	if opts.Type == BackupFull || opts.Type == BackupDatabase {
		if s.DBName != "" {
			dbDumpPath := filepath.Join(tmpDir, "database.sql.gz")
			if err := m.db.Export(ctx, s.DBName, dbDumpPath); err != nil {
				return nil, fmt.Errorf("export database: %w", err)
			}
		}
	}

	// Step 2: Backup site files if needed (full or files type)
	if opts.Type == BackupFull || opts.Type == BackupFiles {
		siteDir := s.SiteDir(m.config.General.SitesRoot)
		filesArchive := filepath.Join(tmpDir, "files.tar.gz")

		// Use tar with streaming gzip to handle large sites
		_, err := m.executor.Run(ctx, "tar", "-czf", filesArchive, "-C", siteDir, ".")
		if err != nil {
			return nil, fmt.Errorf("archive site files: %w", err)
		}
	}

	// Step 3: Write metadata.toml into temp dir
	meta := Metadata{
		Domain:      s.Domain,
		Type:        string(opts.Type),
		ProjectType: string(s.ProjectType),
		PHPVersion:  s.PHPVersion,
		DBName:      s.DBName,
		DBUser:      s.DBUser,
		SiteUser:    s.User,
		CreatedAt:   now,
	}
	metaPath := filepath.Join(tmpDir, "metadata.toml")
	if err := writeMetadata(metaPath, &meta); err != nil {
		return nil, fmt.Errorf("write metadata: %w", err)
	}

	// Step 4: Package everything into final archive
	archiveName := backupFilename(opts.Domain, now)
	archivePath := filepath.Join(backupDir, archiveName)

	_, err = m.executor.Run(ctx, "tar", "-czf", archivePath, "-C", tmpDir, ".")
	if err != nil {
		return nil, fmt.Errorf("create backup archive: %w", err)
	}

	// Restrict permissions - backup contains sensitive data (DB dumps)
	if err := os.Chmod(archivePath, 0600); err != nil {
		return nil, fmt.Errorf("set backup permissions: %w", err)
	}

	// Get file size for info
	info, err := os.Stat(archivePath)
	if err != nil {
		return nil, fmt.Errorf("stat backup: %w", err)
	}

	return &BackupInfo{
		Path:      archivePath,
		Domain:    opts.Domain,
		Type:      opts.Type,
		Size:      info.Size(),
		CreatedAt: now,
	}, nil
}

// Restore restores a site from a backup archive. Path must be inside backup directory.
func (m *Manager) Restore(ctx context.Context, backupPath, domain string) error {
	if err := validateDomain(domain); err != nil {
		return err
	}

	if err := m.isInsideBackupDir(backupPath); err != nil {
		return err
	}

	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	// Extract archive to temp dir
	tmpDir, err := os.MkdirTemp("", "juiscript-restore-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = m.executor.Run(ctx, "tar", "-xzf", backupPath, "-C", tmpDir)
	if err != nil {
		return fmt.Errorf("extract backup: %w", err)
	}

	// Read metadata
	meta, err := readMetadata(filepath.Join(tmpDir, "metadata.toml"))
	if err != nil {
		return fmt.Errorf("read backup metadata: %w", err)
	}

	// Load site metadata for paths
	s, err := site.LoadMetadata(m.config.SitesPath(), domain)
	if err != nil {
		return fmt.Errorf("load site metadata: %w", err)
	}

	backupType := BackupType(meta.Type)

	// Validate backup type from archive metadata
	switch backupType {
	case BackupFull, BackupFiles, BackupDatabase:
		// valid
	default:
		return fmt.Errorf("unknown backup type %q in archive metadata", meta.Type)
	}

	// Restore files if backup contains them
	filesArchive := filepath.Join(tmpDir, "files.tar.gz")
	if backupType == BackupFull || backupType == BackupFiles {
		if _, err := os.Stat(filesArchive); err == nil {
			siteDir := s.SiteDir(m.config.General.SitesRoot)

			// Extract files to site directory
			_, err = m.executor.Run(ctx, "tar", "-xzf", filesArchive, "-C", siteDir)
			if err != nil {
				return fmt.Errorf("restore site files: %w", err)
			}

			// Fix ownership recursively
			_, err = m.executor.Run(ctx, "chown", "-R", s.User+":"+s.User, siteDir)
			if err != nil {
				return fmt.Errorf("fix file ownership: %w", err)
			}
		}
	}

	// Restore database if backup contains it
	dbDump := filepath.Join(tmpDir, "database.sql.gz")
	if backupType == BackupFull || backupType == BackupDatabase {
		if _, err := os.Stat(dbDump); err == nil {
			if err := m.db.Import(ctx, s.DBName, dbDump); err != nil {
				return fmt.Errorf("restore database: %w", err)
			}
		}
	}

	return nil
}

// List returns all backups for a domain, sorted newest first.
func (m *Manager) List(domain string) ([]BackupInfo, error) {
	if err := validateDomain(domain); err != nil {
		return nil, err
	}

	backupDir := m.backupDir()
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read backup dir: %w", err)
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tar.gz") {
			continue
		}

		parsedDomain, createdAt, ok := parseBackupFilename(entry.Name())
		if !ok || parsedDomain != domain {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backups = append(backups, BackupInfo{
			Path:      filepath.Join(backupDir, entry.Name()),
			Domain:    domain,
			Size:      info.Size(),
			CreatedAt: createdAt,
		})
	}

	// Sort newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// Delete removes a single backup file. Path must be inside backup directory.
func (m *Manager) Delete(backupPath string) error {
	if err := m.isInsideBackupDir(backupPath); err != nil {
		return err
	}
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}
	return os.Remove(backupPath)
}

// Cleanup keeps only the N most recent backups for a domain, deleting older ones.
func (m *Manager) Cleanup(domain string, keepLast int) error {
	if keepLast < 1 {
		return fmt.Errorf("keepLast must be >= 1")
	}

	backups, err := m.List(domain)
	if err != nil {
		return err
	}

	// Already sorted newest first by List()
	if len(backups) <= keepLast {
		return nil
	}

	// Delete everything beyond keepLast
	for _, b := range backups[keepLast:] {
		if err := os.Remove(b.Path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("delete old backup %s: %w", b.Path, err)
		}
	}

	return nil
}

// SetupCron creates a cron job for scheduled backups.
// Uses /etc/cron.d/ which doesn't require crontab editing.
func (m *Manager) SetupCron(domain, schedule string) error {
	if err := validateDomain(domain); err != nil {
		return err
	}
	if schedule == "" {
		return fmt.Errorf("cron schedule cannot be empty")
	}

	// Validate cron schedule format to prevent injection via newlines or special chars
	if !cronScheduleRegex.MatchString(schedule) {
		return fmt.Errorf("invalid cron schedule %q: must be 5 fields (minute hour day month weekday)", schedule)
	}

	// Build the cron entry: schedule + command
	// juiscript binary creates a full backup and applies retention
	cronLine := fmt.Sprintf("%s root /usr/local/bin/juiscript backup create --domain %s --type full\n",
		schedule, domain)

	cronFile := fmt.Sprintf("/etc/cron.d/juiscript-%s", domain)

	return m.files.WriteAtomic(cronFile, []byte(cronLine), 0644)
}

// RemoveCron removes the scheduled backup cron job for a domain.
func (m *Manager) RemoveCron(domain string) error {
	if err := validateDomain(domain); err != nil {
		return err
	}

	cronFile := fmt.Sprintf("/etc/cron.d/juiscript-%s", domain)
	if !m.files.Exists(cronFile) {
		return nil // already removed
	}
	return m.files.Remove(cronFile)
}

// writeMetadata encodes metadata to a TOML file.
func writeMetadata(path string, meta *Metadata) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(meta)
}

// readMetadata decodes metadata from a TOML file.
func readMetadata(path string) (*Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta Metadata
	if err := toml.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// FormatSize returns a human-readable file size string.
func FormatSize(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
