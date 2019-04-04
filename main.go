package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type note struct {
	ID      string
	Title   string
	Content string
}

func makeNote(title string, content string) *note {
	id := uuid.New().String()
	return &note{
		ID:      id,
		Title:   title,
		Content: content,
	}
}

func (n *note) equals(nn *note) bool {
	return nn != nil && n.ID == nn.ID && n.Title == nn.Title && n.Content == nn.Content
}

type conn struct {
	db *sql.DB
}

func makeConn() (*conn, error) {
	db, err := sql.Open("sqlite3", "./notes.db")
	if err != nil {
		return nil, err
	}

	return &conn{db}, nil
}

func (c *conn) close() error {
	return c.db.Close()
}

func (c *conn) addNote(n *note) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("insert into notes(note_id, title, content) values(?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(n.ID, n.Title, n.Content)
	if err != nil {
		return err
	}

	tx.Commit()
	return nil
}

func (c *conn) getNote(id string) (*note, error) {
	stmt, err := c.db.Prepare("select title, content from notes where notes.note_id = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var title, content string
	err = stmt.QueryRow(id).Scan(&title, &content)
	if err != nil {
		return nil, err
	}

	return &note{
		ID:      id,
		Title:   title,
		Content: content,
	}, nil
}

func (c *conn) getNotes() (notes []*note, err error) {
	rows, err := c.db.Query("select note_id, title, content from notes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var title, content, id string
	for rows.Next() {
		err = rows.Scan(&id, &title, &content)
		notes = append(notes, &note{
			ID:      id,
			Title:   title,
			Content: content,
		})
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return notes, nil
}

var notesTmpl = template.Must(template.ParseFiles("templates/notes.html"))
var noteTmpl = template.Must(template.ParseFiles("templates/note.html"))

func main() {
	c, err := makeConn()
	if err != nil {
		log.Fatalf("could start sqlite: %v", err)
	}
	defer c.close()

	n := makeNote("My masterpiece", "hello world")
	err = c.addNote(n)
	if err != nil {
		log.Fatalf("failed to add note: %v", err)
	}

	retrievedNote, err := c.getNote(n.ID)
	if err != nil {
		log.Fatalf("failed to retrieve note: %v", err)
	}

	if n.equals(retrievedNote) {
		log.Print("equal")
	} else {
		log.Fatal("not equal")
	}

	http.HandleFunc("/notes", func(w http.ResponseWriter, r *http.Request) {
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

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	http.ListenAndServe(":8080", nil)
}
