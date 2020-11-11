package main

import (
	"MyGoRaftClient/NodeRPC"
	"context"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/grpc"
	"log"
)

var nodeList map[string]string

func main() {
	nodeList = map[string]string{
		"n1": ":10001",
		"n2": ":10002",
		"n3": ":10003",
		"n4": ":10004",
		"n5": ":10005",
		"n6": ":10006",
	}
	for {
		var input string
		fmt.Print("请输入你要发送的信息\n>> ")
		fmt.Scanln(&input)
		ul := uuid.Must(uuid.NewV4(), nil).String()
		conn, err := grpc.Dial("127.0.0.1"+nodeList["n1"], grpc.WithInsecure())
		if err != nil {
			log.Panic("连接出错啦")
		}
		c := NodeRPC.NewRaftNodeClient(conn)
		req := NodeRPC.MessageRequest{
			Msg:   input,
			MsgID: ul,
		}
		ret, err := c.ReceiveMessageFromClient(context.Background(), &req)
		if err != nil {
			fmt.Println(err)
		}
		if ret.Result {
			fmt.Println("消息已成功提交")
		}
		conn.Close()
	}
}
