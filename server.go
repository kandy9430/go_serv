package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	var dir string
	if len(os.Args) > 1 {
		dir = os.Args[1]
	} else {
		dir, _ = os.Getwd()
	}
	log.Println(dir)
	server := &http.Server{
		Addr:    ":8080",
		Handler: http.FileServer(http.Dir(dir)),
	}

	log.Printf("listening on port %v\n", server.Addr)
	log.Fatal(server.ListenAndServe())
}
