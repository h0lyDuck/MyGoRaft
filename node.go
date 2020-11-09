package MyGoRaft

import (
	"sync"
	"time"
)

//声明raft节点类型
type NodeInfo struct {
	id   string
	port string
}

type Node struct {
	info              *NodeInfo
	vote              int
	lock              sync.Mutex
	id                string
	term              int
	votedFor          string
	leader            string
	role              int
	lastMessageTime   int64
	nextHeartBeatTime int64
	heartBeatTimeout  int
	voteCh            chan bool
	heartBeat         chan bool
}

func initNode(id, port string) *Node {
	n := new(Node)
	n.info.id = id
	n.info.port = port
	n.setVote(0)
	n.setVotedFor("-1")
	n.setRole(0)
	n.setLeader("-1")
	n.setTerm(0)
	n.lastMessageTime = 0
	n.nextHeartBeatTime = 0
	n.heartBeatTimeout = heartBeatTimeout
	return n
}

func (n *Node) becomeCandidate() bool {
	sleepTime := randRange(1500, 5000)
	time.Sleep(time.Duration(sleepTime) * time.Microsecond)
	if n.role == 0 && n.votedFor != "-1" && n.leader != "-1" {

		return true
	}
	return false
}

func (n *Node) setVotedFor(new string) {
	n.lock.Lock()
	n.votedFor = new
	n.lock.Unlock()
}

func (n *Node) setTerm(new int) {
	n.lock.Lock()
	n.term = new
	n.lock.Unlock()
}

func (n *Node) setRole(new int) {
	n.lock.Lock()
	n.role = new
	n.lock.Unlock()
}

func (n *Node) setLeader(new string) {
	n.lock.Lock()
	n.leader = new
	n.lock.Unlock()
}

func (n *Node) encVote() {
	n.lock.Lock()
	n.vote++
	n.lock.Unlock()
}

func (n *Node) setVote(new int) {
	n.lock.Lock()
	n.vote = new
	n.lock.Unlock()
}

func (n *Node) reDefault() {
	n.setVote(0)
	n.setLeader("-1")
	n.setVotedFor("-1")
	n.setRole(0)
}
