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
	db     *database
	logger *log.Logger
}

func getClientHost(req *http.Request) string {
	remoteAddr := req.RemoteAddr
	remoteHost, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		remoteHost = remoteAddr
	}

	xForwardedFor := req.Header.Get("X-Forwarded-For")
	if xForwardedFor == "" {
		return remoteHost
	}

	host := strings.TrimSpace(strings.Split(xForwardedFor, ",")[0])
	if host != "" {
		return host
	}

	return remoteHost
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
			getClientHost(req), req.Method, req.URL)
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
	}
}

func (h *handler) HomepageHandler(w http.ResponseWriter, req *http.Request) {
	h.logger.Printf("%s %s %s",
		getClientHost(req), req.Method, req.URL)
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
		if err == errNotFound {
			h.logger.Printf("%s %s %s [Not found]",
				getClientHost(req), req.Method, req.URL)
			http.Error(w, "Not found", http.StatusNotFound)
		} else {
			h.logger.Printf("%s %s %s [%v]",
				getClientHost(req), req.Method, req.URL, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	h.logger.Printf("%s %s %s => %s",
		getClientHost(req), req.Method, req.URL, url)
	http.Redirect(w, req, url, http.StatusFound)
}

func (h *handler) CreateCodeHandler(w http.ResponseWriter, req *http.Request) {
	var customCode string
	if req.URL.Path != "" && req.URL.Path != "/" {
		customCode = req.URL.Path[1:]
	}

	clientHost := getClientHost(req)

	body, err := io.ReadAll(req.Body)
	if err != nil {
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

			err = h.db.CreateCode(url, code, clientHost)
			if err != nil {
				if err == errCodeAlreadyInUse {
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
		err = h.db.CreateCode(url, customCode, clientHost)
		if err != nil {
			if err == errCodeAlreadyInUse {
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
