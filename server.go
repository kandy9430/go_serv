package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func createServer() *http.Server {
	return &http.Server{
		Addr:    ":8080",
		Handler: http.FileServer(http.Dir(os.Getenv("DIR"))),
	}
}

func startServer() {
	srv := createServer()
	log.Printf("Listening on port %v\n", srv.Addr)

	idleConnsClosed := make(chan struct{})
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt)

		<-stop

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
