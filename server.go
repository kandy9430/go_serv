package main

import (
	"log"
	"os"
	"os/exec"
	"time"
)

func main() {
	// var dir string
	pid := os.Getpid()
	ppid := os.Getppid()
	name := os.Args[0]
	args := os.Args[1:]

	log.Printf("In command: %v, parent: %v\n", pid, ppid)
	// if len(args) > 0 {
	// 	dir = args[0]
	// } else {
	// 	dir, _ = os.Getwd()
	// }

	time.Sleep(5 * time.Second)

	cmd := &exec.Cmd{
		Path: name,
		Args: args,
		Env:  []string{"parent=go_serv"},
	}

	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	if parent := os.Getenv("parent"); parent == "go_serv" {
		proc, err := os.FindProcess(ppid)
		if err != nil {
			log.Fatal(err)
		}
		err = proc.Kill()
		if err != nil {
			log.Fatal(err)
		}
	}
	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}
	// liveServer := http.FileServer(http.Dir(dir))
	// log.Println("Listening on port 8080")
	// log.Fatal(http.ListenAndServe(":8080", liveServer))
}
