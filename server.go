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

func main() {
	var dir string

	if len(os.Args) > 1 {
		dir = os.Args[1]
	} else {
		dir, _ = os.Getwd()
	}
	liveServer := http.FileServer(http.Dir(dir))
	log.Println("Listening on port 8080")

	go func() { log.Fatal(http.ListenAndServe(":8080", liveServer)) }()

	doneChan := make(chan bool)

	go func(doneChan chan bool) {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Fatal(err)
			}

			go watchFile(path, doneChan)

			return nil
		})

		if err != nil {
			log.Fatal(err)
		}

		// <-changed
	}(doneChan)

	<-doneChan
	log.Println("Restarting due to changes")
}
