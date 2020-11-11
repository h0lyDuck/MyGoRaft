package main

import (
	"MyGoRaft/NodeRPC"
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"os"
	"time"
)

var heartBeatTimeout = 3000 // ms
var electionTimeout = 4000  // ms
var heartBeatPeriod = 3000  // ms

var nodeList map[string]string
var nodeNum int

var MessageStore = make(map[string]string)

func main() {
	nodeList = map[string]string{
		"n1": ":10001",
		"n2": ":10002",
		"n3": ":10003",
		"n4": ":10004",
		"n5": ":10005",
		"n6": ":10006",
	}
	nodeNum = len(nodeList)
	if len(os.Args) < 2 {
		log.Fatal("参数输入错误")
	}
	id := os.Args[1]
	n := initNode(id, nodeList[id])
	listener, err := net.Listen("tcp", "127.0.0.1"+nodeList[id])
	if err != nil {
		return
	}
	s := grpc.NewServer()
	NodeRPC.RegisterRaftNodeServer(s, n)
	reflection.Register(s)
	go s.Serve(listener)
	go n.startHeartbeat()
	// 检测节点是否上线
	for {
		flag := nodeNum - 1
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
				_, err = c.TestConnection(context.Background(), &req)
				if err != nil {
					log.Println("节点" + key + "还没有上线")
					flag--
				}
				conn.Close()
			}
		}
		if flag == nodeNum-1 {
			log.Println("节点全部上线")
			break
		}
	}
	time.Sleep(time.Second * time.Duration(3))
	for {
		go n.election()
		for {
			time.Sleep(time.Millisecond * 500)
			if n.nextHeartBeatTime != 0 && getCurrentTime() > n.nextHeartBeatTime {
				log.Println("心跳包超时，重新开始选举")
				n.reDefault()
				n.nextHeartBeatTime = 0
				break
			}
		}
	}
}
