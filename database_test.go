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

func TestImageTags(t *testing.T) {
	testDB, closeDB := OpenDB(t)
	defer closeDB()
	ErrHandler(t, testDB.Add("123456", "temp/img.png", 222), 0)
	ErrHandler(t, testDB.CreateTag("sfw", "safe for work"), 0)
	t.Run("add tag to image", func(t *testing.T) {
		err := testDB.AddTag("123456", "sfw")
		ErrHandler(t, err, 0)
	})
	t.Run("fail add tag to image", func(t *testing.T) {
		err := testDB.AddTag("123456", "sfw")
		ErrHandler(t, err, sqlite.ErrConstraintPrimaryKey)
	})
	t.Run("get image by tag", func(t *testing.T) {
		images, err := testDB.FindByTags([]string{"sfw"}, 0, 10)
		ErrHandler(t, err, 0)
		if len(images) != 1 {
			t.Fatalf("want length 1, got %v", len(images))
		}
		if images[0].Dhash != "123456" {
			t.Fatalf("want dhash %q, got %q", "123456", images[0].Dhash)
		}
	})
	ErrHandler(t, testDB.CreateTag("nsfw", "NOT safe for work"), 0)
	ErrHandler(t, testDB.AddTag("123456", "nsfw"), 0)
	// t.Run("get image by tags", func(t *testing.T) {
	// 	images, err := testDB.FindByTags([]string{"sfw", "nsfw"}, 0, 10)
	// 	if len(images) != 1 {
	// 		t.Fatalf("want length 1, got %v", len(images))
	// 	}
	// 	if images[0].Dhash != "123456" {
	// 		t.Fatalf("want dhash %q, got %q", "123456", images[0].Dhash)
	// 	}
	// 	ErrHandler(t, err, 0)
	// })
}

func TestUrls(t *testing.T) {
	testDB, closeDB := OpenDB(t)
	defer closeDB()
	testDB.Add("123456", "temp/img.png", 222)
	t.Run("add url", func(t *testing.T) {
		err := testDB.AddUrl("123456", "www.google.com/cats")
		ErrHandler(t, err, 0)
	})
	t.Run("add another url", func(t *testing.T) {
		err := testDB.AddUrl("123456", "www.google.com/dogs")
		ErrHandler(t, err, 0)
	})
	t.Run("fail to add duplicate url", func(t *testing.T) {
		err := testDB.AddUrl("123457", "www.google.com/cats")
		ErrHandler(t, err, sqlite.ErrConstraintPrimaryKey)
	})
	t.Run("get dhash from url", func(t *testing.T) {
		dhash, err := testDB.FindUrl("www.google.com/cats")
		ErrHandler(t, err, 0)
		if dhash != "123456" {
			t.Fatalf("want dhash %q, got %q", "123456", dhash)
		}
	})
	t.Run("on delete cascade", func(t *testing.T) {
		err := testDB.Remove("123456")
		ErrHandler(t, err, 0)
		dhash, err := testDB.FindUrl("www.google.com/cats")
		if err != sql.ErrNoRows {
			t.Fatalf("want error %v, got %v", sql.ErrNoRows, err)
		}
		if dhash != "" {
			t.Fatalf("image url not deleted, got %q", dhash)
		}
	})
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
	t.Run("remove existing tag", func(t *testing.T) {
		err := testDB.RemoveTag("sfw")
		ErrHandler(t, err, 0)
	})
	t.Run("fail to remove non-existing tag", func(t *testing.T) {
		err := testDB.RemoveTag("sfw")
		ErrHandler(t, err, 0)
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
	t.Run("get existing image", func(t *testing.T) {
		image, err := testDB.Find("123456")
		if image.Path != "temp/sfw.png" {
			t.Fatalf("wrong image path %q", image.Path)
		}
		ErrHandler(t, err, 0)
	})
	t.Run("get non-existing image", func(t *testing.T) {
		image, err := testDB.Find("112233")
		if image.Path != "" {
			t.Fatal("image should be nil")
		}
		if err != sql.ErrNoRows {
			t.Fatalf("wanted error %v, got %v", sql.ErrNoRows, err)
		}
	})
	t.Run("remove existing image", func(t *testing.T) {
		err := testDB.Remove("123456")
		if err != nil {
			t.Fatalf("got error %v", err)
		}
	})

	t.Run("fail to remove non-existing image", func(t *testing.T) {
		err := testDB.Remove("123456")
		if err != nil {
			t.Fatalf("got error %v", err)
		}
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
