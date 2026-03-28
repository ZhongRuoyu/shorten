package shortener

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

type Shortener struct {
	handler *handler
}

func NewShortener(config *Config, logger *log.Logger) (*Shortener, error) {
	db, err := NewDatabase(config.SqliteDb)
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
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.handler.config.ListenPort),
		Handler:      s.handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return server.ListenAndServe()
}
