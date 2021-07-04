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
	"regexp"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username string
	Password []byte
	Chroot   string
	Admin    bool
}

type Action rune
type Verb string

const (
	Allow Verb = "ALLOW"
	Deny  Verb = "DENY"
)

const (
	Read  Action = 'R'
	Write Action = 'W'
)

type Policy struct {
	Verb   Verb
	Action Action
	Name   string
	Users  []string
	Paths  []string
}

type Handle struct {
	dir      string
	err      error
	users    map[string]User
	policies map[string]Policy
	lock     sync.RWMutex
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
	if _, found := tx.db.users[u.Username]; found {
		return ErrExists
	}

	tx.db.users[u.Username] = *u
	tx.db.writeUsers()

	return tx.db.err
}

func (tx *Txn) PutAccessPolicy(p *Policy) error {
	tx.db.policies[p.Name] = *p
	tx.db.writeAccessPolicies()

	return tx.db.err
}

func (tx *RTxn) GetPolicies(path string, username string, action Action) []Policy {
	policies := make([]Policy, 0)

	for _, p := range tx.db.policies {
		if matchesPolicy(path, username, action, &p) {
			policies = append(policies, p)
		}
	}

	return policies
}

func matchesPolicy(path string, username string, action Action, policy *Policy) bool {
	return policy.Action == action &&
		matchesPath(path, policy) &&
		matchesUser(username, policy)
}

func matchesPath(path string, policy *Policy) bool {
	for _, prefix := range policy.Paths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

func matchesUser(username string, policy *Policy) bool {
	for _, user := range policy.Users {
		if user == username {
			return true
		}
	}

	return false
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

func (tx *Txn) DeleteAccessPolicy(name string) error {
	if _, ok := tx.db.policies[name]; !ok {
		return fmt.Errorf("policy %w: %s", ErrNotFound, name)
	}

	delete(tx.db.policies, name)
	tx.db.writeAccessPolicies()

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

		newuser := User{
			Username: record[0],
			Password: []byte(record[1]),
			Chroot:   record[2],
			Admin:    record[3] == "1",
		}

		db.users[newuser.Username] = newuser

		if !ValidateUsername(newuser.Username) {
			db.err = fmt.Errorf("Corrupted FlyDB users table. Invalid username: %s", newuser.Username)
			return
		}

		if record[3] != "0" && record[3] != "1" {
			db.err = fmt.Errorf("Corrupted FlyDB users table. Invalid admin bit: %s", record[3])
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(newuser.Password), []byte("secret"))

		if err != nil && err != bcrypt.ErrMismatchedHashAndPassword {
			db.err = fmt.Errorf("Corrupted FlyDB users table. Invalid password hash: %s", newuser.Password)
			return
		}
	}
}

func ValidateUsername(username string) bool {
	matched, err := regexp.Match("^[a-z_]([a-z0-9_-]{0,31})$", []byte(username))
	return matched && err == nil
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
	csv.FieldsPerRecord = 5

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

		if len(strings.TrimSpace(record[0])) == 0 {
			db.err = fmt.Errorf("Corrupted FlyDB ACP table: missing ACP name at line %d", lineNum)
			return
		}

		if record[1] != "ALLOW" && record[1] != "DENY" {
			db.err = fmt.Errorf("Corrupted FlyDB ACP table: invalid verb (ALLOW/DENY) at line %d", lineNum)
			return
		}

		if record[2] != "R" && record[2] != "W" {
			db.err = fmt.Errorf("Corrupted FlyDB ACP table: invalid action (R/W) at line %d", lineNum)
			return
		}

		paths, err := parsePolicyPaths(record[4], lineNum)

		if err != nil {
			db.err = err
			return
		}

		db.policies[record[0]] = Policy{
			Name:   record[0],
			Verb:   Verb(record[1]),
			Action: Action(record[2][0]),
			Users:  parsePolicyUsers(record[3]),
			Paths:  paths,
		}
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

	if err := csv.Write([]string{"rule", "verb", "action", "users", "paths"}); err != nil {
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
			string(rule.Verb),
			string(rule.Action),
			userList,
			strings.Join(rule.Paths, ":"),
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
	return strings.Split(s, ":")
}

func parsePolicyPaths(s string, lineNum int) ([]string, error) {
	paths := strings.Split(s, ":")

	var err error

	for i, p := range paths {
		paths[i], err = url.QueryUnescape(p)

		if err != nil {
			return nil, fmt.Errorf("Corrupted FlyDB ACP table: invalid path %s at line %d", p, lineNum)
		}
	}

	return paths, nil
}
