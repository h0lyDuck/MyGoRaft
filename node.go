package main

import (
	"MyGoRaft/NodeRPC"
	"context"
	"google.golang.org/grpc"
	"log"
	"strconv"
	"sync"
	"time"
)

type NodeInfo struct {
	id   string
	port string
}

type Node struct {
	info              *NodeInfo
	id                string
	vote              int
	lock              sync.Mutex
	term              int
	votedFor          string
	leader            string
	role              int
	lastMessageTime   int64
	nextHeartBeatTime int64
	heartBeatTimeout  int
	voteResult        chan bool
	heartBeatStart    chan bool
}

type Message struct {
	Msg   string
	MsgID string
}

func initNode(id, port string) *Node {
	info := &NodeInfo{
		id:   id,
		port: port,
	}
	n := new(Node)
	n.info = info
	n.id = id
	n.setVote(0)
	n.setVotedFor("-1")
	n.setRole(0)
	n.setLeader("-1")
	n.setTerm(0)
	n.lastMessageTime = 0
	n.nextHeartBeatTime = 0
	n.heartBeatTimeout = heartBeatTimeout
	n.voteResult = make(chan bool)
	n.heartBeatStart = make(chan bool)
	return n
}

func (n *Node) election() {
	for {
		if n.becomeCandidate() {
			if n.becomeLeader() {
				return
			} else {
				continue
			}
		} else {
			return
		}
	}
}

func (n *Node) becomeCandidate() bool {
	sleepTime := randRange(1500, 5000)
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	if n.role == 0 && n.votedFor == "-1" && n.leader == "-1" {
		n.setRole(1)
		n.setVotedFor(n.id)
		n.setTerm(n.term + 1)
		n.setLeader("-1")
		n.encVote()
		log.Println("超时后无人参选，本节点" + n.id + "变成候选人,当前得票数：" + strconv.Itoa(n.vote))
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
					log.Println("节点" + key + " 给本节点投票")
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
							log.Println("节点" + key + "已确认本节点为leader")
						}
						conn.Close()
					}
				}
				n.heartBeatStart <- true
				n.voteResult <- true
				break
			}
		}
	}()
	select {
	case <-time.After(time.Duration(electionTimeout) * time.Millisecond):
		log.Println("领导人选举超时，本节点变为Follower状态")
		n.reDefault()
		return false
	case success := <-n.voteResult:
		if success {
			log.Println("本节点成功成为leader")
			return true
		}
		return false
	}
}

func (n *Node) startHeartbeat() {
	if <-n.heartBeatStart {
		// 开始心跳检测
		for {
			log.Println("发送心跳检测")
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
					ret, err := c.ResponseHeartBeat(context.Background(), &req)
					if err != nil {
						log.Panic("远程调用出错啦")
					}
					if ret.Result {
						log.Println("节点" + key + "已确认心跳包")
					}
					conn.Close()
				}
			}
			n.nextHeartBeatTime = getCurrentTime() + int64(heartBeatTimeout)
			time.Sleep(time.Millisecond * time.Duration(heartBeatPeriod))
		}
	}
}

func (n *Node) handleMessageAsLeader(Msg Message) bool {
	MessageStore[Msg.MsgID] = Msg.Msg
	confirmNum := 0
	go func() {
		for key, port := range nodeList {
			if key != n.id {
				conn, err := grpc.Dial("127.0.0.1"+port, grpc.WithInsecure())
				if err != nil {
					log.Panic("连接出错啦")
				}
				c := NodeRPC.NewRaftNodeClient(conn)
				req := NodeRPC.MessageRequest{
					Msg:   Msg.Msg,
					MsgID: Msg.MsgID,
				}
				ret, err := c.ReceiveMessageFromLeader(context.Background(), &req)
				if err != nil {
					log.Panic("远程调用出错啦")
				}
				if ret.Result {
					log.Println("节点" + key + "已收到消息")
					confirmNum++
				}
				conn.Close()
			}
		}
	}()
	for {
		time.Sleep(time.Duration(2) * time.Second)
		if confirmNum > nodeNum/2-1 {
			for key, port := range nodeList {
				if key != n.id {
					conn, err := grpc.Dial("127.0.0.1"+port, grpc.WithInsecure())
					if err != nil {
						log.Panic("连接出错啦")
					}
					c := NodeRPC.NewRaftNodeClient(conn)
					req := NodeRPC.MessageRequest{
						Msg:   Msg.Msg,
						MsgID: Msg.MsgID,
					}
					ret, err := c.ConfirmMessage(context.Background(), &req)
					if err != nil {
						log.Panic("远程调用出错啦")
					}
					if ret.Result {
						continue
					}
					conn.Close()
				}
			}
			log.Println()
			break
		}
	}
	return true
}

//
// 节点gRPC part
//
func (n *Node) TestConnection(ctx context.Context, req *NodeRPC.IDRequest) (*NodeRPC.BoolResponse, error) {
	ret := &NodeRPC.BoolResponse{
		Result: true,
	}
	return ret, nil
}

func (n *Node) Vote(ctx context.Context, req *NodeRPC.IDRequest) (*NodeRPC.BoolResponse, error) {
	if n.votedFor == "-1" && n.leader == "-1" {
		n.setVotedFor(req.ID)
		log.Println(n.id + ":已成功投票给节点" + req.ID)
		ret := &NodeRPC.BoolResponse{
			Result: true,
		}
		return ret, nil
	} else {
		ret := &NodeRPC.BoolResponse{
			Result: false,
		}
		return ret, nil
	}
}

func (n *Node) ConfirmLeader(ctx context.Context, req *NodeRPC.IDRequest) (*NodeRPC.BoolResponse, error) {
	n.reDefault()
	n.setLeader(req.ID)
	log.Println("确定节点" + req.ID + "成为leader")
	ret := &NodeRPC.BoolResponse{
		Result: true,
	}
	return ret, nil
}

func (n *Node) ConfirmMessage(ctx context.Context, req *NodeRPC.MessageRequest) (*NodeRPC.BoolResponse, error) {
	ret := &NodeRPC.BoolResponse{
		Result: true,
	}
	return ret, nil
}

func (n *Node) ResponseHeartBeat(ctx context.Context, req *NodeRPC.IDRequest) (*NodeRPC.BoolResponse, error) {
	if req.ID == n.leader {
		log.Println("接收到来自leader节点" + req.ID + "的心跳包")
		n.nextHeartBeatTime = getCurrentTime() + int64(heartBeatTimeout)
		ret := &NodeRPC.BoolResponse{
			Result: true,
		}
		return ret, nil
	}
	ret := &NodeRPC.BoolResponse{
		Result: false,
	}
	return ret, nil
}

func (n *Node) ReceiveMessageFromLeader(ctx context.Context, req *NodeRPC.MessageRequest) (*NodeRPC.BoolResponse, error) {
	log.Println("已收到来自leader的消息ID为" + req.MsgID + "的消息")
	MessageStore[req.MsgID] = req.Msg
	ret := &NodeRPC.BoolResponse{
		Result: true,
	}
	return ret, nil
}

func (n *Node) RedirectMessageToLeader(ctx context.Context, req *NodeRPC.MessageRequest) (*NodeRPC.BoolResponse, error) {
	ret := &NodeRPC.BoolResponse{
		Result: false,
	}
	return ret, nil
}

func (n *Node) ReceiveMessageFromClient(ctx context.Context, req *NodeRPC.MessageRequest) (*NodeRPC.BoolResponse, error) {
	if n.leader == n.id {

	} else {

	}
	ret := &NodeRPC.BoolResponse{
		Result: false,
	}
	return ret, nil
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
