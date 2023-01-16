package main

import (
	"fmt"
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
	liveServer := http.FileServer(http.Dir(dir))
	fmt.Println("Listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", liveServer))
}
