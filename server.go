package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/corona10/goimagehash"
	"github.com/mattn/go-sqlite3"
	_ "golang.org/x/image/webp"
)

type Server struct {
	*http.ServeMux
	DB   Store
	Root string
}

func CreateServer(db Store, root string) (*Server, error) {
	log.SetFlags(log.Lshortfile)
	_, err := os.Stat(root)
	if errors.Is(err, fs.ErrNotExist) {
		err = os.Mkdir(root, 0o777)
		if err != nil {
			return nil, err
		}
	}
	server := &Server{http.NewServeMux(), db, root}
	server.HandleFunc("/file", server.handleFile)
	server.HandleFunc("/url", server.handleUrl)
	return server, nil
}

func (srv *Server) handleFile(w http.ResponseWriter, r *http.Request) {
	tag := r.URL.Query().Get("tag")
	header := r.Header.Get("content-type")
	file := r.Body
	defer file.Close()
	imgExt := strings.Split(strings.ToLower(header), "/")
	log.Println(imgExt)
	if imgExt[0] != "image" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	destPath, size, err := srv.saveFile(file, imgExt[1])
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err)
		return
	}
	hashString, err := srv.addToDB(destPath, tag, size)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err)
		return
	}
	var image struct {
		Hash int64
		Size int64
	}
	image.Hash = hashString
	image.Size = size
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(image)
}

func (srv *Server) handleUrl(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	tag := r.URL.Query().Get("tag")
	if len(tag) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "no tag found in query")
		return
	}
	myUrl := r.Form.Get("url")

	if len(myUrl) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "no url found in form")
		return
	}
	log.Println(myUrl)
	dhash, _ := srv.DB.FindUrl(myUrl)
	if dhash == 0 {
		w.WriteHeader(http.StatusAlreadyReported)
		fmt.Fprint(w, "url already exists")
		return
	}

	res, err := http.Get(myUrl)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, err)
		return
	}

	if res.StatusCode != http.StatusOK {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, res.Status)
		return
	}
	imgExt := strings.Split(strings.ToLower(res.Header.Get("content-type")), "/")
	log.Println(imgExt[0], imgExt[1])
	if imgExt[0] != "image" {
		log.Println("processing html")
		myUrls, err := GetPixivImage(res.Body)
		log.Printf("%+v", myUrls)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, err)
			return
		}
		myUrl = myUrls[0]
		log.Println(myUrl)
		dhash, _ = srv.DB.FindUrl(myUrl)
		if dhash == 0 {
			w.WriteHeader(http.StatusAlreadyReported)
			fmt.Fprint(w, "url already exists")
			return
		}
		req, _ := http.NewRequest(http.MethodGet, myUrl, nil)
		req.Header = map[string][]string{
			"Accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/jxl,image/webp,*/*;q=0.8"},
			"Accept-Encoding":           {"gzip, deflate, br"},
			"Accept-Language":           {"en-US,en;q=0.5"},
			"Host":                      {"i.pximg.net"},
			"Referer":                   {"https://www.pixiv.net/"},
			"Sec-Fetch-Dest":            {"document"},
			"Sec-Fetch-Mode":            {"navigate"},
			"Sec-Fetch-Site":            {"cross-site"},
			"Sec-Fetch-User":            {"?1"},
			"TE":                        {"trailers"},
			"Upgrade-Insecure-Requests": {"1"},
			"User-Agent":                {"Mozilla/5.0 (X11; Linux x86_64; rv:91.0) Gecko/20100101 Firefox/91.0 Waterfox/91.5.0"},
		}
		res, err = http.DefaultClient.Do(req)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, err)
			return
		}
		imgExt = strings.Split(strings.ToLower(res.Header.Get("content-type")), "/")
		log.Println(imgExt[0], imgExt[1])
	}

	if imgExt[0] == "image" {
		dest, size, err := srv.saveFile(res.Body, imgExt[1])
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "unable to save file")
			return
		}
		var image struct {
			Hash int64
			Size int64
		}
		hashString, err := srv.addToDB(dest, tag, size)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "unable to add to db, %v", err)
			return
		}
		srv.DB.AddUrl(hashString, myUrl)
		image.Hash = hashString
		image.Size = size
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(image)
		return
	}
	log.Println(res.Status)
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprint(w, "url not an image nor pixiv link")
}

func (srv *Server) saveFile(src io.ReadCloser, ext string) (string, int64, error) {
	defer src.Close()
	tempName := fmt.Sprintf("%x.%s", time.Now().UnixNano(), ext)
	destPath := path.Join(srv.Root, tempName)
	destFile, err := os.Create(destPath)
	defer destFile.Close()
	if err != nil {
		return "", 0, err
	}
	size, err := io.Copy(destFile, src)
	return destPath, size, err
}

func (srv *Server) addToDB(filePath, tag string, size int64) (int64, error) {
	file, _ := os.Open(filePath)
	image, _, err := image.Decode(file)
	if err != nil {
		return 0, err
	}
	// set sensitivity of hashing algorithm
	dhash, err := goimagehash.DifferenceHash(image)
	if err != nil {
		return 0, err
	}
	hashInt := int64(dhash.GetHash())
	err = srv.DB.Add(hashInt, filePath, size)
	if err != nil {
		log.Println(err)
		// convert to sqlite error
		sqlErr, ok := err.(sqlite3.Error)
		if ok {
			// hash match cause primary key error
			if sqlErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey {
				// get old image record
				image, _ := srv.DB.Find(hashInt)
				// if new image > old image
				if size > image.Size {
					// remove old image
					os.Remove(image.Path)
					// add new image
					srv.DB.Add(hashInt, filePath, size)
				} else {
					os.Remove(filePath)
				}
			}
		}
	}
	// add tag
	err = srv.DB.AddTag(hashInt, tag)
	return hashInt, err
}
