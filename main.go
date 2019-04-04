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

	stmt, err := tx.Prepare("insert into notes(note_id, title, content, author_id) values(?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(n.ID, n.Title, n.Content, n.Author.ID)
	if err != nil {
		return err
	}

	tx.Commit()
	return nil
}

func (c *conn) addAuthor(a *author) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("insert into authors(author_id, author_name) values(?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(a.ID, a.Name)
	if err != nil {
		return err
	}

	tx.Commit()
	return nil
}

func (c *conn) getNote(id string) (*note, error) {
	stmt, err := c.db.Prepare(`
		select n.title, n.content, n.author_id, a.author_name
		from notes n join authors a on a.author_id = n.author_id
		where n.note_id = ?
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var title, content, authorID, authorName string
	err = stmt.QueryRow(id).Scan(&title, &content, &authorID, &authorName)
	if err != nil {
		return nil, err
	}

	auth := &author{
		ID:   authorID,
		Name: authorName,
	}
	return &note{
		ID:      id,
		Title:   title,
		Content: content,
		Author:  auth,
	}, nil
}

func (c *conn) getNotes() (notes []*note, err error) {
	rows, err := c.db.Query(`
		select n.note_id, n.title, n.content, a.author_id, a.author_name
		from notes n join authors a on n.author_id = a.author_id
	`)
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

	a := makeAuthor("Christian Scott")
	err = c.addAuthor(a)
	if err != nil {
		log.Fatalf("failed to add author: %v", err)
	}

	n := makeNote("My masterpiece", "hello world", a)
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
