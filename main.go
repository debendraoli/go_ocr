package main

import (
	"github.com/rs/cors"
	"go_ocr/handlers"
	"go_ocr/helpers/ghostscript"
	"log"
	"net/http"
)

func init() {
	_, err := ghostscript.GetRevision()
	if err != nil {
		panic("error")
	}
}

func main() {
	h := handlers.NewHandler()
	router := http.NewServeMux()
	router.HandleFunc("/ocr", h.UploadFile)
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedHeaders: []string{"Origin", "Accept", "Content-Type"},
		AllowedMethods: []string{"OPTIONS", "GET", "POST"},
	})

	err := http.ListenAndServe(":8080", c.Handler(router))
	if err != nil {
		log.Fatal("Error attempting to ListenAndServe: ", err)
	}
}
