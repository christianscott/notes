package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

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

	var noteID, title, content, authorID, authorName string
	for rows.Next() {
		err = rows.Scan(&noteID, &title, &content, &authorID, &authorName)
		a := &author{
			ID:   authorID,
			Name: authorName,
		}
		notes = append(notes, &note{
			ID:      noteID,
			Title:   title,
			Content: content,
			Author:  a,
		})
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return notes, nil
}
