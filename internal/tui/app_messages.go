package tui

import (
	"github.com/jhin1m/juiscript/internal/backup"
	"github.com/jhin1m/juiscript/internal/cache"
	"github.com/jhin1m/juiscript/internal/database"
	"github.com/jhin1m/juiscript/internal/firewall"
	"github.com/jhin1m/juiscript/internal/nginx"
	"github.com/jhin1m/juiscript/internal/site"
	"github.com/jhin1m/juiscript/internal/ssl"
	"github.com/jhin1m/juiscript/internal/supervisor"
)

// -- Site result messages --

// SiteListMsg delivers the list of sites after async fetch.
type SiteListMsg struct{ Sites []*site.Site }

// SiteListErrMsg reports failure to list sites.
type SiteListErrMsg struct{ Err error }

// SiteCreatedMsg signals a site was successfully created.
type SiteCreatedMsg struct{ Site *site.Site }

// SiteDetailMsg delivers a single site's data for the detail screen.
type SiteDetailMsg struct{ Site *site.Site }

// SiteOpDoneMsg signals a site operation (toggle/delete) succeeded.
type SiteOpDoneMsg struct{}

// SiteOpErrMsg reports a failed site operation.
type SiteOpErrMsg struct{ Err error }

// -- Nginx result messages --

// VhostListMsg delivers the vhost list after async fetch.
type VhostListMsg struct{ Vhosts []nginx.VhostInfo }

// VhostListErrMsg reports failure to list vhosts.
type VhostListErrMsg struct{ Err error }

// NginxOpDoneMsg signals an nginx operation succeeded.
type NginxOpDoneMsg struct{}

// NginxOpErrMsg reports a failed nginx operation.
type NginxOpErrMsg struct{ Err error }

// NginxTestOkMsg signals nginx config test passed.
type NginxTestOkMsg struct{ Output string }

// -- Database result messages --

// DBListMsg delivers the database list after async fetch.
type DBListMsg struct{ Databases []database.DBInfo }

// DBListErrMsg reports failure to list databases.
type DBListErrMsg struct{ Err error }

// DBOpDoneMsg signals a database operation succeeded.
type DBOpDoneMsg struct{}

// DBOpErrMsg reports a failed database operation.
type DBOpErrMsg struct{ Err error }

// -- SSL result messages --

// CertListMsg delivers the certificate list after async fetch.
type CertListMsg struct{ Certs []ssl.CertInfo }

// CertListErrMsg reports failure to list certificates.
type CertListErrMsg struct{ Err error }

// SSLOpDoneMsg signals an SSL operation succeeded.
type SSLOpDoneMsg struct{}

// SSLOpErrMsg reports a failed SSL operation.
type SSLOpErrMsg struct{ Err error }

// -- Service result messages --
// Note: ServiceStatusMsg and ServiceStatusErrMsg already exist in app.go.

// ServiceOpDoneMsg signals a service action succeeded.
type ServiceOpDoneMsg struct{}

// ServiceOpErrMsg reports a failed service action.
type ServiceOpErrMsg struct{ Err error }

// -- Queue worker result messages --

// WorkerListMsg delivers the worker list after async fetch.
type WorkerListMsg struct{ Workers []supervisor.WorkerStatus }

// WorkerListErrMsg reports failure to list workers.
type WorkerListErrMsg struct{ Err error }

// QueueOpDoneMsg signals a queue operation succeeded.
type QueueOpDoneMsg struct{}

// QueueOpErrMsg reports a failed queue operation.
type QueueOpErrMsg struct{ Err error }

// -- Backup result messages --

// BackupListMsg delivers the backup list after async fetch.
type BackupListMsg struct{ Backups []backup.BackupInfo }

// BackupListErrMsg reports failure to list backups.
type BackupListErrMsg struct{ Err error }

// BackupOpDoneMsg signals a backup operation succeeded.
type BackupOpDoneMsg struct{}

// BackupOpErrMsg reports a failed backup operation.
type BackupOpErrMsg struct{ Err error }

// -- Firewall result messages --

// FirewallStatusMsg delivers UFW + Fail2ban status.
type FirewallStatusMsg struct {
	UFW   *firewall.UFWStatus
	Jails []firewall.F2bJailStatus
}

// FirewallStatusErrMsg reports failure to fetch firewall status.
type FirewallStatusErrMsg struct{ Err error }

// FirewallOpDoneMsg signals a firewall operation succeeded.
type FirewallOpDoneMsg struct{}

// FirewallOpErrMsg reports a failed firewall operation.
type FirewallOpErrMsg struct{ Err error }

// -- Cache result messages --

// CacheStatusMsg delivers Redis/Opcache status.
type CacheStatusMsg struct {
	Status *cache.CacheStatus
}

// CacheStatusErrMsg reports failure to fetch cache status.
type CacheStatusErrMsg struct{ Err error }

// CacheOpDoneMsg signals a cache operation succeeded.
type CacheOpDoneMsg struct{}

// CacheOpErrMsg reports a failed cache operation.
type CacheOpErrMsg struct{ Err error }
