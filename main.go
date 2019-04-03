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
	title   string
	content string
	id      string
}

func makeNote(title string, content string) *note {
	id := uuid.New().String()
	return &note{title, content, id}
}

func (n *note) equals(nn *note) bool {
	return nn != nil && n.id == nn.id && n.content == nn.content
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

	_, err = stmt.Exec(n.id, n.title, n.content)
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

	return &note{title, content, id}, nil
}

func (c *conn) getNotes() (notes []*note, err error) {
	rows, err := c.db.Query("select * from notes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var title, content, id string
	for rows.Next() {
		err = rows.Scan(&title, &content, &id)
		notes = append(notes, &note{title, content, id})
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return notes, nil
}

var notesTmpl = template.Must(template.ParseFiles("templates/notes.html"))

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

	retrievedNote, err := c.getNote(n.id)
	if err != nil {
		log.Fatalf("failed to retrieve note: %v", err)
	}

	if n.equals(retrievedNote) {
		log.Print("equal")
	} else {
		log.Fatal("not equal")
	}

	http.HandleFunc("/notes", func(w http.ResponseWriter, r *http.Request) {
		notes, err := c.getNotes()
		if err != nil {
			fmt.Fprintf(w, "error: %v", err)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		notesTmpl.Execute(w, notes)
	})

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	http.ListenAndServe(":8080", nil)
}
