package shortener

import (
	"database/sql"
	"errors"

	"github.com/mattn/go-sqlite3"
)

type database struct {
	db *sql.DB
}

var errNotFound = errors.New("database: not found")
var errCodeAlreadyInUse = errors.New("database: code already in use")

func newDatabase(dataSource string) (*database, error) {
	db, err := sql.Open("sqlite3", dataSource)
	if err != nil {
		return nil, err
	}

	return &database{db: db}, nil
}

func (d *database) Init() error {
	_, err := d.db.Exec(`
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

func (d *database) GetUrl(code string) (string, error) {
	var url string

	err := d.db.QueryRow(`
			SELECT url
			FROM Urls
			WHERE code = ?;
		`, code).Scan(&url)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errNotFound
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

func (d *database) CreateCode(url string, code string, createdBy string) error {
	_, err := d.db.Exec(`
			INSERT INTO Urls(code, url, created_at, created_by, hits, last_hit)
			VALUES (?, ?, UNIXEPOCH(), ?, 0, NULL);
		`, code, url, createdBy)
	if err != nil {
		sqliteErr, ok := err.(sqlite3.Error)
		if ok {
			if sqliteErr.Code == sqlite3.ErrConstraint {
				return errCodeAlreadyInUse
			}
		}
		return err
	}

	return nil
}
