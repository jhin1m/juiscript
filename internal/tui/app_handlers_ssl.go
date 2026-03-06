package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// fetchCerts loads the certificate list asynchronously.
func (a *App) fetchCerts() tea.Cmd {
	if a.sslMgr == nil {
		return nil
	}
	return func() tea.Msg {
		certs, err := a.sslMgr.ListCerts()
		if err != nil {
			return CertListErrMsg{Err: err}
		}
		return CertListMsg{Certs: certs}
	}
}

// handleObtainCert obtains an SSL certificate for a domain.
func (a *App) handleObtainCert(domain, email string) tea.Cmd {
	if a.sslMgr == nil {
		return nil
	}
	// Derive webRoot from config: sitesRoot/site_user/public
	webRoot := fmt.Sprintf("%s/site_%s/public", a.cfg.General.SitesRoot,
		strings.ReplaceAll(domain, ".", "_"))
	return func() tea.Msg {
		if err := a.sslMgr.Obtain(domain, webRoot, email); err != nil {
			return SSLOpErrMsg{Err: err}
		}
		return SSLOpDoneMsg{}
	}
}

// handleRevokeCert revokes an SSL certificate for a domain.
func (a *App) handleRevokeCert(domain string) tea.Cmd {
	if a.sslMgr == nil {
		return nil
	}
	return func() tea.Msg {
		if err := a.sslMgr.Revoke(domain); err != nil {
			return SSLOpErrMsg{Err: err}
		}
		return SSLOpDoneMsg{}
	}
}

// handleRenewCert renews an SSL certificate for a domain.
func (a *App) handleRenewCert(domain string) tea.Cmd {
	if a.sslMgr == nil {
		return nil
	}
	return func() tea.Msg {
		if err := a.sslMgr.Renew(domain); err != nil {
			return SSLOpErrMsg{Err: err}
		}
		return SSLOpDoneMsg{}
	}
}
