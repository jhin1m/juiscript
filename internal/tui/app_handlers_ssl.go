package tui

import (
	"fmt"

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

// handleObtainCert is a placeholder -- needs domain + email input form.
func (a *App) handleObtainCert() tea.Cmd {
	if a.sslMgr == nil {
		return nil
	}
	return func() tea.Msg {
		return SSLOpErrMsg{Err: fmt.Errorf("SSL obtain requires domain and email input (not yet implemented)")}
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
