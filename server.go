// failing when a file is removed from directory
// stops the server, and then fails when trying to restart
// 	probably something to do with the next call to startServer()
package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func watchFile(filePath string, changed chan bool) {
	log.Printf("watching: %s\n", filePath)
	initialStat, err := os.Stat(filePath)
	if err != nil {
		log.Fatal(err)
	}

	for {
		stat, err := os.Stat(filePath)
		if err != nil {
			log.Fatal(err)
		}

		if initialStat.ModTime() != stat.ModTime() && initialStat.Size() != stat.Size() {
			break
		}
		time.Sleep(1 * time.Second)
	}
	changed <- true
}

func addWatchers(dir string, c chan bool) {
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		go watchFile(path, c)

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
}

func startServer(dir string) {
	server := &http.Server{
		Addr:    ":8080",
		Handler: http.FileServer(http.Dir(dir)),
	}

	log.Println("Listening on port 8080")

	doneChan := make(chan bool)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	go addWatchers(dir, doneChan)
	<-doneChan

	log.Println("Restarting due to changes")
}

func main() {
	var dir string

	if len(os.Args) > 1 {
		dir = os.Args[1]
	} else {
		dir, _ = os.Getwd()
	}

	startServer(dir)
}
