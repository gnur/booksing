package main

import (
	"os/exec"
	"strings"
)

type convertRequest struct {
	GetURL   string
	Filename string
	PutURL   string
}

func main() {
	for {
		// get this from somewhere
		req := convertRequest{}
		convertBook(req)
	}
}

func convertBook(req convertRequest) error {

	//TODO: download book from getURL

	mobiPath := strings.Replace(req.Filename, ".epub", ".mobi", 1)
	cmd := exec.Command("ebook-convert", req.Filename, mobiPath)

	_, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	//TODO: upload it to putURL
	//TODO: let booksing know it is converted now
	return nil
}
