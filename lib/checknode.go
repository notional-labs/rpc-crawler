package lib

import (
	"context"
	"fmt"
	"strings"
)

var (
	earliestBlock     map[string]int
	rpcAddr           map[string]bool
	grpcAddr          map[string]bool
	apiAddr           map[string]bool
	moniker           map[string]string
	initialNode       string
	nodeAddrGRPC      string
	totalNodesChecked int
	initialChainID    string
	archiveNodes      map[string]bool
)

func CheckNode(nodeAddr string) {
	if IsNodeVisited(nodeAddr) {
		return
	}
	MarkNodeAsVisited(nodeAddr)
	// Check if the node is the initial node
	if initialNode == "" {
		earliestBlock = map[string]int{}
		rpcAddr = map[string]bool{}
		grpcAddr = map[string]bool{}
		apiAddr = map[string]bool{}
		archiveNodes = map[string]bool{}
		moniker = map[string]string{}
		initialNode = nodeAddr
		client, err := FetchClient(nodeAddr)
		if err != nil {
			fmt.Println("Failed to fetch status from", nodeAddr)
			return
		}
		ctx := context.TODO()
		status, err := client.Status(ctx)
		if err != nil {
			fmt.Println("cannot fetch status")
		}
		initialChainID = status.NodeInfo.Network
		moniker[nodeAddr] = status.NodeInfo.Moniker
	}
	// Skip if the node address is localhost and it's not the initial node
	if nodeAddr != initialNode && strings.Contains(nodeAddr, "localhost") {
		return
	}
	// Increment total nodes
	totalNodesChecked++
	client, err := FetchClient(nodeAddr)
	if err != nil {
		fmt.Println("Failed to fetch status from", nodeAddr)
		return
	}
	netinfo, err := FetchNetInfo(client)
	if err == nil {
		fmt.Println("Got net info from", nodeAddr)
		CheckNodeGRPC(nodeAddr)
		ctx := context.TODO()
		status, err := client.Status(ctx)
		moniker[nodeAddr] = status.NodeInfo.Moniker
		if err != nil {
			fmt.Println("Failed to fetch client from", nodeAddr)
			return
		}
		// Verify chain_id
		if status.NodeInfo.Network != initialChainID {
			fmt.Println("Node", nodeAddr, "is on a different chain_id")
			return
		}
		// Add to successful nodes
		earliestBlock[nodeAddr] = int(status.SyncInfo.EarliestBlockHeight)
		rpcAddr[nodeAddr] = true
		// If the node has block 1, it's an archive node
		if int(status.SyncInfo.EarliestBlockHeight) == 1 {
			archiveNodes[nodeAddr] = true
		}
	} else {
		fmt.Println("Failed to fetch net_info from", nodeAddr)
		CheckNodeGRPC(nodeAddr)
		// Add to unsuccessful nodes
		rpcAddr[nodeAddr] = false
		return
	}
	for _, peer := range netinfo.Peers {
		peer := peer
		ProcessPeer(peer)
	}
}

func CheckNodeGRPC(nodeAddr string) {
	nodeAddrGRPC = strings.Replace(nodeAddr, "26657", "9090", 1)
	nodeAddrGRPC = strings.Replace(nodeAddrGRPC, "http://", "", 1)
	nodeAddrGRPC = strings.Replace(nodeAddrGRPC, "https://", "", 1)
	err := FetchNodeInfoGRPC(nodeAddrGRPC)
	if err == nil {
		fmt.Println("Got node info GRPC from", nodeAddrGRPC)
		// Add to successful nodes
		grpcAddr[nodeAddr] = true
	} else {
		fmt.Println("Failed to fetch node info GRPC from", nodeAddrGRPC)
		// Add to unsuccessful nodes
		grpcAddr[nodeAddr] = false
	}
}
