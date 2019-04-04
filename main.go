package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
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

	notesTmpl = template.Must(template.ParseFiles("templates/notes.html"))
	noteTmpl  = template.Must(template.ParseFiles("templates/note.html"))
)

func main() {
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
	router.Handle("/", notes(c))

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

			notesTmpl.Execute(w, notes)
		} else {
			note, err := c.getNote(id)
			if err != nil {
				fmt.Fprintf(w, "error: %v", err)
				return
			}

			noteTmpl.Execute(w, note)
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
