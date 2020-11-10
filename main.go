package MyGoRaft

var heartBeatTimeout = 3

var nodeList map[string]string
var nodeNum int

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
}
