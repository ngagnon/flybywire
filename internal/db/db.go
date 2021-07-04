package db

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"path"
	"sync"
)

type Handle struct {
	dir      string
	err      error
	users    map[string]User
	policies map[string]Policy
	lock     sync.RWMutex

	cert     *tls.Certificate
	certLock sync.RWMutex

	GetCertificate func(*tls.ClientHelloInfo) (*tls.Certificate, error)
}

type RTxn struct {
	db *Handle
}

type Txn struct {
	db *Handle
}

var ErrNotFound = errors.New("not found")
var ErrExists = errors.New("already exists")

func Open(dir string) (*Handle, error) {
	db := &Handle{
		dir:      dir,
		users:    make(map[string]User, 0),
		policies: make(map[string]Policy, 0),
	}

	found, err := readVersionFile(dir)

	if err != nil {
		return nil, err
	}

	if found {
		db.readUsers()
		db.readAccessPolicies()
	} else {
		dbFolder := path.Join(dir, ".fly")

		if err := os.MkdirAll(dbFolder, 0700); err != nil {
			return nil, fmt.Errorf("Could not create FlyDB folder: %w", err)
		}

		db.writeVersionFile()
		db.writeUsers()
		db.writeAccessPolicies()
	}

	db.loadTlsCert()
	db.GetCertificate = func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
		db.certLock.RLock()
		defer db.certLock.RUnlock()
		return db.cert, nil
	}

	if db.err != nil {
		return nil, db.err
	}

	return db, nil
}

func (db *Handle) Txn() *Txn {
	db.lock.Lock()
	return &Txn{db: db}
}

func (tx *Txn) Complete() {
	tx.db.lock.Unlock()
}

func (db *Handle) RTxn() *RTxn {
	db.lock.RLock()
	return &RTxn{db: db}
}

func (tx *RTxn) Complete() {
	tx.db.lock.RUnlock()
}

func readVersionFile(dir string) (found bool, err error) {
	versionPath := path.Join(dir, ".fly/version")
	version, err := os.ReadFile(versionPath)

	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("Could not open FlyDB version file: %w", err)
	}

	version = bytes.TrimSpace(version)

	if string(version) != "1" {
		return false, fmt.Errorf("Unexpected FlyDB version: %d", version)
	}

	return true, nil
}

func (db *Handle) writeVersionFile() {
	if db.err != nil {
		return
	}

	versionPath := path.Join(db.dir, ".fly/version")

	if err := os.WriteFile(versionPath, []byte("1\n"), 0600); err != nil {
		db.err = fmt.Errorf("Could not create FlyDB version file: %w", err)
	}
}
