package main_test

import (
	main "abhemanyus/imagedatabase"
	"database/sql"
	"os"
	"testing"
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
		ErrHandler(t, err, false)
	})
	t.Run("fail to create duplicate tag", func(t *testing.T) {
		err := testDB.CreateTag("sfw", "safe for work")
		ErrHandler(t, err, true)
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

func ErrHandler(t testing.TB, got error, want bool) {
	t.Helper()
	if want {
		if got == nil {
			t.Fatal("want error, got nil")
		} else {
			t.Logf("want error, got %v", got)
		}
	} else {
		if got == nil {
			t.Log("no errors")
		} else {
			t.Logf("got error %v", got)
		}
	}
}
