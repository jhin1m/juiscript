package firewall

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/jhin1m/juiscript/internal/system"
)

// jailNameRe validates fail2ban jail names (alphanumeric, dashes, underscores).
var jailNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// UFWRule represents a single UFW firewall rule.
type UFWRule struct {
	Num    int    // Rule number from `ufw status numbered`
	To     string // e.g. "22/tcp", "80/tcp", "443"
	Action string // "ALLOW IN", "DENY IN", etc.
	From   string // "Anywhere", specific IP, etc.
}

// UFWStatus holds UFW state and rules.
type UFWStatus struct {
	Active bool
	Rules  []UFWRule
}

// F2bJailStatus holds fail2ban jail info.
type F2bJailStatus struct {
	Name      string
	BannedIPs []string
	BanCount  int
}

// Manager wraps UFW and Fail2ban commands for firewall management.
type Manager struct {
	executor system.Executor
}

// NewManager creates a firewall manager.
func NewManager(exec system.Executor) *Manager {
	return &Manager{executor: exec}
}

// --- Input validation ---

// validatePort checks port is 1-65535.
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be 1-65535, got %d", port)
	}
	return nil
}

// validateIP checks IP address format (IPv4 or IPv6).
func validateIP(ip string) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %q", ip)
	}
	return nil
}

// validateProtocol checks protocol is tcp, udp, or both.
func validateProtocol(proto string) error {
	switch proto {
	case "tcp", "udp", "both", "":
		return nil
	default:
		return fmt.Errorf("protocol must be tcp, udp, or both; got %q", proto)
	}
}

// portTarget builds the port/proto string for UFW commands.
func portTarget(port int, proto string) string {
	s := strconv.Itoa(port)
	if proto != "" && proto != "both" {
		s += "/" + proto
	}
	return s
}

// --- UFW operations ---

// Status returns current UFW status and rules.
func (m *Manager) Status(ctx context.Context) (*UFWStatus, error) {
	out, err := m.executor.Run(ctx, "ufw", "status", "numbered")
	if err != nil {
		return nil, fmt.Errorf("ufw status: %w", err)
	}
	return parseUFWStatus(out), nil
}

// Enable enables UFW firewall.
func (m *Manager) Enable(ctx context.Context) error {
	_, err := m.executor.RunWithInput(ctx, "y\n", "ufw", "enable")
	if err != nil {
		return fmt.Errorf("ufw enable: %w", err)
	}
	return nil
}

// Disable disables UFW firewall.
func (m *Manager) Disable(ctx context.Context) error {
	_, err := m.executor.Run(ctx, "ufw", "disable")
	if err != nil {
		return fmt.Errorf("ufw disable: %w", err)
	}
	return nil
}

// AllowPort adds a UFW allow rule for a port.
func (m *Manager) AllowPort(ctx context.Context, port int, proto string) error {
	if err := validatePort(port); err != nil {
		return err
	}
	if err := validateProtocol(proto); err != nil {
		return err
	}
	target := portTarget(port, proto)
	_, err := m.executor.Run(ctx, "ufw", "allow", target)
	if err != nil {
		return fmt.Errorf("ufw allow %s: %w", target, err)
	}
	return nil
}

// DenyPort adds a UFW deny rule for a port.
func (m *Manager) DenyPort(ctx context.Context, port int, proto string) error {
	if err := validatePort(port); err != nil {
		return err
	}
	if err := validateProtocol(proto); err != nil {
		return err
	}
	target := portTarget(port, proto)
	_, err := m.executor.Run(ctx, "ufw", "deny", target)
	if err != nil {
		return fmt.Errorf("ufw deny %s: %w", target, err)
	}
	return nil
}

// DeleteRule removes a UFW rule by number.
func (m *Manager) DeleteRule(ctx context.Context, ruleNum int) error {
	if ruleNum < 1 {
		return fmt.Errorf("rule number must be >= 1, got %d", ruleNum)
	}
	num := strconv.Itoa(ruleNum)
	_, err := m.executor.RunWithInput(ctx, "y\n", "ufw", "delete", num)
	if err != nil {
		return fmt.Errorf("ufw delete rule %d: %w", ruleNum, err)
	}
	return nil
}

// --- UFW output parsing ---

// parseUFWStatus parses `ufw status numbered` output.
func parseUFWStatus(raw string) *UFWStatus {
	status := &UFWStatus{}
	lines := strings.Split(raw, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check active/inactive status
		if strings.HasPrefix(trimmed, "Status:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "Status:"))
			status.Active = val == "active"
			continue
		}

		// Parse numbered rules: [ N] ...
		if !strings.HasPrefix(trimmed, "[") {
			continue
		}

		rule := parseUFWRule(trimmed)
		if rule != nil {
			status.Rules = append(status.Rules, *rule)
		}
	}
	return status
}

// parseUFWRule parses a single line like "[ 1] 22/tcp  ALLOW IN  Anywhere"
func parseUFWRule(line string) *UFWRule {
	closeBracket := strings.Index(line, "]")
	if closeBracket < 0 {
		return nil
	}
	numStr := strings.TrimSpace(line[1:closeBracket])
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return nil
	}

	rest := strings.TrimSpace(line[closeBracket+1:])
	fields := strings.Fields(rest)
	if len(fields) < 3 {
		return nil
	}

	return &UFWRule{
		Num:    num,
		To:     fields[0],
		Action: strings.Join(fields[1:len(fields)-1], " "),
		From:   fields[len(fields)-1],
	}
}

// --- Fail2ban operations ---

// F2bStatus returns status of all fail2ban jails.
func (m *Manager) F2bStatus(ctx context.Context) ([]F2bJailStatus, error) {
	out, err := m.executor.Run(ctx, "fail2ban-client", "status")
	if err != nil {
		return nil, fmt.Errorf("fail2ban status: %w", err)
	}

	jails := parseF2bJailList(out)
	var results []F2bJailStatus
	for _, jail := range jails {
		js, err := m.jailStatus(ctx, jail)
		if err != nil {
			// Graceful: include jail with zero info on error
			results = append(results, F2bJailStatus{Name: jail})
			continue
		}
		results = append(results, *js)
	}
	return results, nil
}

// jailStatus gets detailed status for a single jail.
func (m *Manager) jailStatus(ctx context.Context, jail string) (*F2bJailStatus, error) {
	out, err := m.executor.Run(ctx, "fail2ban-client", "status", jail)
	if err != nil {
		return nil, fmt.Errorf("fail2ban jail %s: %w", jail, err)
	}
	return parseF2bJailStatus(jail, out), nil
}

// validateJail checks jail name contains only safe characters.
func validateJail(jail string) error {
	if !jailNameRe.MatchString(jail) {
		return fmt.Errorf("invalid jail name: %q", jail)
	}
	return nil
}

// BanIP bans an IP in a fail2ban jail.
func (m *Manager) BanIP(ctx context.Context, ip, jail string) error {
	if err := validateIP(ip); err != nil {
		return err
	}
	if jail == "" {
		jail = "sshd"
	}
	if err := validateJail(jail); err != nil {
		return err
	}
	_, err := m.executor.Run(ctx, "fail2ban-client", "set", jail, "banip", ip)
	if err != nil {
		return fmt.Errorf("ban %s in %s: %w", ip, jail, err)
	}
	return nil
}

// UnbanIP unbans an IP from a fail2ban jail.
func (m *Manager) UnbanIP(ctx context.Context, ip, jail string) error {
	if err := validateIP(ip); err != nil {
		return err
	}
	if jail == "" {
		jail = "sshd"
	}
	if err := validateJail(jail); err != nil {
		return err
	}
	_, err := m.executor.Run(ctx, "fail2ban-client", "set", jail, "unbanip", ip)
	if err != nil {
		return fmt.Errorf("unban %s from %s: %w", ip, jail, err)
	}
	return nil
}

// --- Fail2ban output parsing ---

// parseF2bJailList parses `fail2ban-client status` to extract jail names.
func parseF2bJailList(raw string) []string {
	for _, line := range strings.Split(raw, "\n") {
		if strings.Contains(line, "Jail list:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) < 2 {
				return nil
			}
			jailStr := strings.TrimSpace(parts[1])
			if jailStr == "" {
				return nil
			}
			var jails []string
			for _, j := range strings.Split(jailStr, ",") {
				j = strings.TrimSpace(j)
				if j != "" {
					jails = append(jails, j)
				}
			}
			return jails
		}
	}
	return nil
}

// parseF2bJailStatus parses `fail2ban-client status <jail>` output.
func parseF2bJailStatus(jail, raw string) *F2bJailStatus {
	js := &F2bJailStatus{Name: jail}
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "Currently banned:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				js.BanCount, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
			}
		}
		if strings.Contains(trimmed, "Banned IP list:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				ipStr := strings.TrimSpace(parts[1])
				if ipStr != "" {
					js.BannedIPs = strings.Fields(ipStr)
				}
			}
		}
	}
	return js
}
