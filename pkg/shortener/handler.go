package shortener

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type handler struct {
	config *Config
	db     *Database
	logger *log.Logger
}

const maxBodySize = 8192

func getRemoteHost(remoteAddr string) string {
	remoteHost, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return remoteHost
}

func (h *handler) getClientHost(req *http.Request) string {
	if !h.config.TrustProxy {
		return getRemoteHost(req.RemoteAddr)
	}

	xForwardedFor := req.Header.Get("X-Forwarded-For")
	if xForwardedFor == "" {
		return getRemoteHost(req.RemoteAddr)
	}

	host := strings.TrimSpace(strings.Split(xForwardedFor, ",")[0])
	if host != "" {
		return host
	}

	return getRemoteHost(req.RemoteAddr)
}

func isValidHttpUrl(input string) bool {
	url, err := url.ParseRequestURI(input)
	if err != nil {
		return false
	}
	if url.Scheme != "https" && url.Scheme != "http" {
		return false
	}
	return true
}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		if req.URL.Path == "" || req.URL.Path == "/" {
			h.HomepageHandler(w, req)
		} else {
			h.RedirectHandler(w, req)
		}
	case "POST":
		h.CreateCodeHandler(w, req)
	default:
		h.logger.Printf("%s %s %s Method not allowed",
			h.getClientHost(req), req.Method, req.URL)
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
	}
}

func (h *handler) HomepageHandler(w http.ResponseWriter, req *http.Request) {
	h.logger.Printf("%s %s %s",
		h.getClientHost(req), req.Method, req.URL)
	if h.config.MainPage == "" {
		fmt.Fprintf(w, "hello, world\n")
	} else {
		http.Redirect(w, req, h.config.MainPage, http.StatusFound)
	}
}

func (h *handler) RedirectHandler(w http.ResponseWriter, req *http.Request) {
	code := req.URL.Path[1:]

	url, err := h.db.GetUrl(code)
	if err != nil {
		if err == ErrNotFound {
			h.logger.Printf("%s %s %s [Not found]",
				h.getClientHost(req), req.Method, req.URL)
			http.Error(w, "Not found", http.StatusNotFound)
		} else {
			h.logger.Printf("%s %s %s [%v]",
				h.getClientHost(req), req.Method, req.URL, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	h.logger.Printf("%s %s %s => %s",
		h.getClientHost(req), req.Method, req.URL, url)
	http.Redirect(w, req, url, http.StatusFound)
}

func (h *handler) CreateCodeHandler(w http.ResponseWriter, req *http.Request) {
	var customCode string
	if req.URL.Path != "" && req.URL.Path != "/" {
		customCode = req.URL.Path[1:]
	}

	var username string
	if h.config.Auth {
		authHeader := req.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			h.logger.Printf("%s %s %s [Missing credentials]",
				h.getClientHost(req), req.Method, req.URL)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		apiKey := strings.TrimPrefix(authHeader, "Bearer ")
		var err error
		username, err = h.db.CheckApiKey(apiKey)
		if err == ErrNotFound {
			h.logger.Printf("%s %s %s [Invalid credentials]",
				h.getClientHost(req), req.Method, req.URL)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if err != nil {
			h.logger.Printf("%s %s %s [Error checking credentials: %v]",
				h.getClientHost(req), req.Method, req.URL, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	clientHost := h.getClientHost(req)
	createdBy := clientHost
	if h.config.Auth {
		createdBy = username
	}

	req.Body = http.MaxBytesReader(w, req.Body, maxBodySize)
	body, err := io.ReadAll(req.Body)
	if err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			h.logger.Printf("%s %s %s [Request body too large]",
				clientHost, req.Method, req.URL)
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		h.logger.Printf("%s %s %s [%v]",
			clientHost, req.Method, req.URL, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	url := strings.TrimSpace(string(body))

	if !isValidHttpUrl(url) {
		h.logger.Printf("%s %s %s [Invalid URL]",
			clientHost, req.Method, req.URL)
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	if customCode != "" && !isValidCode(customCode) {
		h.logger.Printf("%s %s %s [Invalid code]",
			clientHost, req.Method, req.URL)
		http.Error(w, "Invalid code", http.StatusBadRequest)
		return
	}

	var code string
	if customCode == "" {
		for attempt := range 3 {
			code, err = generateCode(h.config.CodeLength)
			if err != nil {
				h.logger.Printf("%s %s %s [Attempt %d: %s: %v]",
					clientHost, req.Method, req.URL, attempt, code, err)
				code = ""
				continue
			}

			err = h.db.CreateCode(url, code, createdBy)
			if err != nil {
				if err == ErrCodeAlreadyInUse {
					h.logger.Printf("%s %s %s [Attempt %d: %s: Code already in use]",
						clientHost, req.Method, req.URL, attempt, code)
				} else {
					h.logger.Printf("%s %s %s [Attempt %d: %s: %v]",
						clientHost, req.Method, req.URL, attempt, code, err)
				}
				code = ""
				continue
			}

			break
		}

		if code == "" {
			h.logger.Printf("%s %s %s [Could not generate code]",
				clientHost, req.Method, req.URL)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		err = h.db.CreateCode(url, customCode, createdBy)
		if err != nil {
			if err == ErrCodeAlreadyInUse {
				h.logger.Printf("%s %s %s [Code already in use]",
					clientHost, req.Method, req.URL)
				http.Error(w, "Code already in use", http.StatusConflict)
			} else {
				h.logger.Printf("%s %s %s [%v]",
					clientHost, req.Method, req.URL, err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}

		code = customCode
	}

	newUrl := fmt.Sprintf("%s%s", h.config.UrlPrefix, code)
	h.logger.Printf("%s %s %s (%s) => %s",
		clientHost, req.Method, req.URL, url, newUrl)
	fmt.Fprintf(w, "%s\n", newUrl)
}
