package main

import (
	"fmt"
	// "github.com/cmu440-F15/paxosapp/paxos"
	// "net"
	"flag"
	"net/http"
)

var (
	staticDir = flag.String("static", "", "directory to static files")
)

// func handler(w http.ResponseWriter, r *http.Request) {
// fmt.Fprintf(w, "%s\n", r.URL.Path)
// }

func main() {
	flag.Parse()
	if *staticDir == "" {
		fmt.Println("Please provide a path to static files.")
		return
	}

	fs := http.FileServer(http.Dir(*staticDir))
	http.Handle("/", fs)
	// http.HandleFunc("/", handler)
	http.ListenAndServe(":10086", nil)
}
