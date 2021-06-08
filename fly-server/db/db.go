package db

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
)

type User struct {
	Username string
	Password []byte
	Chroot   string
	Admin    bool
}

type access rune

const (
	Denied access = '_'
	Read   access = 'R'
	Write  access = 'W'
)

type Policy struct {
	Name       string
	Users      []string
	Paths      []string
	FileAccess access
	AcpAccess  access
}

type Handle struct {
	dir      string
	err      error
	users    map[string]User
	policies []Policy
	lock     sync.RWMutex
}

type RTxn struct {
	db *Handle
}

type Txn struct {
	db *Handle
}

var ErrNotFound = errors.New("not found")

func Open(dir string) (*Handle, error) {
	db := &Handle{
		dir:      dir,
		users:    make(map[string]User, 0),
		policies: make([]Policy, 0),
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

func (tx *RTxn) FetchAllUsers() []User {
	users := make([]User, 0, len(tx.db.users))

	for _, u := range tx.db.users {
		users = append(users, u)
	}

	return users
}

func (tx *Txn) NumUsers() int {
	return len(tx.db.users)
}

func (tx *RTxn) NumUsers() int {
	return len(tx.db.users)
}

func (tx *Txn) FindUser(username string) (user User, found bool) {
	return tx.db.findUser(username)
}

func (tx *RTxn) FindUser(username string) (user User, found bool) {
	return tx.db.findUser(username)
}

func (db *Handle) findUser(username string) (user User, found bool) {
	user, found = db.users[username]
	return
}

func (tx *Txn) AddUser(u *User) error {
	tx.db.users[u.Username] = *u
	tx.db.writeUsers()

	return tx.db.err
}

func (tx *Txn) AddAccessPolicy(p *Policy) error {
	tx.db.policies = append(tx.db.policies, *p)
	tx.db.writeAccessPolicies()

	return tx.db.err
}

func (tx *RTxn) FetchAllPolicies() []Policy {
	policies := make([]Policy, 0, len(tx.db.policies))

	for _, p := range tx.db.policies {
		policies = append(policies, p)
	}

	return policies
}

func (tx *Txn) UpdateUser(username string, f func(u *User)) error {
	user, ok := tx.db.users[username]

	if !ok {
		return fmt.Errorf("user %w: %s", ErrNotFound, username)
	}

	f(&user)
	tx.db.users[username] = user
	tx.db.writeUsers()

	return tx.db.err
}

func (tx *Txn) DeleteUser(username string) error {
	if _, ok := tx.db.users[username]; !ok {
		return fmt.Errorf("user %w: %s", ErrNotFound, username)
	}

	delete(tx.db.users, username)
	tx.db.writeUsers()

	return tx.db.err
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

func (db *Handle) readUsers() {
	if db.err != nil {
		return
	}

	dbPath := path.Join(db.dir, ".fly/users.csv")
	f, err := os.Open(dbPath)

	if err != nil {
		db.err = fmt.Errorf("Could not open the FlyDB users table: %w", err)
		return
	}

	defer f.Close()
	csv := csv.NewReader(f)
	csv.ReuseRecord = true
	csv.FieldsPerRecord = 4

	// Skip the header
	_, err = csv.Read()

	if err != nil {
		db.err = fmt.Errorf("Could not read header from the FlyDB users table: %w", err)
		return
	}

	for {
		record, err := csv.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			db.err = fmt.Errorf("Could not read from the FlyDB users table: %w", err)
			return
		}

		// @TODO: make sure the username is valid
		// @TODO: quickly check the password hashes with bcrypt.Cost()
		// @TODO: make sure the chroot is a valid path
		// @TODO: make sure admin is either "1" or "0"
		newuser := User{
			Username: record[0],
			Password: []byte(record[1]),
			Chroot:   record[2],
			Admin:    record[3] == "1",
		}

		db.users[newuser.Username] = newuser
	}
}

func (db *Handle) writeUsers() {
	if db.err != nil {
		return
	}

	tmpPath := path.Join(db.dir, ".fly/users.csv~")
	f, err := os.OpenFile(tmpPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		db.err = fmt.Errorf("Could not open the FlyDB user table for writing: %w", err)
		return
	}

	defer f.Close()
	csv := csv.NewWriter(f)

	if err := csv.Write([]string{"username", "password", "chroot", "admin"}); err != nil {
		db.err = fmt.Errorf("Could not write the header to the FlyDB user table: %w", err)
		return
	}

	records := make([][]string, len(db.users))
	i := 0

	for _, user := range db.users {
		admin := "0"

		if user.Admin {
			admin = "1"
		}

		records[i] = []string{
			user.Username,
			string(user.Password),
			user.Chroot,
			admin,
		}

		i++
	}

	if err = csv.WriteAll(records); err != nil {
		db.err = fmt.Errorf("Could not write records to the FlyDB user table: %w", err)
		return
	}

	finalPath := strings.TrimRight(tmpPath, "~")

	if err = os.Rename(tmpPath, finalPath); err != nil {
		db.err = fmt.Errorf("Could not commit the FlyDB user table: %w", err)
		return
	}
}

func (db *Handle) readAccessPolicies() {
	if db.err != nil {
		return
	}

	dbPath := path.Join(db.dir, ".fly/acp.csv")
	f, err := os.Open(dbPath)

	if err != nil {
		db.err = fmt.Errorf("Could not open the FlyDB ACP table: %w", err)
		return
	}

	defer f.Close()
	csv := csv.NewReader(f)
	csv.ReuseRecord = true
	csv.FieldsPerRecord = 4

	// Skip the header
	_, err = csv.Read()

	if err != nil {
		db.err = fmt.Errorf("Could not read header from the FlyDB ACP table: %w", err)
		return
	}

	for lineNum := 1; true; lineNum++ {
		record, err := csv.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			db.err = fmt.Errorf("Could not read from the FlyDB ACP table: %w", err)
			return
		}

		if len(record[3]) != 2 {
			db.err = fmt.Errorf("Corrupted FlyDB ACP table: invalid access bits at line %d", lineNum)
			return
		}

		fileAccess := rune(record[3][0])
		acpAccess := rune(record[3][1])

		db.policies = append(db.policies, Policy{
			Name:       record[0],
			Users:      parsePolicyUsers(record[1]),
			Paths:      parsePolicyPaths(record[2]),
			FileAccess: access(fileAccess),
			AcpAccess:  access(acpAccess),
		})

		// @TODO: validate integrity
	}
}

func (db *Handle) writeAccessPolicies() {
	if db.err != nil {
		return
	}

	tmpPath := path.Join(db.dir, ".fly/acp.csv~")
	f, err := os.OpenFile(tmpPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		db.err = fmt.Errorf("Could not open FlyDB ACP table for writing: %w", err)
		return
	}

	defer f.Close()
	csv := csv.NewWriter(f)

	if err := csv.Write([]string{"rule", "users", "paths", "allow"}); err != nil {
		db.err = fmt.Errorf("Could not write header to the FlyDB ACP table: %w", err)
		return
	}

	records := make([][]string, len(db.policies))
	i := 0

	for _, rule := range db.policies {
		userList := "*"

		if rule.Users != nil {
			userList = strings.Join(rule.Users, ":")
		}

		sanitizePaths(&rule)

		records[i] = []string{
			rule.Name,
			userList,
			strings.Join(rule.Paths, ":"),
			string([]rune{rune(rule.FileAccess), rune(rule.AcpAccess)}),
		}

		i++
	}

	if err = csv.WriteAll(records); err != nil {
		db.err = fmt.Errorf("Could not write records to the FlyDB ACP table: %w", err)
		return
	}

	finalPath := strings.TrimRight(tmpPath, "~")

	if err = os.Rename(tmpPath, finalPath); err != nil {
		db.err = fmt.Errorf("Could not finalize writing to the FlyDB ACP table: %w", err)
		return
	}
}

func sanitizePaths(policy *Policy) {
	for i, p := range policy.Paths {
		p = strings.ReplaceAll(p, "%", "%25")
		p = strings.ReplaceAll(p, ":", "%3A")
		policy.Paths[i] = p
	}
}

func parsePolicyUsers(s string) []string {
	if s == "*" {
		return nil
	}

	return strings.Split(s, ":")
}

// @TODO: QueryUnescape could return error
func parsePolicyPaths(s string) []string {
	paths := strings.Split(s, ":")

	for i, p := range paths {
		paths[i], _ = url.QueryUnescape(p)
	}

	return paths
}
