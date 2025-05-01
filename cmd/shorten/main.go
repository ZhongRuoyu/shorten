package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ZhongRuoyu/shorten/pkg/shortener"
)

var (
	listenPort = flag.Int("listen-port", 8080,
		"Port to listen for HTTP requests")
	urlPrefix = flag.String("url-prefix", "http://localhost:8080/",
		"Prefix to shortened URL, e.g. https://example.com/")
	mainPage = flag.String("main-page", "",
		"URL for main page of shortener; leave blank for default home page")
	codeLength = flag.Int("code-length", 6,
		"Length of shortened code")
	sqliteDb = flag.String("sqlite-db", "urls.db",
		"Path to SQLite database for URL storage")
	logFile = flag.String("log-file", "access.log",
		"Path to access log file")
)

func loadConfig() *shortener.Config {
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Usage:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *urlPrefix == "" || *urlPrefix == "http://localhost:8080/" {
		if *listenPort != 80 {
			*urlPrefix = fmt.Sprintf("http://localhost:%d/", *listenPort)
		} else {
			*urlPrefix = "http://localhost/"
		}
	}

	return &shortener.Config{
		ListenPort: *listenPort,
		UrlPrefix:  *urlPrefix,
		MainPage:   *mainPage,
		CodeLength: *codeLength,
		SqliteDb:   *sqliteDb,
		LogFile:    *logFile,
	}
}

func main() {
	config := loadConfig()
	logFile, err := os.OpenFile(config.LogFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		logger := log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lmicroseconds)
		logger.Fatalf("Failed to open log file: %v", err)
	}

	writer := io.MultiWriter(os.Stdout, logFile)
	logger := log.New(writer, "", log.Ldate|log.Ltime|log.Lmicroseconds)
	logger.Printf("Config: %+v", *config)

	shortener, err := shortener.NewShortener(config, logger)
	if err != nil {
		logger.Fatalf("Failed to create shortener: %v", err)
	}
	defer func() {
		if err := shortener.Close(); err != nil {
			logger.Printf("Failed to cleanup: %v", err)
		}
	}()

	logger.Println("Starting HTTP server")
	logger.Fatal(shortener.ListenAndServe())
}
