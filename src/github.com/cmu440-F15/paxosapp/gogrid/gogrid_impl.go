package main

import (
	"fmt"
	"github.com/cmu440-F15/paxosapp/paxos"
	"github.com/cmu440-F15/paxosapp/rpc/paxosrpc"
	// "net"
	"encoding/json"
	"flag"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type gridData struct {
	Rows int
	Cols int
	Data [][]string
}

var (
	staticDir  = flag.String("static", "", "directory to static files")
	serverPort = flag.Uint("port", 10086, "port to to access the web app")
	// paxoPort   = flag.Uint("paxoport", 10087, "port to communicate with other nodes")
	paxoNodes = flag.String("paxonodes", "peterxia.com:10087,peterxia.com:10089", "hostports (containing self) seperated by commas")
	srvId     = flag.Int("srvid", -1, "srvId")
	paxosNode paxos.PaxosNode
	grid      = gridData{
		Rows: 3,
		Cols: 3,
		Data: [][]string{
			[]string{"a", "b", "c"},
			[]string{"d", "e", "f"},
			[]string{"f", "g", "h"},
		},
	}
	gridLock = &sync.RWMutex{}
)

func handler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(&grid)
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, r.URL.Path)
	// http.Handle("/", http.FileServer(http.Dir("/")))
}

type cellChangeReq struct {
	Row   int
	Col   int
	Value string
}

func cellchangeHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		return
	}
	r.ParseForm()
	// fmt.Println(r.Form)
	// fmt.Println(r.PostForm)
}

func doUpdate() {
	c := time.NewTicker(time.Duration(5) * time.Second).C
	for {
		for r := 0; r < grid.Rows; r++ {
			for c := 0; c < grid.Cols; c++ {
				args := paxosrpc.GetValueArgs{
					Key: fmt.Sprintf("%d,%d", r, c),
				}
				reply := paxosrpc.GetValueReply{}
				if err := paxosNode.GetValue(&args, &reply); err != nil {
					continue
				}

				if reply.V != nil {
					gridLock.Lock()
					grid.Data[r][c] = reply.V.(string)
					gridLock.Unlock()
				}
			}
		}
		<-c
	}
}

func main() {
	flag.Parse()
	if *staticDir == "" {
		fmt.Println("Please provide a path to static files.")
		return
	}

	if *srvId < 0 {
		fmt.Println("Please provide a srvid.")
		return
	}

	hostportsSlice := strings.Split(*paxoNodes, ",")
	hostports := make(map[int]string)
	for i, hp := range hostportsSlice {
		hostports[i] = hp
	}

	// paxosnode, err := paxos.NewPaxosNode(hostports[*srvId], hostports, len(hostports), *srvId, 60, false)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// paxosNode = paxosnode

	http.HandleFunc("/api/data", handler)
	http.HandleFunc("/api/cellchange", cellchangeHandle)
	http.HandleFunc("/api/test", testHandler)

	fs := http.FileServer(http.Dir(*staticDir))
	http.Handle("/", fs)

	fmt.Println("Starting server on port " + strconv.FormatUint(uint64(*serverPort), 10))
	http.ListenAndServe(":"+strconv.FormatUint(uint64(*serverPort), 10), nil)
}
