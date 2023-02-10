// dotfiles are ignored, but it will still restart when a new dotfile is created or removed
// should probably also add graceful shutdown
// need to set up FileServer to ignore dotfiles. This is in the docs
package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// creates a simple server
// would like to add functionality to ignore .dotfiles
func createServer() *http.Server {
	return &http.Server{
		Addr:    ":8080",
		Handler: http.FileServer(http.Dir(os.Getenv("DIR"))),
	}
}

// watches an individual file for changes.
// gets initial file Stat, and then compares that with current file Stat after 1 second
// if the file changes, changed channel will fill
// which will unblock call to Shutdown() in startServer()
func watchFile(filePath string, changed chan bool) {
	initialStat, err := os.Stat(filePath)
	if err != nil {
		log.Fatalf("Error with initial file stat: %v\n", err)
	}

	for {
		stat, err := os.Stat(filePath)
		if err != nil {
			log.Printf("Error with file stat: %v\n", err)
			break
		}

		if initialStat.ModTime() != stat.ModTime() && initialStat.Size() != stat.Size() {
			break
		}
		time.Sleep(1 * time.Second)
	}
	changed <- true
}

// adds watchers for reach file in the directory
// c is used to pass the doneChan from startServer() to each fileWatcher
func addWatchers(c chan bool) {
	dir := os.Getenv("DIR")
	err := filepath.WalkDir(dir, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		if isDotFile(path.Base(name)) {
			return nil
		}

		go watchFile(name, c)

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Watching directory: %v\n", dir)
}

// checks if file is a dotfile. Expects to be passed the base name (path.Base(filename))
func isDotFile(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	return false
}

// creates a new server and calls ListenAndServe()
// sets up watchers for the directory
// calls Shutdown() on server if files change
// recursive call to startServer() to restart the server
// need to add graceful shutdown
func startServer() {
	srv := createServer()
	log.Printf("Listening on port %v\n", srv.Addr)

	idleConnsClosed := make(chan struct{})
	go func() {
		doneChan := make(chan bool)
		go addWatchers(doneChan)
		<-doneChan

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Error restarting server: %v\n", err)
		}
		log.Printf("Server restarted")
		close(idleConnsClosed)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Error starting or closing server: %v\n", err)
	}
	<-idleConnsClosed

	startServer()
}

func main() {
	var dir string
	if len(os.Args) > 1 {
		dir = os.Args[1]
	} else {
		dir, _ = os.Getwd()
	}

	// add directory to watch to ENV to be accessed easily by other functions
	if err := os.Setenv("DIR", dir); err != nil {
		log.Fatal(err)
	}

	startServer()
}
