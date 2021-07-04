package db

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"
)

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

func (tx *Txn) DeleteAccessPolicy(name string) error {
	if _, ok := tx.db.policies[name]; !ok {
		return fmt.Errorf("policy %w: %s", ErrNotFound, name)
	}

	delete(tx.db.policies, name)
	tx.db.writeAccessPolicies()

	return tx.db.err
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
