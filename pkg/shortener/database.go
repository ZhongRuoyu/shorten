package shortener

import (
	"database/sql"
	"errors"
	"sync"

	"github.com/mattn/go-sqlite3"
)

type Database struct {
	mu sync.Mutex
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
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`
			PRAGMA foreign_keys = ON;

			CREATE TABLE IF NOT EXISTS Users(
				username TEXT PRIMARY KEY,
				active   INTEGER NOT NULL
			);

			CREATE TABLE IF NOT EXISTS Urls(
				code       TEXT PRIMARY KEY,
				url        TEXT NOT NULL,
				created_at INTEGER NOT NULL,
				created_by TEXT NOT NULL,
				hits       INTEGER NOT NULL,
				last_hit   INTEGER
			);

			CREATE TABLE IF NOT EXISTS ApiKeys(
				key_hash TEXT PRIMARY KEY,
				username TEXT NOT NULL,
				active   INTEGER NOT NULL,
				FOREIGN KEY(username) REFERENCES Users(username)
			);
		`)
	return err
}

func (d *Database) CreateUser(username string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`
			INSERT INTO Users(username, active)
			VALUES (?, 1);
		`, username)
	if err != nil {
		sqliteErr, ok := err.(sqlite3.Error)
		if ok && sqliteErr.Code == sqlite3.ErrConstraint {
			return ErrUsernameAlreadyInUse
		}
		return err
	}

	return nil
}

func (d *Database) ListUsers() ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.db.Query(`
			SELECT username
			FROM Users
			WHERE active = 1;
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			return nil, err
		}
		users = append(users, username)
	}
	return users, rows.Err()
}

func (d *Database) DeleteUser(username string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`
			UPDATE ApiKeys
			SET active = 0
			WHERE username = ?;
		`, username)
	if err != nil {
		return err
	}

	result, err := d.db.Exec(`
			UPDATE Users
			SET active = 0
			WHERE username = ?
			  AND active = 1;
		`, username)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (d *Database) CreateApiKey(username string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var active int
	err := d.db.QueryRow(`
			SELECT active
			FROM Users
			WHERE username = ?
			  AND active = 1;
		`, username).Scan(&active)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrNotFound
		}
		return "", err
	}

	for range 3 {
		key, err := GenerateApiKey()
		if err != nil {
			return "", err
		}

		keyHash, err := HashApiKey(key)
		if err != nil {
			return "", err
		}
		_, err = d.db.Exec(`
				INSERT INTO ApiKeys(key_hash, username, active)
				VALUES (?, ?, 1);
			`, keyHash, username)
		if err != nil {
			sqliteErr, ok := err.(sqlite3.Error)
			if ok && sqliteErr.Code == sqlite3.ErrConstraint {
				continue
			}
			return "", err
		}

		return key, nil
	}

	return "", errors.New("database: could not generate API key")
}

func (d *Database) CheckApiKey(key string) (string, error) {
	keyHash, err := HashApiKey(key)
	if err != nil {
		return "", err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	return d.checkApiKeyByHash(keyHash)
}

func (d *Database) CheckApiKeyByHash(keyHash string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.checkApiKeyByHash(keyHash)
}

func (d *Database) checkApiKeyByHash(keyHash string) (string, error) {
	var username string

	err := d.db.QueryRow(`
			SELECT ak.username
			FROM ApiKeys ak
			JOIN Users u
			  ON ak.username = u.username
			WHERE ak.key_hash = ?
			  AND ak.active = 1
			  AND u.active = 1;
		`, keyHash).Scan(&username)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrNotFound
		}
		return "", err
	}

	return username, nil
}

func (d *Database) ListApiKeys(username string) ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	rows, err := d.db.Query(`
			SELECT key_hash
			FROM ApiKeys
			WHERE username = ?
			  AND active = 1;
		`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

func (d *Database) DeleteApiKey(key string) error {
	keyHash, err := HashApiKey(key)
	if err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	return d.deleteApiKeyByHash(keyHash)
}

func (d *Database) DeleteApiKeyByHash(keyHash string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.deleteApiKeyByHash(keyHash)
}

func (d *Database) deleteApiKeyByHash(keyHash string) error {
	result, err := d.db.Exec(`
			UPDATE ApiKeys
			SET active = 0
			WHERE key_hash = ?
			  AND active = 1;
		`, keyHash)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (d *Database) GetUrl(code string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

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
			SET hits = hits + 1, last_hit = UNIXEPOCH()
			WHERE code = ?;
		`, code)
	if err != nil {
		return "", err
	}

	return url, nil
}

func (d *Database) CreateCode(url string, code string, createdBy string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`
			INSERT INTO Urls(code, url, created_at, created_by, hits, last_hit)
			VALUES (?, ?, UNIXEPOCH(), ?, 0, NULL);
		`, code, url, createdBy)
	if err != nil {
		sqliteErr, ok := err.(sqlite3.Error)
		if ok && sqliteErr.Code == sqlite3.ErrConstraint {
			return ErrCodeAlreadyInUse
		}
		return err
	}

	return nil
}
