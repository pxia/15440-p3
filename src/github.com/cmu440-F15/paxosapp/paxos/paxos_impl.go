package paxos

import (
	"errors"
	"github.com/cmu440-F15/paxosapp/rpc/paxosrpc"
	"net"
	"net/http"
	"net/rpc"
	"sync"
	"time"
)

type paxosInstance struct {
	lock *sync.Mutex
	n_a  int
	n_h  int
	v_a  interface{}
	V    chan interface{}
}

func newPaxosInstance() *paxosInstance {
	return &paxosInstance{
		lock: &sync.Mutex{},
		n_a:  0,
		n_h:  0,
		v_a:  nil,
		V:    make(chan interface{}, 100),
	}
}

type paxosNode struct {
	rpcMap           map[string]*rpc.Client
	proposalNums     map[string]int
	proposalNumsLock *sync.Mutex

	proposals     map[string]*paxosInstance
	proposalsLock *sync.Mutex

	commits     map[string]interface{}
	commitsLock *sync.Mutex

	timeout  time.Duration
	numNodes int
	srvId    int
}

type rpcPair struct {
	hostport string
	client   *rpc.Client
}

func tryDial(hostport string, numRetries int, res chan rpcPair) {
	for i := 0; i < numRetries; i++ {
		if cli, err := rpc.DialHTTP("tcp", hostport); err == nil {
			res <- rpcPair{
				hostport: hostport,
				client:   cli,
			}
			return
		}
		time.Sleep(time.Second)
	}
	res <- rpcPair{
		hostport: hostport,
		client:   nil,
	}
	return
}

// NewPaxosNode creates a new PaxosNode. This function should return only when
// all nodes have joined the ring, and should return a non-nil error if the node
// could not be started in spite of dialing the other nodes numRetries times.
//
// hostMap is a map from node IDs to their hostports, numNodes is the number
// of nodes in the ring, replace is a flag which indicates whether this node
// is a replacement for a node which failed.
func NewPaxosNode(myHostPort string, hostMap map[int]string, numNodes, srvId, numRetries int, replace bool) (PaxosNode, error) {
	pn := &paxosNode{
		proposalNums:     make(map[string]int),
		proposalNumsLock: &sync.Mutex{},
		proposals:        make(map[string]*paxosInstance),
		proposalsLock:    &sync.Mutex{},
		commits:          make(map[string]interface{}),
		commitsLock:      &sync.Mutex{},
		timeout:          time.Second * time.Duration(15),
		numNodes:         numNodes,
		srvId:            srvId,
	}

	listener, err := net.Listen("tcp", myHostPort)
	if err != nil {
		return nil, err
	}

	// Wrap the tribServer before registering it for RPC.
	err = rpc.RegisterName("PaxosNode", paxosrpc.Wrap(pn))
	if err != nil {
		return nil, err
	}

	rpc.HandleHTTP()
	go http.Serve(listener, nil)

	rpcClients := make(chan rpcPair, numNodes)
	for _, hostport := range hostMap {
		go tryDial(hostport, numRetries, rpcClients)
	}

	rpcMap := make(map[string]*rpc.Client)
	for i := 0; i < numNodes; i++ {
		c := <-rpcClients
		if c.client == nil {
			return nil, errors.New("Cannot connect to all nodes")
		}
		rpcMap[c.hostport] = c.client
	}

	pn.rpcMap = rpcMap

	return pn, nil
}

func (pn *paxosNode) GetNextProposalNumber(args *paxosrpc.ProposalNumberArgs, reply *paxosrpc.ProposalNumberReply) error {
	pn.proposalNumsLock.Lock()
	defer pn.proposalNumsLock.Unlock()

	if _, ok := pn.proposalNums[args.Key]; !ok {
		pn.proposalNums[args.Key] = 0
	}
	reply.N = (pn.proposalNums[args.Key]/pn.numNodes+1)*pn.numNodes + pn.srvId
	return nil
}

func (pn *paxosNode) Propose(args *paxosrpc.ProposeArgs, reply *paxosrpc.ProposeReply) error {
	preplies := make(chan *paxosrpc.PrepareReply, pn.numNodes)
	pargs := paxosrpc.PrepareArgs{
		Key: args.Key,
		N:   args.N,
	}

	oks := 0
	max_n := 0
	var V interface{}
	for _, cli := range pn.rpcMap {
		// rpclock := &sync.Mutex{}
		go func(c *rpc.Client) {
			var preply paxosrpc.PrepareReply
			// rpclock.Lock()
			err := c.Call("PaxosNode.RecvPrepare", &pargs, &preply)
			// rpclock.Unlock()
			if err != nil {
				preplies <- nil
			} else {
				preplies <- &preply
			}
		}(cli)
	}

	timeout := time.After(pn.timeout)
	for i := 0; i < pn.numNodes; i++ {
		select {
		case preply := <-preplies:
			if preply == nil || preply.Status != paxosrpc.OK {
				continue
			}
			oks += 1
			if preply.N_a > max_n {
				max_n = preply.N_a
				V = preply.V_a
			}
		case <-timeout:
			return errors.New("prepare timeout")
		}
	}

	paxos, _ := pn.proposals[args.Key]

	if oks < pn.numNodes/2+1 {
		// rip
		// wait for commit from others
		*reply = paxosrpc.ProposeReply{
			V: <-paxos.V,
		}
		return nil
	}

	// accept
	if max_n == 0 {
		V = args.V
	}
	areplies := make(chan *paxosrpc.AcceptReply, pn.numNodes)
	aargs := paxosrpc.AcceptArgs{
		Key: args.Key,
		N:   args.N,
		V:   V,
	}
	oks = 0
	for _, cli := range pn.rpcMap {
		go func(c *rpc.Client) {
			var areply paxosrpc.AcceptReply
			err := c.Call("PaxosNode.RecvAccept", &aargs, &areply)
			if err != nil {
				// node fail, rip
				areplies <- nil
			} else {
				areplies <- &areply
			}
		}(cli)
	}

	timeout = time.After(pn.timeout)
	for i := 0; i < pn.numNodes; i++ {
		select {
		case areply := <-areplies:
			if areply == nil || areply.Status != paxosrpc.OK {
				continue
			}
			oks += 1
		case <-timeout:
			return errors.New("accept timeout")
		}
	}

	if oks < pn.numNodes/2+1 {
		// wait for commit..
		*reply = paxosrpc.ProposeReply{
			V: <-paxos.V,
		}
		return nil
	}

	// commit
	cargs := paxosrpc.CommitArgs{
		Key: args.Key,
		V:   V,
	}

	finish := make(chan bool, pn.numNodes)
	for _, cli := range pn.rpcMap {
		go func(c *rpc.Client) {
			var creply paxosrpc.CommitReply
			c.Call("PaxosNode.RecvCommit", &cargs, &creply)
			finish <- false
		}(cli)
	}

	timeout = time.After(pn.timeout)
	for i := 0; i < pn.numNodes; i++ {
		select {
		case <-finish:
		case <-timeout:
			return errors.New("commit timeout")
		}
	}

	*reply = paxosrpc.ProposeReply{
		V: V,
	}
	return nil
}

func (pn *paxosNode) GetValue(args *paxosrpc.GetValueArgs, reply *paxosrpc.GetValueReply) error {
	pn.commitsLock.Lock()
	defer pn.commitsLock.Unlock()

	if val, ok := pn.commits[args.Key]; !ok {
		*reply = paxosrpc.GetValueReply{
			Status: paxosrpc.KeyNotFound,
		}
	} else {
		*reply = paxosrpc.GetValueReply{
			Status: paxosrpc.KeyFound,
			V:      val,
		}
	}

	return nil
}

func (pn *paxosNode) RecvPrepare(args *paxosrpc.PrepareArgs, reply *paxosrpc.PrepareReply) error {
	pn.proposalsLock.Lock()
	defer pn.proposalsLock.Unlock()
	pn.proposalNumsLock.Lock()
	defer pn.proposalNumsLock.Unlock()

	if _, ok := pn.proposals[args.Key]; !ok {
		pn.proposals[args.Key] = newPaxosInstance()
	}
	paxos, _ := pn.proposals[args.Key]

	if paxos.n_h > 0 && paxos.n_h > args.N {
		// prepare reject
		*reply = paxosrpc.PrepareReply{
			Status: paxosrpc.Reject,
		}
	} else {
		// prepare accept
		pn.proposalNums[args.Key] = args.N
		paxos.n_h = args.N
		*reply = paxosrpc.PrepareReply{
			Status: paxosrpc.OK,
			N_a:    paxos.n_a,
			V_a:    paxos.v_a,
		}
	}

	return nil
}

func (pn *paxosNode) RecvAccept(args *paxosrpc.AcceptArgs, reply *paxosrpc.AcceptReply) error {
	pn.proposalsLock.Lock()
	defer pn.proposalsLock.Unlock()
	pn.proposalNumsLock.Lock()
	defer pn.proposalNumsLock.Unlock()

	if _, ok := pn.proposals[args.Key]; !ok {
		pn.proposals[args.Key] = newPaxosInstance()
	}
	paxos, _ := pn.proposals[args.Key]

	if paxos.n_h > 0 && paxos.n_h > args.N {
		// accept reject
		*reply = paxosrpc.AcceptReply{
			Status: paxosrpc.Reject,
		}
	} else {
		// accept accept
		paxos.n_h = args.N
		paxos.v_a = args.V
		paxos.n_a = args.N
		pn.proposalNums[args.Key] = args.N

		*reply = paxosrpc.AcceptReply{
			Status: paxosrpc.OK,
		}
	}

	return nil
}

func (pn *paxosNode) RecvCommit(args *paxosrpc.CommitArgs, reply *paxosrpc.CommitReply) error {
	pn.commitsLock.Lock()
	pn.commits[args.Key] = args.V
	pn.commitsLock.Unlock()

	pn.proposalsLock.Lock()
	// paxos.V <- args.V
	delete(pn.proposals, args.Key)
	pn.proposalsLock.Unlock()
	return nil
}

func (pn *paxosNode) RecvReplaceServer(args *paxosrpc.ReplaceServerArgs, reply *paxosrpc.ReplaceServerReply) error {
	return errors.New("not implemented")
}

func (pn *paxosNode) RecvReplaceCatchup(args *paxosrpc.ReplaceCatchupArgs, reply *paxosrpc.ReplaceCatchupReply) error {
	return errors.New("not implemented")
}
