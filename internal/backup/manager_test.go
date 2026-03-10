package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jhin1m/juiscript/internal/config"
)

// --- Mock Infrastructure ---

type mockExecutor struct {
	commands []string
	failOn   map[string]error
	// onRun is called after recording the command; use to create side-effect files
	onRun func(name string, args []string) error
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{failOn: make(map[string]error)}
}

func (m *mockExecutor) Run(_ context.Context, name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	m.commands = append(m.commands, cmd)
	if err, ok := m.failOn[name]; ok {
		return "", err
	}
	if m.onRun != nil {
		if err := m.onRun(name, args); err != nil {
			return "", err
		}
	}
	return "", nil
}

func (m *mockExecutor) RunWithInput(_ context.Context, _ string, name string, args ...string) (string, error) {
	return m.Run(context.Background(), name, args...)
}

type mockFileManager struct {
	written  map[string][]byte
	symlinks map[string]string
	exists   map[string]bool
	failOn   map[string]error
}

func newMockFileManager() *mockFileManager {
	return &mockFileManager{
		written:  make(map[string][]byte),
		symlinks: make(map[string]string),
		exists:   make(map[string]bool),
		failOn:   make(map[string]error),
	}
}

func (f *mockFileManager) WriteAtomic(path string, data []byte, _ os.FileMode) error {
	if err, ok := f.failOn["write"]; ok {
		return err
	}
	f.written[path] = data
	f.exists[path] = true
	return nil
}

func (f *mockFileManager) Symlink(target, link string) error {
	if err, ok := f.failOn["symlink"]; ok {
		return err
	}
	f.symlinks[link] = target
	f.exists[link] = true
	return nil
}

func (f *mockFileManager) RemoveSymlink(path string) error {
	delete(f.symlinks, path)
	delete(f.exists, path)
	return nil
}

func (f *mockFileManager) Remove(path string) error {
	delete(f.written, path)
	delete(f.exists, path)
	return nil
}

func (f *mockFileManager) Exists(path string) bool {
	return f.exists[path]
}

func (f *mockFileManager) ReadFile(path string) ([]byte, error) {
	if data, ok := f.written[path]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

// testConfig creates a config with backup dir pointing to a temp directory.
func testConfig(backupDir string) *config.Config {
	return &config.Config{
		Backup: config.BackupConfig{Dir: backupDir},
	}
}

// createTestBackupFiles creates N fake backup archives in dir for the given domain.
func createTestBackupFiles(t *testing.T, dir, domain string, count int) []string {
	t.Helper()
	var paths []string
	for i := 0; i < count; i++ {
		ts := time.Now().Add(time.Duration(-i) * time.Hour)
		name := backupFilename(domain, ts)
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("fake-archive"), 0600); err != nil {
			t.Fatalf("create test backup file: %v", err)
		}
		paths = append(paths, path)
	}
	return paths
}

func TestBackupFilename(t *testing.T) {
	ts := time.Date(2026, 3, 2, 15, 4, 5, 0, time.UTC)
	got := backupFilename("example.com", ts)
	want := "example.com_20260302_150405.tar.gz"
	if got != want {
		t.Errorf("backupFilename() = %q, want %q", got, want)
	}
}

func TestParseBackupFilename(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		wantDomain string
		wantTime   time.Time
		wantOk     bool
	}{
		{
			name:       "valid backup filename",
			filename:   "example.com_20260302_150405.tar.gz",
			wantDomain: "example.com",
			wantTime:   time.Date(2026, 3, 2, 15, 4, 5, 0, time.UTC),
			wantOk:     true,
		},
		{
			name:       "domain with underscores",
			filename:   "my_app_site.com_20260101_120000.tar.gz",
			wantDomain: "my_app_site.com",
			wantTime:   time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			wantOk:     true,
		},
		{
			name:     "missing extension",
			filename: "example.com_20260302_150405",
			wantOk:   false,
		},
		{
			name:     "too few parts",
			filename: "example.tar.gz",
			wantOk:   false,
		},
		{
			name:     "invalid timestamp",
			filename: "example.com_notadate_nottime.tar.gz",
			wantOk:   false,
		},
		{
			name:     "empty string",
			filename: "",
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domain, createdAt, ok := parseBackupFilename(tt.filename)
			if ok != tt.wantOk {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOk)
			}
			if !ok {
				return
			}
			if domain != tt.wantDomain {
				t.Errorf("domain = %q, want %q", domain, tt.wantDomain)
			}
			if !createdAt.Equal(tt.wantTime) {
				t.Errorf("createdAt = %v, want %v", createdAt, tt.wantTime)
			}
		})
	}
}

func TestValidateDomain(t *testing.T) {
	tests := []struct {
		domain  string
		wantErr bool
	}{
		{"example.com", false},
		{"my-site.example.com", false},
		{"site_test.com", false},
		{"", true},
		{"../etc/passwd", true},
		{"site;rm -rf /", true},
		{"site com", true},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			err := validateDomain(tt.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDomain(%q) err = %v, wantErr %v", tt.domain, err, tt.wantErr)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatSize(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestCleanup_KeepLastValidation(t *testing.T) {
	m := &Manager{}
	err := m.Cleanup("example.com", 0)
	if err == nil {
		t.Error("Cleanup(keepLast=0) should return error")
	}
}

func TestCronScheduleValidation(t *testing.T) {
	tests := []struct {
		schedule string
		valid    bool
	}{
		{"0 2 * * *", true},          // daily at 2am
		{"*/15 * * * *", true},       // every 15 min
		{"0 0 * * 0", true},          // weekly Sunday
		{"30 4 1,15 * *", true},      // 1st and 15th
		{"", false},                   // empty
		{"* * *", false},              // too few fields
		{"0 2 * * * extra", false},    // too many fields
		{"0 2 * * *\n* * * * * root rm -rf /", false}, // injection attempt
	}

	for _, tt := range tests {
		valid := cronScheduleRegex.MatchString(tt.schedule)
		if valid != tt.valid {
			t.Errorf("cronScheduleRegex.Match(%q) = %v, want %v", tt.schedule, valid, tt.valid)
		}
	}
}

// --- List Tests ---

func TestList_WithBackups(t *testing.T) {
	dir := t.TempDir()
	createTestBackupFiles(t, dir, "example.com", 3)

	m := &Manager{
		config:   testConfig(dir),
		executor: newMockExecutor(),
		files:    newMockFileManager(),
	}

	backups, err := m.List("example.com")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(backups) != 3 {
		t.Fatalf("expected 3 backups, got %d", len(backups))
	}

	// Verify sorted newest-first
	for i := 1; i < len(backups); i++ {
		if backups[i].CreatedAt.After(backups[i-1].CreatedAt) {
			t.Error("backups should be sorted newest-first")
		}
	}

	// Verify all have correct domain and non-zero size
	for _, b := range backups {
		if b.Domain != "example.com" {
			t.Errorf("Domain = %q, want example.com", b.Domain)
		}
		if b.Size == 0 {
			t.Error("expected non-zero size")
		}
	}
}

func TestList_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	m := &Manager{config: testConfig(dir)}

	backups, err := m.List("example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if backups != nil {
		t.Errorf("expected nil, got %d backups", len(backups))
	}
}

func TestList_NonExistentDir(t *testing.T) {
	m := &Manager{config: testConfig("/tmp/does-not-exist-juiscript-test")}

	backups, err := m.List("example.com")
	if err != nil {
		t.Fatalf("unexpected error for non-existent dir: %v", err)
	}
	if backups != nil {
		t.Errorf("expected nil for non-existent dir")
	}
}

func TestList_FiltersByDomain(t *testing.T) {
	dir := t.TempDir()
	createTestBackupFiles(t, dir, "site-a.com", 2)
	createTestBackupFiles(t, dir, "site-b.com", 3)

	m := &Manager{config: testConfig(dir)}

	backupsA, err := m.List("site-a.com")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(backupsA) != 2 {
		t.Errorf("expected 2 for site-a.com, got %d", len(backupsA))
	}

	backupsB, err := m.List("site-b.com")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(backupsB) != 3 {
		t.Errorf("expected 3 for site-b.com, got %d", len(backupsB))
	}
}

// --- Delete Tests ---

func TestDelete_Success(t *testing.T) {
	dir := t.TempDir()
	paths := createTestBackupFiles(t, dir, "example.com", 1)

	m := &Manager{config: testConfig(dir)}

	if err := m.Delete(paths[0]); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := os.Stat(paths[0]); !os.IsNotExist(err) {
		t.Error("file should be removed after Delete")
	}
}

func TestDelete_OutsideBackupDir(t *testing.T) {
	dir := t.TempDir()
	m := &Manager{config: testConfig(dir)}

	err := m.Delete("/etc/passwd")
	if err == nil {
		t.Fatal("expected error for path outside backup dir")
	}
}

func TestDelete_NonExistent(t *testing.T) {
	dir := t.TempDir()
	m := &Manager{config: testConfig(dir)}

	err := m.Delete(filepath.Join(dir, "nonexistent.tar.gz"))
	if err == nil {
		t.Fatal("expected error for non-existent backup")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

// --- Cleanup Tests ---

func TestCleanup_KeepsCorrectNumber(t *testing.T) {
	dir := t.TempDir()
	createTestBackupFiles(t, dir, "example.com", 5)

	m := &Manager{config: testConfig(dir)}

	if err := m.Cleanup("example.com", 2); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	remaining, err := m.List("example.com")
	if err != nil {
		t.Fatalf("List after cleanup: %v", err)
	}
	if len(remaining) != 2 {
		t.Errorf("expected 2 remaining, got %d", len(remaining))
	}
}

func TestCleanup_LessThanKeepLast(t *testing.T) {
	dir := t.TempDir()
	createTestBackupFiles(t, dir, "example.com", 2)

	m := &Manager{config: testConfig(dir)}

	if err := m.Cleanup("example.com", 5); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	remaining, err := m.List("example.com")
	if err != nil {
		t.Fatalf("List after cleanup: %v", err)
	}
	if len(remaining) != 2 {
		t.Errorf("expected all 2 to remain, got %d", len(remaining))
	}
}

// --- SetupCron Tests ---

func TestSetupCron_Success(t *testing.T) {
	files := newMockFileManager()
	m := &Manager{
		config: testConfig(t.TempDir()),
		files:  files,
	}

	if err := m.SetupCron("example.com", "0 2 * * *"); err != nil {
		t.Fatalf("SetupCron failed: %v", err)
	}

	cronFile := "/etc/cron.d/juiscript-example.com"
	content, ok := files.written[cronFile]
	if !ok {
		t.Fatal("expected cron file to be written")
	}

	cronStr := string(content)
	if !strings.Contains(cronStr, "0 2 * * *") {
		t.Error("cron file should contain schedule")
	}
	if !strings.Contains(cronStr, "example.com") {
		t.Error("cron file should contain domain")
	}
}

func TestSetupCron_InvalidSchedule(t *testing.T) {
	m := &Manager{
		config: testConfig(t.TempDir()),
		files:  newMockFileManager(),
	}

	err := m.SetupCron("example.com", "not a schedule")
	if err == nil {
		t.Fatal("expected error for invalid schedule")
	}
}

func TestSetupCron_InvalidDomain(t *testing.T) {
	m := &Manager{
		config: testConfig(t.TempDir()),
		files:  newMockFileManager(),
	}

	err := m.SetupCron("../evil", "0 2 * * *")
	if err == nil {
		t.Fatal("expected error for invalid domain")
	}
}

func TestSetupCron_EmptySchedule(t *testing.T) {
	m := &Manager{
		config: testConfig(t.TempDir()),
		files:  newMockFileManager(),
	}

	err := m.SetupCron("example.com", "")
	if err == nil {
		t.Fatal("expected error for empty schedule")
	}
}

// --- RemoveCron Tests ---

func TestRemoveCron_Success(t *testing.T) {
	files := newMockFileManager()
	cronFile := "/etc/cron.d/juiscript-example.com"
	files.exists[cronFile] = true

	m := &Manager{
		config: testConfig(t.TempDir()),
		files:  files,
	}

	if err := m.RemoveCron("example.com"); err != nil {
		t.Fatalf("RemoveCron failed: %v", err)
	}

	if files.exists[cronFile] {
		t.Error("cron file should be removed")
	}
}

func TestRemoveCron_NotExists(t *testing.T) {
	m := &Manager{
		config: testConfig(t.TempDir()),
		files:  newMockFileManager(),
	}

	// Should not error (idempotent)
	if err := m.RemoveCron("example.com"); err != nil {
		t.Fatalf("RemoveCron should not error for non-existent: %v", err)
	}
}

// --- Metadata Roundtrip ---

func TestMetadata_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metadata.toml")

	now := time.Now().Truncate(time.Second)
	meta := &Metadata{
		Domain:      "example.com",
		Type:        "full",
		ProjectType: "laravel",
		PHPVersion:  "8.3",
		DBName:      "db_example",
		DBUser:      "usr_example",
		SiteUser:    "site_example_com",
		CreatedAt:   now,
	}

	if err := writeMetadata(path, meta); err != nil {
		t.Fatalf("writeMetadata: %v", err)
	}

	loaded, err := readMetadata(path)
	if err != nil {
		t.Fatalf("readMetadata: %v", err)
	}

	if loaded.Domain != meta.Domain {
		t.Errorf("Domain = %q, want %q", loaded.Domain, meta.Domain)
	}
	if loaded.Type != meta.Type {
		t.Errorf("Type = %q", loaded.Type)
	}
	if loaded.ProjectType != meta.ProjectType {
		t.Errorf("ProjectType = %q", loaded.ProjectType)
	}
	if loaded.PHPVersion != meta.PHPVersion {
		t.Errorf("PHPVersion = %q", loaded.PHPVersion)
	}
	if loaded.DBName != meta.DBName {
		t.Errorf("DBName = %q", loaded.DBName)
	}
	if loaded.DBUser != meta.DBUser {
		t.Errorf("DBUser = %q", loaded.DBUser)
	}
	if loaded.SiteUser != meta.SiteUser {
		t.Errorf("SiteUser = %q", loaded.SiteUser)
	}
	if !loaded.CreatedAt.Equal(meta.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", loaded.CreatedAt, meta.CreatedAt)
	}
}

// --- Create Tests (files-only, no DB dependency) ---

func TestCreate_FilesOnly_Success(t *testing.T) {
	backupDir := t.TempDir()
	sitesDir := t.TempDir()
	sitesRoot := t.TempDir()

	// Pre-create site metadata so LoadMetadata works
	siteUser := "site_example_com"
	siteHomeDir := filepath.Join(sitesRoot, siteUser)
	siteDir := filepath.Join(siteHomeDir, "example.com")
	os.MkdirAll(siteDir, 0750)

	// Write a site metadata TOML file
	metaContent := fmt.Sprintf(`domain = "example.com"
user = "%s"
project_type = "laravel"
php_version = "8.3"
web_root = "%s/public"
db_name = ""
db_user = ""
ssl_enabled = false
enabled = true
created_at = 2026-03-01T00:00:00Z
`, siteUser, siteDir)
	os.WriteFile(filepath.Join(sitesDir, "example.com.toml"), []byte(metaContent), 0640)

	cfg := &config.Config{
		General: config.GeneralConfig{
			SitesRoot: sitesRoot,
			SitesDir:  sitesDir,
		},
		Backup: config.BackupConfig{Dir: backupDir},
	}

	exec := newMockExecutor()
	// tar -czf creates an archive file; simulate it by creating an empty file
	exec.onRun = func(name string, args []string) error {
		if name == "tar" && len(args) > 1 && args[0] == "-czf" {
			return os.WriteFile(args[1], []byte("fake-tar"), 0600)
		}
		return nil
	}

	m := NewManager(cfg, exec, newMockFileManager(), nil)

	info, err := m.Create(context.Background(), Options{
		Domain: "example.com",
		Type:   BackupFiles,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if info.Domain != "example.com" {
		t.Errorf("Domain = %q", info.Domain)
	}
	if info.Type != BackupFiles {
		t.Errorf("Type = %q", info.Type)
	}
	if info.Path == "" {
		t.Error("Path should not be empty")
	}

	// Verify the archive file exists
	if _, err := os.Stat(info.Path); os.IsNotExist(err) {
		t.Error("backup archive should exist")
	}
}

func TestCreate_InvalidDomain(t *testing.T) {
	m := &Manager{config: testConfig(t.TempDir())}

	_, err := m.Create(context.Background(), Options{
		Domain: "../evil",
		Type:   BackupFiles,
	})
	if err == nil {
		t.Fatal("expected error for invalid domain")
	}
}

// --- Restore Tests ---

func TestRestore_InvalidDomain(t *testing.T) {
	m := &Manager{config: testConfig(t.TempDir())}

	err := m.Restore(context.Background(), "/some/path", "../evil")
	if err == nil {
		t.Fatal("expected error for invalid domain")
	}
}

func TestRestore_OutsideBackupDir(t *testing.T) {
	m := &Manager{config: testConfig(t.TempDir())}

	err := m.Restore(context.Background(), "/etc/passwd", "example.com")
	if err == nil {
		t.Fatal("expected error for path outside backup dir")
	}
}

func TestRestore_FilesOnly_Success(t *testing.T) {
	backupDir := t.TempDir()
	sitesDir := t.TempDir()
	sitesRoot := t.TempDir()

	// Pre-create site metadata
	siteUser := "site_restore_com"
	siteDir := filepath.Join(sitesRoot, siteUser, "restore.com")
	os.MkdirAll(siteDir, 0750)
	metaContent := fmt.Sprintf(`domain = "restore.com"
user = "%s"
project_type = "laravel"
php_version = "8.3"
web_root = "%s/public"
db_name = ""
db_user = ""
ssl_enabled = false
enabled = true
created_at = 2026-03-01T00:00:00Z
`, siteUser, siteDir)
	os.WriteFile(filepath.Join(sitesDir, "restore.com.toml"), []byte(metaContent), 0640)

	cfg := &config.Config{
		General: config.GeneralConfig{
			SitesRoot: sitesRoot,
			SitesDir:  sitesDir,
		},
		Backup: config.BackupConfig{Dir: backupDir},
	}

	exec := newMockExecutor()
	// tar -xzf extracts; simulate by creating metadata.toml in the temp dir
	exec.onRun = func(name string, args []string) error {
		if name == "tar" && len(args) > 0 && args[0] == "-xzf" {
			// Find the -C target dir and create metadata.toml there
			for i, a := range args {
				if a == "-C" && i+1 < len(args) {
					metaPath := filepath.Join(args[i+1], "metadata.toml")
					meta := `domain = "restore.com"
type = "files"
project_type = "laravel"
php_version = "8.3"
db_name = ""
db_user = ""
site_user = "` + siteUser + `"
created_at = 2026-03-01T00:00:00Z
`
					return os.WriteFile(metaPath, []byte(meta), 0640)
				}
			}
		}
		return nil
	}

	// Create a fake backup file inside the backup dir
	archivePath := filepath.Join(backupDir, "restore.com_20260301_000000.tar.gz")
	os.WriteFile(archivePath, []byte("fake-archive"), 0600)

	m := NewManager(cfg, exec, newMockFileManager(), nil)

	err := m.Restore(context.Background(), archivePath, "restore.com")
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
}

func TestRestore_NonExistentBackup(t *testing.T) {
	dir := t.TempDir()
	m := &Manager{config: testConfig(dir)}

	err := m.Restore(context.Background(), filepath.Join(dir, "nope.tar.gz"), "example.com")
	if err == nil {
		t.Fatal("expected error for non-existent backup")
	}
}

func TestIsInsideBackupDir(t *testing.T) {
	m := &Manager{
		config: &config.Config{
			Backup: config.BackupConfig{Dir: "/var/backups/juiscript"},
		},
	}

	tests := []struct {
		path    string
		wantErr bool
	}{
		{"/var/backups/juiscript/test.tar.gz", false},
		{"/var/backups/juiscript/subdir/test.tar.gz", false},
		{"/etc/cron.d/juiscript-evil", true},
		{"/var/backups/juiscript/../../../etc/passwd", true},
		{"/tmp/test.tar.gz", true},
	}

	for _, tt := range tests {
		err := m.isInsideBackupDir(tt.path)
		if (err != nil) != tt.wantErr {
			t.Errorf("isInsideBackupDir(%q) err = %v, wantErr %v", tt.path, err, tt.wantErr)
		}
	}
}
