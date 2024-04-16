package main

import (
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/notional-labs/rpc-crawler/lib"
)

func main() {
	initialNode := "http://localhost:26657"
	if len(os.Args) > 1 {
		initialNode = os.Args[1]
	}

	start := time.Now()
	color.Green("[%s] Starting RPC crawler...\n", start.Format("2006-01-02 15:04:05"))

	lib.CheckNode(initialNode)

	elapsed := time.Since(start)
	color.Green("[%s] RPC crawler completed in %s\n", time.Now().Format("2006-01-02 15:04:05"), elapsed)

	lib.WriteNodesToToml(initialNode)
}
