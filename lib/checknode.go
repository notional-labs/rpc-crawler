package lib

import (
	"fmt"
	"strconv"
	"strings"
)

var (
	earliest_block    map[string]int
	rpc_addr          map[string]bool
	grpc_addr         map[string]bool
	api_addr          map[string]bool
	moniker           map[string]string
	initialNode       string
	nodeAddrGRPC      string
	nodeAddrAPI       string
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
		earliest_block = map[string]int{}
		rpc_addr = map[string]bool{}
		grpc_addr = map[string]bool{}
		api_addr = map[string]bool{}
		archiveNodes = map[string]bool{}
		moniker = map[string]string{}
		initialNode = nodeAddr
		status, err := FetchStatus(nodeAddr)
		if err != nil {
			fmt.Println("Failed to fetch status from", nodeAddr)
			return
		}
		initialChainID = status.Result.NodeInfo.Network
		moniker[nodeAddr] = status.Result.NodeInfo.Moniker
	}

	// Skip if the node address is localhost and it's not the initial node
	if nodeAddr != initialNode && strings.Contains(nodeAddr, "localhost") {
		return
	}

	// Increment total nodes
	totalNodesChecked++

	netinfo, err := FetchNetInfo(nodeAddr)
	if err == nil {
		fmt.Println("Got net info from", nodeAddr)
		CheckNodeGRPC(nodeAddr)
		CheckNodeAPI(nodeAddr)
		status, err := FetchStatus(nodeAddr)
		moniker[nodeAddr] = status.Result.NodeInfo.Moniker
		if err != nil {
			fmt.Println("Failed to fetch status from", nodeAddr)
			return
		}

		// Verify chain_id
		if status.Result.NodeInfo.Network != initialChainID {
			fmt.Println("Node", nodeAddr, "is on a different chain_id")
			return
		}

		// Record the earliest block height
		earliestBlockHeight, err := strconv.Atoi(status.Result.SyncInfo.EarliestBlockHeight)
		if err != nil {
			return
		}
		// Add to successful nodes
		earliest_block[nodeAddr] = earliestBlockHeight
		rpc_addr[nodeAddr] = true
		// If the node has block 1, it's an archive node
		if earliestBlockHeight == 1 {
			archiveNodes[nodeAddr] = true
		}

	} else {
		fmt.Println("Failed to fetch net_info from", nodeAddr)
		CheckNodeGRPC(nodeAddr)
		CheckNodeAPI(nodeAddr)
		// Add to unsuccessful nodes
		rpc_addr[nodeAddr] = false
		return
	}

	for _, peer := range netinfo.Result.Peers {
		peer := peer
		ProcessPeer(&peer)
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
		grpc_addr[nodeAddr] = true
	} else {
		fmt.Println("Failed to fetch node info GRPC from", nodeAddrGRPC)

		// Add to unsuccessful nodes
		grpc_addr[nodeAddr] = false
	}
}

func CheckNodeAPI(nodeAddr string) {
	nodeAddrAPI := strings.Replace(nodeAddr, "26657", "1317", 1)
	err := FetchNodeInfoAPI(nodeAddrAPI)
	if err == nil {
		fmt.Println("Got node info from", nodeAddrAPI)

		// Add to successful nodes
		api_addr[nodeAddr] = true
	} else {
		fmt.Println("Failed to fetch node info from", nodeAddrAPI)

		// Add to unsuccessful nodes
		api_addr[nodeAddr] = false
	}
}
