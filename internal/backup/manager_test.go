package backup

import (
	"testing"
	"time"

	"github.com/jhin1m/juiscript/internal/config"
)

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
