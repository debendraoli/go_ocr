package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/otiai10/gosseract"
	"go_ocr/helpers"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type FileReq struct {
	JobTag       string `json:"job_tag,omitempty"`
	Content      string `json:"content" binding:"required"`
	Languages     string `json:"languages,omitempty"`
	Replacements string `json:"replacements,omitempty"`
	WhiteList string `json:"whitelist_chars,omitempty"`
}

type docPages struct {
	PageNumber  uint16 `json:"page_number" binding:"required"`
	Length     uint16 `json:"length" binding:"required"`
	Content    string `json:"content" binding:"required"`
}

type Rendered struct {
	JobTag     string `json:"job_tag,omitempty"`
	Pages      []docPages
}


type handlers struct {
	throttle  chan int
	watcher    *fsnotify.Watcher
}

func NewHandler() *handlers {
	return &handlers{
		throttle:  make(chan int, 60),
	}
}

func (h *handlers) UploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var filesRequested []FileReq

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err = json.Unmarshal(body, &filesRequested); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if len(filesRequested) == 0 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	filesInfo := make(map[string]FileReq)
	tmpFiles := make(map[*os.File]string)

	for _, file := range filesRequested {
		b, err := base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		tmpDir, err := ioutil.TempDir(os.TempDir(), "conv")
		tmpFile, err := ioutil.TempFile(tmpDir, "file")
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if _, err = tmpFile.Write(b); err != nil {
			fmt.Println(err)
		}
		filesInfo[tmpDir] = file
		tmpFiles[tmpFile] = http.DetectContentType(b)
	}

	convertedFileChan := make(chan map[string][]string)
	var totalFiles uint16
	var totalProcessedFiles uint16

	for file, contentType := range tmpFiles {
		fmt.Println(contentType)
		go helpers.ConvertToPNG(file, convertedFileChan)
	}
	extractedTextsChan := make(chan map[string]docPages)
	var extractedFiles []Rendered
	extractedPage := make(map[string][]docPages)
	for {
		select {
		case changedFile := <-convertedFileChan:
			for pdfFile, files := range changedFile {
				totalFiles += uint16(len(files))
				for _, file := range files {
					go textExtractor(extractedTextsChan, file, filesInfo[pdfFile])
				}
			}
		case extraction := <-extractedTextsChan:
			for key, page := range extraction {
				extractedPage[key] = append(extractedPage[key], page)
			}
			totalProcessedFiles++
			if totalFiles == totalProcessedFiles {
				for fileID, pages := range extractedPage {
					_ = os.RemoveAll(fileID)
					extractedFiles = append(extractedFiles, Rendered{JobTag: filesInfo[fileID].JobTag, Pages: pages})
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(extractedFiles); err != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
				return
			}
		}
	}
}

func textExtractor(extraction chan map[string]docPages, changedFile string, fileDetails FileReq) {
	client := gosseract.NewClient()
	defer client.Close()
	pageNumber, _ := strconv.Atoi(strings.TrimRight(changedFile[strings.LastIndex(changedFile, "/"):], ".png")[1:])
	pageProp := make(map[string]docPages)
	page := docPages{PageNumber: uint16(pageNumber)}
	tag := changedFile[:strings.LastIndex(changedFile, "/")]
	pageProp[tag] = page
	if err := client.SetImage(changedFile); err != nil {
		page.Length = 0
		page.Content = "Error processing"
		return
	}
	if fileDetails.Languages != "" {
		client.Languages = strings.Split(fileDetails.Languages, ",")
	}
	if fileDetails.WhiteList != "" {
		_ = client.SetWhitelist(fileDetails.WhiteList)
	}
	text, err := client.Text()
	if err != nil {
		page.Length = 0
		page.Content = "Error processing."
		return
	}
	page.Content = strings.Trim(text, fileDetails.Replacements)
	page.Length = uint16(len(text))
	extraction <- pageProp
	_ = os.Remove(changedFile)
	return
}
