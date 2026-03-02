package system

import (
	"context"
	"fmt"
	"os/user"
	"strconv"
)

// UserManager handles Linux user creation/deletion for site isolation.
// Each website gets its own Linux user for security isolation.
type UserManager interface {
	Create(username, homeDir string) error
	Delete(username string) error
	Exists(username string) bool
	// LookupUID returns the numeric UID for a username.
	LookupUID(username string) (int, int, error)
}

type userMgmt struct {
	exec Executor
}

// NewUserManager creates a user manager backed by real system commands.
func NewUserManager(exec Executor) UserManager {
	return &userMgmt{exec: exec}
}

func (u *userMgmt) Create(username, homeDir string) error {
	if u.Exists(username) {
		return fmt.Errorf("user already exists: %s", username)
	}

	// Create system user with home dir, bash shell, no login password
	// -m: create home dir, -d: home dir path, -s: shell
	_, err := u.exec.Run(context.Background(),
		"useradd",
		"-m",
		"-d", homeDir,
		"-s", "/bin/bash",
		username,
	)
	if err != nil {
		return fmt.Errorf("create user %s: %w", username, err)
	}

	return nil
}

func (u *userMgmt) Delete(username string) error {
	if !u.Exists(username) {
		return nil // already gone, idempotent
	}

	// -r: remove home dir and mail spool
	_, err := u.exec.Run(context.Background(),
		"userdel",
		"-r",
		username,
	)
	if err != nil {
		return fmt.Errorf("delete user %s: %w", username, err)
	}

	return nil
}

func (u *userMgmt) Exists(username string) bool {
	_, err := user.Lookup(username)
	return err == nil
}

func (u *userMgmt) LookupUID(username string) (int, int, error) {
	usr, err := user.Lookup(username)
	if err != nil {
		return 0, 0, fmt.Errorf("lookup user %s: %w", username, err)
	}

	uid, err := strconv.Atoi(usr.Uid)
	if err != nil {
		return 0, 0, fmt.Errorf("parse uid: %w", err)
	}

	gid, err := strconv.Atoi(usr.Gid)
	if err != nil {
		return 0, 0, fmt.Errorf("parse gid: %w", err)
	}

	return uid, gid, nil
}
