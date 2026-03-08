package firewall

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// mockExecutor simulates UFW/Fail2ban commands for testing.
type mockExecutor struct {
	commands []string
	failOn   map[string]error
	outputs  map[string]string
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{
		failOn:  make(map[string]error),
		outputs: make(map[string]string),
	}
}

func (m *mockExecutor) Run(_ context.Context, name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	m.commands = append(m.commands, cmd)

	if output, ok := m.outputs[cmd]; ok {
		if err, ok := m.failOn[cmd]; ok {
			return output, err
		}
		return output, nil
	}
	if err, ok := m.failOn[cmd]; ok {
		return "", err
	}
	return "", nil
}

func (m *mockExecutor) RunWithInput(_ context.Context, _ string, name string, args ...string) (string, error) {
	return m.Run(context.Background(), name, args...)
}

func (m *mockExecutor) hasCommand(substr string) bool {
	for _, cmd := range m.commands {
		if strings.Contains(cmd, substr) {
			return true
		}
	}
	return false
}

// --- Validation tests ---

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"valid min", 1, false},
		{"valid max", 65535, false},
		{"valid http", 80, false},
		{"valid https", 443, false},
		{"zero", 0, true},
		{"negative", -1, true},
		{"too high", 65536, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePort(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePort(%d) error = %v, wantErr = %v", tt.port, err, tt.wantErr)
			}
		})
	}
}

func TestValidateIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{"valid ipv4", "192.168.1.1", false},
		{"valid ipv4 private", "10.0.0.1", false},
		{"valid ipv6 loopback", "::1", false},
		{"valid ipv6", "2001:db8::1", false},
		{"invalid text", "not-an-ip", true},
		{"invalid octets", "999.999.999.999", true},
		{"empty string", "", true},
		{"injection attempt", "1.2.3.4; rm -rf /", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIP(tt.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateIP(%q) error = %v, wantErr = %v", tt.ip, err, tt.wantErr)
			}
		})
	}
}

func TestValidateProtocol(t *testing.T) {
	tests := []struct {
		proto   string
		wantErr bool
	}{
		{"tcp", false},
		{"udp", false},
		{"both", false},
		{"", false},
		{"icmp", true},
		{"invalid", true},
	}
	for _, tt := range tests {
		t.Run(tt.proto, func(t *testing.T) {
			err := validateProtocol(tt.proto)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateProtocol(%q) error = %v, wantErr = %v", tt.proto, err, tt.wantErr)
			}
		})
	}
}

// --- portTarget tests ---

func TestPortTarget(t *testing.T) {
	tests := []struct {
		port  int
		proto string
		want  string
	}{
		{80, "tcp", "80/tcp"},
		{443, "udp", "443/udp"},
		{22, "both", "22"},
		{8080, "", "8080"},
	}
	for _, tt := range tests {
		got := portTarget(tt.port, tt.proto)
		if got != tt.want {
			t.Errorf("portTarget(%d, %q) = %q, want %q", tt.port, tt.proto, got, tt.want)
		}
	}
}

// --- UFW parsing tests ---

func TestParseUFWStatus_Active(t *testing.T) {
	raw := `Status: active

     To                         Action      From
     --                         ------      ----
[ 1] 22/tcp                     ALLOW IN    Anywhere
[ 2] 80/tcp                     ALLOW IN    Anywhere
[ 3] 443/tcp                    ALLOW IN    Anywhere`

	status := parseUFWStatus(raw)
	if !status.Active {
		t.Error("expected active")
	}
	if len(status.Rules) != 3 {
		t.Fatalf("got %d rules, want 3", len(status.Rules))
	}
	if status.Rules[0].Num != 1 || status.Rules[0].To != "22/tcp" {
		t.Errorf("rule 1: num=%d to=%q", status.Rules[0].Num, status.Rules[0].To)
	}
	if status.Rules[1].Num != 2 || status.Rules[1].To != "80/tcp" {
		t.Errorf("rule 2: num=%d to=%q", status.Rules[1].Num, status.Rules[1].To)
	}
	if status.Rules[2].Action != "ALLOW IN" {
		t.Errorf("rule 3 action=%q", status.Rules[2].Action)
	}
	if status.Rules[0].From != "Anywhere" {
		t.Errorf("rule 1 from=%q", status.Rules[0].From)
	}
}

func TestParseUFWStatus_Inactive(t *testing.T) {
	raw := "Status: inactive"
	status := parseUFWStatus(raw)
	if status.Active {
		t.Error("expected inactive")
	}
	if len(status.Rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(status.Rules))
	}
}

func TestParseUFWStatus_DenyRule(t *testing.T) {
	raw := `Status: active

     To                         Action      From
     --                         ------      ----
[ 1] 3306/tcp                   DENY IN     Anywhere`

	status := parseUFWStatus(raw)
	if len(status.Rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(status.Rules))
	}
	if status.Rules[0].Action != "DENY IN" {
		t.Errorf("expected DENY IN, got %q", status.Rules[0].Action)
	}
}

func TestParseUFWRule_Invalid(t *testing.T) {
	// No bracket
	if parseUFWRule("no bracket here") != nil {
		t.Error("expected nil for line without bracket")
	}
	// Bad number
	if parseUFWRule("[abc] stuff") != nil {
		t.Error("expected nil for non-numeric rule number")
	}
	// Too few fields
	if parseUFWRule("[ 1] only") != nil {
		t.Error("expected nil for too few fields")
	}
}

// --- Fail2ban parsing tests ---

func TestParseF2bJailList(t *testing.T) {
	raw := `Status
|- Number of jail:      2
` + "`" + `- Jail list:   sshd, nginx-http-auth`

	jails := parseF2bJailList(raw)
	if len(jails) != 2 {
		t.Fatalf("got %d jails, want 2", len(jails))
	}
	if jails[0] != "sshd" {
		t.Errorf("jail 0 = %q", jails[0])
	}
	if jails[1] != "nginx-http-auth" {
		t.Errorf("jail 1 = %q", jails[1])
	}
}

func TestParseF2bJailList_Empty(t *testing.T) {
	raw := `Status
|- Number of jail:      0
` + "`" + `- Jail list:`

	jails := parseF2bJailList(raw)
	if len(jails) != 0 {
		t.Errorf("expected 0 jails, got %d", len(jails))
	}
}

func TestParseF2bJailStatus(t *testing.T) {
	raw := `Status for the jail: sshd
|- Filter
|  |- Currently failed: 5
|  ` + "`" + `- Total failed:     20
` + "`" + `- Actions
   |- Currently banned: 3
   ` + "`" + `- Banned IP list:   1.2.3.4 5.6.7.8 9.10.11.12`

	js := parseF2bJailStatus("sshd", raw)
	if js.Name != "sshd" {
		t.Errorf("name = %q", js.Name)
	}
	if js.BanCount != 3 {
		t.Errorf("ban count = %d, want 3", js.BanCount)
	}
	if len(js.BannedIPs) != 3 {
		t.Fatalf("banned IPs count = %d, want 3", len(js.BannedIPs))
	}
	if js.BannedIPs[0] != "1.2.3.4" {
		t.Errorf("first banned IP = %q", js.BannedIPs[0])
	}
}

func TestParseF2bJailStatus_NoBanned(t *testing.T) {
	raw := `Status for the jail: sshd
|- Filter
` + "`" + `- Actions
   |- Currently banned: 0
   ` + "`" + `- Banned IP list:`

	js := parseF2bJailStatus("sshd", raw)
	if js.BanCount != 0 {
		t.Errorf("ban count = %d", js.BanCount)
	}
	if len(js.BannedIPs) != 0 {
		t.Errorf("expected 0 banned IPs, got %d", len(js.BannedIPs))
	}
}

// --- UFW operation tests ---

func TestAllowPort_Success(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.AllowPort(context.Background(), 80, "tcp")
	if err != nil {
		t.Fatalf("AllowPort failed: %v", err)
	}
	if !exec.hasCommand("ufw allow 80/tcp") {
		t.Error("expected ufw allow 80/tcp")
	}
}

func TestAllowPort_InvalidPort(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.AllowPort(context.Background(), 0, "tcp")
	if err == nil {
		t.Fatal("expected error for port 0")
	}
}

func TestAllowPort_InvalidProtocol(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.AllowPort(context.Background(), 80, "icmp")
	if err == nil {
		t.Fatal("expected error for invalid protocol")
	}
}

func TestDenyPort_Success(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.DenyPort(context.Background(), 3306, "tcp")
	if err != nil {
		t.Fatalf("DenyPort failed: %v", err)
	}
	if !exec.hasCommand("ufw deny 3306/tcp") {
		t.Error("expected ufw deny 3306/tcp")
	}
}

func TestDeleteRule_Success(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.DeleteRule(context.Background(), 1)
	if err != nil {
		t.Fatalf("DeleteRule failed: %v", err)
	}
	if !exec.hasCommand("ufw delete 1") {
		t.Error("expected ufw delete 1")
	}
}

func TestDeleteRule_InvalidNumber(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.DeleteRule(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error for rule number 0")
	}
}

func TestStatus_Success(t *testing.T) {
	exec := newMockExecutor()
	exec.outputs["ufw status numbered"] = `Status: active

     To                         Action      From
     --                         ------      ----
[ 1] 22/tcp                     ALLOW IN    Anywhere`
	mgr := NewManager(exec)

	status, err := mgr.Status(context.Background())
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if !status.Active {
		t.Error("expected active")
	}
	if len(status.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(status.Rules))
	}
}

func TestStatus_Error(t *testing.T) {
	exec := newMockExecutor()
	exec.failOn["ufw status numbered"] = fmt.Errorf("ufw not found")
	mgr := NewManager(exec)

	_, err := mgr.Status(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEnable(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.Enable(context.Background())
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}
	if !exec.hasCommand("ufw enable") {
		t.Error("expected ufw enable")
	}
}

func TestDisable(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.Disable(context.Background())
	if err != nil {
		t.Fatalf("Disable failed: %v", err)
	}
	if !exec.hasCommand("ufw disable") {
		t.Error("expected ufw disable")
	}
}

// --- Fail2ban operation tests ---

func TestBanIP_Success(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.BanIP(context.Background(), "1.2.3.4", "sshd")
	if err != nil {
		t.Fatalf("BanIP failed: %v", err)
	}
	if !exec.hasCommand("fail2ban-client set sshd banip 1.2.3.4") {
		t.Error("expected fail2ban-client set sshd banip 1.2.3.4")
	}
}

func TestBanIP_DefaultJail(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.BanIP(context.Background(), "10.0.0.1", "")
	if err != nil {
		t.Fatalf("BanIP failed: %v", err)
	}
	if !exec.hasCommand("fail2ban-client set sshd banip 10.0.0.1") {
		t.Error("expected default jail sshd")
	}
}

func TestBanIP_InvalidIP(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.BanIP(context.Background(), "not-an-ip", "sshd")
	if err == nil {
		t.Fatal("expected error for invalid IP")
	}
}

func TestUnbanIP_Success(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.UnbanIP(context.Background(), "1.2.3.4", "sshd")
	if err != nil {
		t.Fatalf("UnbanIP failed: %v", err)
	}
	if !exec.hasCommand("fail2ban-client set sshd unbanip 1.2.3.4") {
		t.Error("expected fail2ban-client set sshd unbanip 1.2.3.4")
	}
}

func TestF2bStatus_Success(t *testing.T) {
	exec := newMockExecutor()
	exec.outputs["fail2ban-client status"] = `Status
|- Number of jail:      1
` + "`" + `- Jail list:   sshd`

	exec.outputs["fail2ban-client status sshd"] = `Status for the jail: sshd
|- Filter
` + "`" + `- Actions
   |- Currently banned: 2
   ` + "`" + `- Banned IP list:   1.2.3.4 5.6.7.8`

	mgr := NewManager(exec)
	jails, err := mgr.F2bStatus(context.Background())
	if err != nil {
		t.Fatalf("F2bStatus failed: %v", err)
	}
	if len(jails) != 1 {
		t.Fatalf("expected 1 jail, got %d", len(jails))
	}
	if jails[0].Name != "sshd" {
		t.Errorf("jail name = %q", jails[0].Name)
	}
	if jails[0].BanCount != 2 {
		t.Errorf("ban count = %d", jails[0].BanCount)
	}
	if len(jails[0].BannedIPs) != 2 {
		t.Errorf("banned IPs = %d", len(jails[0].BannedIPs))
	}
}

func TestF2bStatus_NotInstalled(t *testing.T) {
	exec := newMockExecutor()
	exec.failOn["fail2ban-client status"] = fmt.Errorf("command not found")
	mgr := NewManager(exec)

	_, err := mgr.F2bStatus(context.Background())
	if err == nil {
		t.Fatal("expected error when fail2ban not installed")
	}
}
