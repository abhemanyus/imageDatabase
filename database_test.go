package main_test

import (
	main "abhemanyus/imagedatabase"
	"database/sql"
	"os"
	"testing"

	sqlite "github.com/mattn/go-sqlite3"
)

func TestCreateDB(t *testing.T) {
	testDB, closeDB := OpenDB(t)
	defer closeDB()
	if testDB == nil {
		t.Fatal("database not created")
	}
}

func TestTags(t *testing.T) {
	testDB, closeDB := OpenDB(t)
	defer closeDB()
	t.Run("create tag", func(t *testing.T) {
		err := testDB.CreateTag("sfw", "safe for work")
		ErrHandler(t, err, 0)
	})
	t.Run("fail to create duplicate tag", func(t *testing.T) {
		err := testDB.CreateTag("sfw", "safe for work")
		ErrHandler(t, err, sqlite.ErrConstraintPrimaryKey)
	})
}

func TestImages(t *testing.T) {
	testDB, closeDB := OpenDB(t)
	defer closeDB()
	t.Run("create image", func(t *testing.T) {
		err := testDB.Add("123456", "temp/sfw.png", 200)
		ErrHandler(t, err, 0)
	})
	t.Run("fail to create duplicate dhash", func(t *testing.T) {
		err := testDB.Add("123456", "temp/nsfw.png", 220)
		ErrHandler(t, err, sqlite.ErrConstraintPrimaryKey)
	})
	t.Run("fail to create same path", func(t *testing.T) {
		err := testDB.Add("1234567", "temp/sfw.png", 200)
		ErrHandler(t, err, sqlite.ErrConstraintUnique)
	})
}

func OpenDB(t testing.TB) (main.Store, func()) {
	t.Helper()
	db, err := sql.Open("sqlite3", "test.db")
	if err != nil {
		t.Fatal(err)
	}
	testDB, err := main.CreateDB(db)
	if err != nil {
		t.Fatal(err)
	}
	CloseDB := func() {
		os.Remove("test.db")
	}
	return testDB, CloseDB
}

func ErrHandler(t testing.TB, got error, want sqlite.ErrNoExtended) {
	t.Helper()
	if want == 0 {
		if got != nil {
			t.Fatalf("got error %v", got)
		} else {
			return
		}
	}
	sqlErr, ok := got.(sqlite.Error)
	if !ok {
		t.Fatalf("can't convert to sql error %v", got)
	}
	if sqlErr.ExtendedCode != want {
		t.Fatalf("want error code %d, got %d", want, sqlErr.ExtendedCode)
	}
}
