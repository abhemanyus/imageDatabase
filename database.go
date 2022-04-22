package main

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Image struct {
	dhash     string
	path      string
	size      int64
	createdAt time.Time
}

type Database struct {
	rawQuery struct {
		truncateTable,
		insertImage,
		deleteImage,
		findImage,
		addUrl,
		addTag,
		createTag,
		deleteTag,
		getUrls,
		getTags,
		findUrl,
		findByTags *sql.Stmt
	}
}

type Store interface {
	DeleteAll() error
	Add(dhash, path string, size uint) error
	Remove(dhash string) error
	Find(dhash string) (*Image, error)
	AddUrl(dhash, url string) error
	AddTag(dhash, label string) error
	CreateTag(dhash, description string) error
	FindByTags(labels []string, offset, limit int64) (*[]Image, error)
	FindUrl(url string) (string, error)
}

func CreateDB(db *sql.DB) (Store, error) {
	database := &Database{}
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS images (
		dhash TEXT NOT NULL PRIMARY KEY,
		path TEXT NOT NULL UNIQUE,
		size INTEGER NOT NULL DEFAULT 0,
		createdAt DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS tags (
		label TEXT NOT NULL PRIMARY KEY,
		description TEXT DEFAULT "nothing yet"
	);`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS imageUrls (
		url TEXT NOT NULL PRIMARY KEY,
		dhash TEXT NOT NULL,
		FOREIGN KEY (dhash) REFERENCES images (dhash) ON DELETE CASCADE
	);`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`
	CREATE INDEX IF NOT EXISTS imageToUrls ON imageUrls(dhash);`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS imageTags (
		label TEXT NOT NULL,
		dhash TEXT NOT NULL,
		PRIMARY KEY (label, dhash),
		FOREIGN KEY (label) REFERENCES tags (label) ON DELETE CASCADE,
		FOREIGN KEY (dhash) REFERENCES images (dhash) ON DELETE CASCADE
	);`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`
	CREATE INDEX IF NOT EXISTS imageToTags ON imageTags(dhash);`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`
	CREATE INDEX IF NOT EXISTS tagToImages ON imageTags(label)
	`)
	if err != nil {
		return nil, err
	}

	query, err := db.Prepare(`
		DROP TABLE images;
		DROP TABLE tags;
		DROP TABLE imageUrls;
		DROP TABLE imageTags;
	`)
	if err != nil {
		return nil, err
	}
	database.rawQuery.truncateTable = query
	query, err = db.Prepare("INSERT INTO images (dhash, path, size) VALUES (?, ?, ?);")
	if err != nil {
		return nil, err
	}
	database.rawQuery.insertImage = query

	query, err = db.Prepare("DELETE FROM images WHERE dhash = ?;")
	if err != nil {
		return nil, err
	}
	database.rawQuery.deleteImage = query
	query, err = db.Prepare("SELECT dhash, path, size, createdAt FROM images WHERE dhash = ?;")
	if err != nil {
		return nil, err
	}
	database.rawQuery.findImage = query

	query, err = db.Prepare("INSERT INTO imageTags (label, dhash) VALUES (?, ?);")
	if err != nil {
		return nil, err
	}
	database.rawQuery.addTag = query
	query, err = db.Prepare("INSERT INTO imageUrls (url, dhash) VALUES (?, ?);")
	if err != nil {
		return nil, err
	}
	database.rawQuery.addUrl = query
	query, err = db.Prepare("INSERT INTO tags (label, description) VALUES (?, ?);")
	if err != nil {
		return nil, err
	}
	database.rawQuery.createTag = query
	query, err = db.Prepare("SELECT dhash, label FROM imageTags WHERE dhash = ?;")
	if err != nil {
		return nil, err
	}
	database.rawQuery.getTags = query
	query, err = db.Prepare("SELECT dhash, url FROM imageUrls WHERE dhash = ?;")
	if err != nil {
		return nil, err
	}
	database.rawQuery.getUrls = query
	query, err = db.Prepare("SELECT images.dhash, path, size, createdAt FROM imageTags INNER JOIN images ON imageTags.dhash = images.dhash WHERE label IN (?) LIMIT ? OFFSET ?;")
	if err != nil {
		return nil, err
	}
	database.rawQuery.findByTags = query
	query, err = db.Prepare("SELECT dhash FROM imageUrls WHERE url = ?;")
	if err != nil {
		return nil, err
	}
	database.rawQuery.findUrl = query
	return database, err
}

func (db *Database) DeleteAll() error {
	_, err := db.rawQuery.truncateTable.Exec()
	return err
}

func (db *Database) Add(dhash, path string, size uint) error {
	_, err := db.rawQuery.insertImage.Exec(dhash, path, size)
	return err
}

func (db *Database) Remove(dhash string) error {
	_, err := db.rawQuery.insertImage.Exec(dhash)
	return err
}

func (db *Database) Find(dhash string) (*Image, error) {
	var image Image
	err := db.rawQuery.insertImage.QueryRow(dhash).Scan(&image.dhash, &image.path, &image.size, &image.createdAt)
	return &image, err
}

func (db *Database) AddUrl(dhash, url string) error {
	_, err := db.rawQuery.addUrl.Exec(dhash)
	return err
}

func (db *Database) AddTag(dhash, label string) error {
	_, err := db.rawQuery.addTag.Exec(label, dhash)
	return err
}

func (db *Database) CreateTag(label, description string) error {
	_, err := db.rawQuery.createTag.Exec(label, description)
	return err
}

func (db *Database) FindByTags(labels []string, offset, limit int64) (*[]Image, error) {
	rows, err := db.rawQuery.findByTags.Query(labels, limit, offset)
	var images []Image
	if err != nil {
		return &images, err
	}
	for rows.Next() {
		var image Image
		err := rows.Scan(&image.dhash, &image.path, &image.size, &image.createdAt)
		if err == nil {
			images = append(images, image)
		}
	}
	return &images, nil
}

func (db *Database) FindUrl(url string) (string, error) {
	var dhash string
	err := db.rawQuery.findUrl.QueryRow(url).Scan(&dhash)
	return dhash, err
}
