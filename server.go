// failing when a file is removed from directory
// stops the server, and then fails when trying to restart
// 	probably something to do with the next call to startServer()
package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func createServer() *http.Server {
	return &http.Server{
		Addr:    ":8080",
		Handler: http.FileServer(http.Dir(os.Getenv("DIR"))),
	}
}

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

// func startServer(dir string) {
// 	srv := createServer()
//
// 	log.Println("Listening on port 8080")
//
// 	doneChan := make(chan bool)
//
// 	go func() {
// 		if err := server.ListenAndServe(); err != nil {
// 			log.Fatal(err)
// 		}
// 	}()
//
// 	go addWatchers(dir, doneChan)
// 	<-doneChan
//
// 	log.Println("Restarting due to changes")
// }

func startServer() {
	srv := createServer()
	log.Printf("Listening on port %v\n", srv.Addr)

	idleConnsClosed := make(chan struct{})
	go func() {
		// stop := make(chan os.Signal, 1)
		// signal.Notify(stop, os.Interrupt)

		// <-stop

		doneChan := make(chan bool)
		go addWatchers(dir, doneChan)
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

// func watchForInterrupt(srv *http.Server) {
// 	stop := make(chan os.Signal, 1)
// 	signal.Notify(stop, os.Interrupt)
//
// 	<-stop
//
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()
//
// 	if err := srv.Shutdown(ctx); err != nil {
// 		log.Fatal(err)
// 	}
// 	log.Println("Server restarted")
// 	// true ->c
// }

func main() {
	var dir string
	if len(os.Args) > 1 {
		dir = os.Args[1]
	} else {
		dir, _ = os.Getwd()
	}

	if err := os.Setenv("DIR", dir); err != nil {
		log.Fatal(err)
	}

	startServer()
}
