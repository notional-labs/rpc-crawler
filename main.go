package main

import (
	"os"

	"github.com/notional-labs/rpc-crawler/lib"
)

func main() {
	initialNode := "http://localhost:26657"
	if len(os.Args) > 1 {
		initialNode = os.Args[1]
	}

	lib.CheckNode(initialNode)
	lib.WriteNodesToToml(initialNode)

}
