package shortener

import (
	"fmt"
	"log"
	"net/http"
)

type Shortener struct {
	handler *handler
}

func NewShortener(config *Config, logger *log.Logger) (*Shortener, error) {
	db, err := newDatabase(config.SqliteDb)
	if err != nil {
		return nil, fmt.Errorf("shortener: failed to create database: %v", err)
	}

	err = db.Init()
	if err != nil {
		return nil, fmt.Errorf("shortener: failed to initialize database: %v", err)
	}

	return &Shortener{
		handler: &handler{
			config: config,
			db:     db,
			logger: logger,
		}}, nil
}

func (s *Shortener) Close() error {
	if s.handler.db != nil {
		return s.handler.db.db.Close()
	}
	return nil
}

func (s *Shortener) ListenAndServe() error {
	return http.ListenAndServe(
		fmt.Sprintf(":%d", s.handler.config.ListenPort), s.handler)
}
