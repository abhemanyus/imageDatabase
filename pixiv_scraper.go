package main

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/PuerkitoBio/goquery"
)

type Illustrations struct {
	Key map[string]struct {
		Urls struct {
			Original string `json:"original"`
		} `json:"urls"`
	} `json:"illust"`
}

func GetPixivImage(pixiv_html io.Reader) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(pixiv_html)
	if err != nil {
		return nil, err
	}
	data, ok := doc.Find("#meta-preload-data").Attr("content")
	if !ok {
		return nil, errors.New("content not found")
	}
	var pixiv Illustrations
	err = json.Unmarshal([]byte(data), &pixiv)
	if err != nil {
		return nil, err
	}
	var urls []string
	for k := range pixiv.Key {
		url := pixiv.Key[k].Urls.Original
		urls = append(urls, url)
	}
	return urls, nil
}
