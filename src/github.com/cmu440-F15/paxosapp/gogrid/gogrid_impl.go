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
	paxoNodes = flag.String("paxonodes", "peterxia.com:10087,peterxia.com:10089,ytchen.com:10087", "hostports (containing self) seperated by commas")
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
	go doChange(r.Form)
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

func doChange(form map[string][]string) {
	change := cellChangeReq{}

	if row, err := strconv.Atoi(form["Row"][0]); err != nil {
		return
	} else {
		change.Row = row
	}

	if col, err := strconv.Atoi(form["Col"][0]); err != nil {
		return
	} else {
		change.Col = col
	}

	change.Value = form["Value"][0]
	gridLock.Lock()
	grid.Data[change.Row][change.Col] = change.Value
	gridLock.Unlock()

	nargs := paxosrpc.ProposalNumberArgs{
		Key: fmt.Sprintf("%d,%d", change.Row, change.Col),
	}
	nreply := paxosrpc.ProposalNumberReply{}
	if err := paxosNode.GetNextProposalNumber(&nargs, &nreply); err != nil {
		return
	}

	pargs := paxosrpc.ProposeArgs{
		N:   nreply.N,
		Key: nargs.Key,
		V:   change.Value,
	}
	preply := paxosrpc.ProposeReply{}
	if err := paxosNode.Propose(&pargs, &preply); err != nil {
		return
	}

	gridLock.Lock()
	grid.Data[change.Row][change.Col] = preply.V.(string)
	gridLock.Unlock()
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

	paxosnode, err := paxos.NewPaxosNode(hostports[*srvId], hostports, len(hostports), *srvId, 60, false)
	if err != nil {
		fmt.Println(err)
		return
	}
	paxosNode = paxosnode

	http.HandleFunc("/api/data", handler)
	http.HandleFunc("/api/cellchange", cellchangeHandle)
	http.HandleFunc("/api/test", testHandler)
	go doUpdate()

	fs := http.FileServer(http.Dir(*staticDir))
	http.Handle("/", fs)

	fmt.Println("Starting server on port " + strconv.FormatUint(uint64(*serverPort), 10))
	http.ListenAndServe(":"+strconv.FormatUint(uint64(*serverPort), 10), nil)
}
