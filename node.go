package MyGoRaft

import (
	"MyGoRaft/NodeRPC"
	"context"
	"google.golang.org/grpc"
	"log"
	"sync"
	"time"
)

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
	voteResult        chan bool
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
	n.voteResult = make(chan bool)
	return n
}

func (n *Node) becomeCandidate() bool {
	sleepTime := randRange(1500, 5000)
	time.Sleep(time.Duration(sleepTime) * time.Microsecond)
	if n.role == 0 && n.votedFor != "-1" && n.leader != "-1" {
		n.setRole(1)
		n.setVotedFor("-1")
		n.setTerm(n.term + 1)
		n.setLeader("-1")
		n.encVote()
		log.Println("超时后无人参选，本节点" + n.id + "变成候选人")
		return true
	}
	return false
}

func (n *Node) becomeLeader() bool {
	log.Println("本节点开始参加选举")
	go func() {
		for key, port := range nodeList {
			if key != n.id {
				conn, err := grpc.Dial("127.0.0.1"+port, grpc.WithInsecure())
				if err != nil {
					log.Panic("连接出错啦")
				}
				c := NodeRPC.NewRaftNodeClient(conn)
				req := NodeRPC.IDRequest{
					ID: n.id,
				}
				ret, err := c.Vote(context.Background(), &req)
				if err != nil {
					log.Panic("远程调用出错啦")
				}
				if ret.Result {
					n.encVote()
				}
				conn.Close()
			}
			if n.vote > nodeNum/2 && n.leader == "-1" {
				log.Println("获得超半数的选票,变成leader")
				n.setRole(2)
				n.setLeader(n.id)
				for key, port := range nodeList {
					if key != n.id {
						conn, err := grpc.Dial("127.0.0.1"+port, grpc.WithInsecure())
						if err != nil {
							log.Panic("连接出错啦")
						}
						c := NodeRPC.NewRaftNodeClient(conn)
						req := NodeRPC.IDRequest{
							ID: n.id,
						}
						ret, err := c.ConfirmLeader(context.Background(), &req)
						if err != nil {
							log.Panic("远程调用出错啦")
						}
						if ret.Result {
							log.Println("节点" + key + "已确认本节点为领导者")
						}
						conn.Close()
					}
				}
				n.voteResult <- true
				return
			}
		}
	}()
	select {
	case <-time.After(time.Duration(heartBeatTimeout) * time.Second):
		log.Println("领导人选举超时，本节点变为Follower状态")
		n.reDefault()
		return false
	case success := <-n.voteResult:
		if success {
			return true
		}
		return false
	}
}

//
// 节点gRPC part
//
func (n *Node) Vote(ctx context.Context, req *NodeRPC.IDRequest) (*NodeRPC.BoolResponse, error) {

}

//
// 节点修改自身属性 part
//
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
