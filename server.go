// dotfiles are ignored, but it will still restart when a new dotfile is created or removed
package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// creates a simple server that ignores dotfiles
func createServer() *http.Server {
	return &http.Server{
		Addr:    ":8080",
		Handler: http.FileServer(dotFileHidingFileSystem{http.Dir(os.Getenv("DIR"))}),
	}
}

// containsDotFile reports whether name contains a path element starting with a period.
// The name is assumed to be a delimited by forward slashes, as guaranteed
// by the http.FileSystem interface.
func containsDotFile(name string) bool {
	parts := strings.Split(name, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

// dotFileHidingFile is the http.File use in dotFileHidingFileSystem.
// It is used to wrap the Readdir method of http.File so that we can
// remove files and directories that start with a period from its output.
type dotFileHidingFile struct {
	http.File
}

// Readdir is a wrapper around the Readdir method of the embedded File
// that filters out all files that start with a period in their name.
func (f dotFileHidingFile) Readdir(n int) (fis []fs.FileInfo, err error) {
	files, err := f.File.Readdir(n)
	for _, file := range files { // Filters out the dot files
		if !strings.HasPrefix(file.Name(), ".") {
			fis = append(fis, file)
		}
	}
	return
}

// dotFileHidingFileSystem is an http.FileSystem that hides
// hidden "dot files" from being served.
type dotFileHidingFileSystem struct {
	http.FileSystem
}

// Open is a wrapper around the Open method of the embedded FileSystem
// that serves a 403 permission error when name has a file or directory
// with whose name starts with a period in its path.
func (fsys dotFileHidingFileSystem) Open(name string) (http.File, error) {
	if containsDotFile(name) { // If dot file, return 403 response
		return nil, fs.ErrPermission
	}

	file, err := fsys.FileSystem.Open(name)
	if err != nil {
		return nil, err
	}
	return dotFileHidingFile{file}, err
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
func startServer() {
	srv := createServer()
	log.Printf("Listening on port %v\n", srv.Addr)

	// for restarting server when there are changes to files
	restart := make(chan struct{})
	go func() {
		doneChan := make(chan bool)

		go addWatchers(doneChan)
		<-doneChan

		killServer(srv)
		close(restart)
	}()

	// for graceful shutdown upon Sigint
	shutdown := make(chan struct{})
	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		<-interrupt

		killServer(srv)
		close(shutdown)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Error starting or closing server: %v\n", err)
	}

	// either restart or shutdown server depending on what channel is closed
	select {
	case <-restart:
		log.Printf("Server restarted")
		startServer()
	case <-shutdown:
		log.Printf("Server stopped")
	}
}

func killServer(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error stopping server: %v\n", err)
	}
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
