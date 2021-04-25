package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

type user struct {
	username string
	password []byte
	chroot   string
	admin    bool
}

type access rune

const (
	Denied access = '_'
	Read   access = 'R'
	Write  access = 'W'
)

type acp struct {
	name       string
	users      []string
	paths      []string
	fileAccess access
	acpAccess  access
}

func readDatabase() {
	users = make(map[string]user, 0)
	policies = make([]acp, 0)

	if found := readVersionFile(); found {
		readUserDb()
		readAccessDb()
		singleUser = len(users) == 0
	} else {
		dbFolder := path.Join(dir, ".fly")

		if err := os.MkdirAll(dbFolder, 0700); err != nil {
			fmt.Println("ERROR: could not create FlyDB folder:", err)
			os.Exit(1)
		}

		writeVersionFile()
		writeUserDb()
		writeAccessDb()

		singleUser = true
	}
}

func readVersionFile() (found bool) {
	versionPath := path.Join(dir, ".fly/version")
	version, err := os.ReadFile(versionPath)

	if errors.Is(err, os.ErrNotExist) {
		found = false
		return
	}

	if err != nil {
		fmt.Println("ERROR: could not open version file:", err)
		os.Exit(1)
	}

	found = true
	version = bytes.TrimSpace(version)

	if string(version) != "1" {
		fmt.Println("Unexpected FlyDB version:", version)
		os.Exit(1)
	}

	return
}

func writeVersionFile() {
	versionPath := path.Join(dir, ".fly/version")

	if err := os.WriteFile(versionPath, []byte("1\n"), 0600); err != nil {
		fmt.Println("ERROR: could not create new FlyDB file:", err)
		os.Exit(1)
	}
}

func readUserDb() {
	dbPath := path.Join(dir, ".fly/users.csv")
	f, err := os.Open(dbPath)

	if err != nil {
		abortDbCorrupt("users.csv", err)
	}

	defer f.Close()
	csv := csv.NewReader(f)
	csv.ReuseRecord = true
	csv.FieldsPerRecord = 4

	// Skip the header
	_, err = csv.Read()

	if err != nil {
		abortDbCorrupt("users.csv", err)
	}

	for {
		record, err := csv.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			abortDbCorrupt("users.csv", err)
		}

		newuser := user{
			username: record[0],
			password: []byte(record[1]),
			chroot:   record[2],
			admin:    record[3] == "1",
		}

		users[newuser.username] = newuser
	}

	// @TODO: validate integrity?
}

func writeUserDb() {
	tmpPath := path.Join(dir, ".fly/users.csv~")
	f, err := os.OpenFile(tmpPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		fmt.Println("ERROR: couldn't open FlyDB for writing:", err)
		os.Exit(1)
	}

	defer f.Close()
	csv := csv.NewWriter(f)

	if err := csv.Write([]string{"username", "password", "chroot", "admin"}); err != nil {
		fmt.Println("ERROR: couldn't write to FlyDB:", err)
		os.Exit(1)
	}

	records := make([][]string, len(users))
	i := 0

	for _, user := range users {
		admin := "0"

		if user.admin {
			admin = "1"
		}

		records[i] = []string{
			user.username,
			string(user.password),
			user.chroot,
			admin,
		}

		i++
	}

	if err = csv.WriteAll(records); err != nil {
		fmt.Println("ERROR: couldn't write to FlyDB:", err)
		os.Exit(1)
	}

	finalPath := strings.TrimRight(tmpPath, "~")

	if err = os.Rename(tmpPath, finalPath); err != nil {
		fmt.Println("ERROR: couldn't write to FlyDB:", err)
		os.Exit(1)
	}
}

func readAccessDb() {
	dbPath := path.Join(dir, ".fly/acp.csv")
	f, err := os.Open(dbPath)

	if err != nil {
		abortDbCorrupt("acp.csv", err)
	}

	defer f.Close()
	csv := csv.NewReader(f)
	csv.ReuseRecord = true
	csv.FieldsPerRecord = 4

	// Skip the header
	_, err = csv.Read()

	if err != nil {
		abortDbCorrupt("acp.csv", err)
	}

	for {
		record, err := csv.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			abortDbCorrupt("acp.csv", err)
		}

		if len(record[3]) != 2 {
			abortDbCorrupt("acp.csv", errors.New("Invalid ACL"))
		}

		fileAccess := rune(record[3][0])
		acpAccess := rune(record[3][1])

		policies = append(policies, acp{
			name:       record[0],
			users:      parseAcpUsers(record[1]),
			paths:      strings.Split(record[2], ":"),
			fileAccess: access(fileAccess),
			acpAccess:  access(acpAccess),
		})
	}

	// @TODO: validate integrity?
}

func writeAccessDb() {
	tmpPath := path.Join(dir, ".fly/acp.csv~")
	f, err := os.OpenFile(tmpPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		fmt.Println("ERROR: couldn't open FlyDB for writing:", err)
		os.Exit(1)
	}

	defer f.Close()
	csv := csv.NewWriter(f)

	if err := csv.Write([]string{"rule", "users", "paths", "allow"}); err != nil {
		fmt.Println("ERROR: couldn't write to FlyDB:", err)
		os.Exit(1)
	}

	records := make([][]string, len(policies))
	i := 0

	for _, rule := range policies {
		userList := "*"

		if rule.users != nil {
			userList = strings.Join(rule.users, ":")
		}

		records[i] = []string{
			rule.name,
			userList,
			strings.Join(rule.paths, ":"),
			string([]rune{rune(rule.fileAccess), rune(rule.acpAccess)}),
		}

		i++
	}

	if err = csv.WriteAll(records); err != nil {
		fmt.Println("ERROR: couldn't write to FlyDB:", err)
		os.Exit(1)
	}

	finalPath := strings.TrimRight(tmpPath, "~")

	if err = os.Rename(tmpPath, finalPath); err != nil {
		fmt.Println("ERROR: couldn't write to FlyDB:", err)
		os.Exit(1)
	}
}

func parseAcpUsers(s string) []string {
	if s == "*" {
		return nil
	}

	return strings.Split(s, ":")
}

func abortDbCorrupt(fileName string, err error) {
	fmt.Printf("ERROR: FlyDB is corrupted: %s: %s\n", fileName, err)
	os.Exit(1)
}
