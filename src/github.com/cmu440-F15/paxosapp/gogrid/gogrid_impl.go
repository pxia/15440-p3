package main

import (
	"fmt"
	// "github.com/cmu440-F15/paxosapp/paxos"
	// "net"
	"encoding/json"
	"flag"
	"net/http"
	"strconv"
)

var (
	staticDir  = flag.String("static", "", "directory to static files")
	serverPort = flag.Uint("port", 10086, "port to to access the web app")
	paxoPort   = flag.Uint("paxoport", 10087, "port to communicate with other nodes")
)

type gridData struct {
	Rows int
	Cols int
	Data [][]string
}

func handler(w http.ResponseWriter, r *http.Request) {
	// fmt.Fprintf(w, r.URL.Path)
	grid := gridData{
		Rows: 3,
		Cols: 3,
		Data: [][]string{
			[]string{"a", "b", "c"},
			[]string{"d", "e", "f"},
			[]string{"f", "g", "h"},
		},
	}
	json.NewEncoder(w).Encode(&grid)
}

func main() {
	flag.Parse()
	if *staticDir == "" {
		fmt.Println("Please provide a path to static files.")
		return
	}

	http.HandleFunc("/api/data", handler)

	fs := http.FileServer(http.Dir(*staticDir))
	http.Handle("/", fs)
	// http.HandleFunc("/", handler)
	http.ListenAndServe(":"+strconv.FormatUint(uint64(*serverPort), 10), nil)
}
