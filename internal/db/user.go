package db

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username string
	Password []byte
	Chroot   string
	Admin    bool
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

func ValidateUsername(username string) bool {
	matched, err := regexp.Match("^[a-z_]([a-z0-9_-]{0,31})$", []byte(username))
	return matched && err == nil
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
