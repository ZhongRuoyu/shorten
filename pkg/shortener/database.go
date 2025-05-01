package shortener

import (
	"database/sql"
	"encoding/base64"
	"errors"

	"github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

var ErrNotFound = errors.New("database: not found")
var ErrUsernameAlreadyInUse = errors.New("database: username already in use")
var ErrCodeAlreadyInUse = errors.New("database: code already in use")

func NewDatabase(dataSource string) (*Database, error) {
	db, err := sql.Open("sqlite3", dataSource)
	if err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}

func (d *Database) Init() error {
	_, err := d.db.Exec(`
			CREATE TABLE IF NOT EXISTS Users(
				username      TEXT PRIMARY KEY,
				salt          TEXT NOT NULL,
				password_hash TEXT NOT NULL
			);

			CREATE TABLE IF NOT EXISTS Urls(
				code       TEXT PRIMARY KEY,
				url        TEXT NOT NULL,
				created_at INTEGER NOT NULL,
				created_by TEXT NOT NULL,
				hits       INTEGER NOT NULL,
				last_hit   INTEGER
			);
		`)
	return err
}

func (d *Database) CreateUser(username, password string) error {
	salt, passwordHash, err := hashPassword(password)
	if err != nil {
		return err
	}

	saltBase64 := base64.StdEncoding.EncodeToString(salt)
	passwordHashBase64 := base64.StdEncoding.EncodeToString(passwordHash)
	_, err = d.db.Exec(`
			INSERT INTO Users(username, salt, password_hash)
			VALUES (?, ?, ?);
		`, username, saltBase64, passwordHashBase64)
	if err != nil {
		sqliteErr, ok := err.(sqlite3.Error)
		if ok {
			if sqliteErr.Code == sqlite3.ErrConstraint {
				return ErrUsernameAlreadyInUse
			}
		}
		return err
	}

	return nil
}

func (d *Database) CheckCredentials(username, password string) (bool, error) {
	var saltBase64, passwordHashBase64 string

	err := d.db.QueryRow(`
			SELECT salt, password_hash
			FROM Users
			WHERE username = ?;
		`, username).Scan(&saltBase64, &passwordHashBase64)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, ErrNotFound
		}
		return false, err
	}

	salt, err := base64.StdEncoding.DecodeString(saltBase64)
	if err != nil {
		return false, err
	}
	passwordHash, err := base64.StdEncoding.DecodeString(passwordHashBase64)
	if err != nil {
		return false, err
	}

	ok, err := checkPasswordHash(password, salt, passwordHash)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (d *Database) UpdateCredentials(username, password string) error {
	salt, passwordHash, err := hashPassword(password)
	if err != nil {
		return err
	}

	saltBase64 := base64.StdEncoding.EncodeToString(salt)
	passwordHashBase64 := base64.StdEncoding.EncodeToString(passwordHash)
	_, err = d.db.Exec(`
			UPDATE Users
			SET salt = ?, password_hash = ?
			WHERE username = ?;
		`, saltBase64, passwordHashBase64, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}

	return nil
}

func (d *Database) DeleteUser(username string) error {
	_, err := d.db.Exec(`
			DELETE FROM Users
			WHERE username = ?;
		`, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}

	return nil
}

func (d *Database) GetUrl(code string) (string, error) {
	var url string

	err := d.db.QueryRow(`
			SELECT url
			FROM Urls
			WHERE code = ?;
		`, code).Scan(&url)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrNotFound
		}
		return "", err
	}

	_, err = d.db.Exec(`
		UPDATE Urls
		SET hits = hits + 1,
				last_hit = UNIXEPOCH()
		WHERE code = ?;
	`, code)
	if err != nil {
		return "", err
	}

	return url, nil
}

func (d *Database) CreateCode(url string, code string, createdBy string) error {
	_, err := d.db.Exec(`
			INSERT INTO Urls(code, url, created_at, created_by, hits, last_hit)
			VALUES (?, ?, UNIXEPOCH(), ?, 0, NULL);
		`, code, url, createdBy)
	if err != nil {
		sqliteErr, ok := err.(sqlite3.Error)
		if ok {
			if sqliteErr.Code == sqlite3.ErrConstraint {
				return ErrCodeAlreadyInUse
			}
		}
		return err
	}

	return nil
}
