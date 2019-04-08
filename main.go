package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type author struct {
	ID   string
	Name string
}

func makeAuthor(name string) *author {
	id := uuid.New().String()
	return &author{
		ID:   id,
		Name: name,
	}
}

func (a *author) equals(aa *author) bool {
	return aa != nil && a.ID == aa.ID && a.Name == aa.Name
}

type note struct {
	ID      string
	Title   string
	Content string
	Author  *author
}

func makeNote(title string, content string, auth *author) *note {
	id := uuid.New().String()
	return &note{
		ID:      id,
		Title:   title,
		Content: content,
		Author:  auth,
	}
}

func (n *note) equals(nn *note) bool {
	return nn != nil &&
		n.ID == nn.ID &&
		n.Title == nn.Title &&
		n.Content == nn.Content &&
		n.Author.equals(nn.Author)
}

type key int

const (
	requestIDKey key = 0
)

var (
	listenAddr string

	tmpl = make(map[string]*template.Template)
)

func main() {
	tmpl["note"] = template.Must(template.ParseFiles("templates/note.html", "templates/_base.html"))
	tmpl["notes"] = template.Must(template.ParseFiles("templates/notes.html", "templates/_base.html"))

	flag.StringVar(&listenAddr, "listen-addr", ":8080", "server listen address")
	flag.Parse()

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Println("Server is starting...")

	c, err := makeConn()
	if err != nil {
		log.Fatalf("could start sqlite: %v", err)
	}
	defer c.close()

	router := http.NewServeMux()
	router.Handle("/notes", notes(c))
	router.Handle("/healthz", health())
	router.Handle("/static/", static(logger))

	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      tracing(nextRequestID)(logging(logger)(router)),
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Could not listen on %s: %v\n", listenAddr, err)
	}
}

func notes(c *conn) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		id := r.URL.Query().Get("note_id")
		if id == "" {
			notes, err := c.getNotes()
			if err != nil {
				fmt.Fprintf(w, "error: %v", err)
				return
			}

			tmpl["notes"].ExecuteTemplate(w, "base", notes)
		} else {
			note, err := c.getNote(id)
			if err != nil {
				fmt.Fprintf(w, "error: %v", err)
				return
			}

			tmpl["note"].ExecuteTemplate(w, "base", note)
		}
	})
}

func health() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func static(logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(".", r.URL.Path)

		var contentType string
		if strings.HasSuffix(path, ".css") {
			contentType = "text/css"
		} else {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		f, err := os.Open(path)

		if err == nil {
			logger.Println(path)
			defer f.Close()
			w.Header().Add("Content-Type", contentType)

			br := bufio.NewReader(f)
			br.WriteTo(w)
			return
		} else {
			logger.Printf("%v\n", err)
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func tracing(nextRequestID func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-Id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
