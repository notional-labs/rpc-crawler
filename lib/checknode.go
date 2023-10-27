package lib

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

var initialChainID string

var archiveNodes = struct {
	sync.RWMutex
	nodes []string
}{nodes: []string{}}

var (
	totalNodesChecked int
	successfulNodes   = struct {
		sync.RWMutex
		nodes map[string]int
	}{nodes: make(map[string]int)}
)

var unsuccessfulNodes = struct {
	sync.RWMutex
	nodes []string
}{nodes: []string{}}

var successfulNodesGRPC = struct {
	sync.RWMutex
	nodes []string
}{nodes: []string{}}

var unsuccessfulNodesGRPC = struct {
	sync.RWMutex
	nodes []string
}{nodes: []string{}}

var successfulNodesAPI = struct {
	sync.RWMutex
	nodes []string
}{nodes: []string{}}

var unsuccessfulNodesAPI = struct {
	sync.RWMutex
	nodes []string
}{nodes: []string{}}

var (
	initialNode  string
	nodeAddrGRPC string
)

func CheckNode(nodeAddr string) {
	if IsNodeVisited(nodeAddr) {
		return
	}

	MarkNodeAsVisited(nodeAddr)

	// Check if the node is the initial node
	if initialNode == "" {
		initialNode = nodeAddr
		status, err := FetchStatus(nodeAddr)
		if err != nil {
			fmt.Println("Failed to fetch status from", nodeAddr)
			return
		}
		initialChainID = status.Result.NodeInfo.Network
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
		successfulNodes.Lock()
		successfulNodes.nodes[nodeAddr] = earliestBlockHeight
		successfulNodes.Unlock()

		// If the node has block 1, it's an archive node
		if earliestBlockHeight == 1 {
			archiveNodes.Lock()
			archiveNodes.nodes = append(archiveNodes.nodes, nodeAddr)
			archiveNodes.Unlock()
		}

	} else {
		fmt.Println("Failed to fetch net_info from", nodeAddr)
		CheckNodeGRPC(nodeAddr)
		CheckNodeAPI(nodeAddr)
		// Add to unsuccessful nodes
		unsuccessfulNodes.Lock()
		unsuccessfulNodes.nodes = append(unsuccessfulNodes.nodes, nodeAddr)
		unsuccessfulNodes.Unlock()
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
		successfulNodesGRPC.Lock()
		successfulNodesGRPC.nodes = append(successfulNodesGRPC.nodes, nodeAddrGRPC)
		successfulNodesGRPC.Unlock()
	} else {
		fmt.Println("Failed to fetch node info GRPC from", nodeAddrGRPC)

		// Add to unsuccessful nodes
		unsuccessfulNodesGRPC.Lock()
		unsuccessfulNodesGRPC.nodes = append(unsuccessfulNodesGRPC.nodes, nodeAddrGRPC)
		unsuccessfulNodesGRPC.Unlock()
	}
}

func CheckNodeAPI(nodeAddr string) {
	nodeAddrAPI := strings.Replace(nodeAddr, "26657", "1317", 1)
	err := FetchNodeInfoAPI(nodeAddrAPI)
	if err == nil {
		fmt.Println("Got node info from", nodeAddrAPI)

		// Add to successful nodes
		successfulNodesAPI.Lock()
		successfulNodesAPI.nodes = append(successfulNodesAPI.nodes, nodeAddrAPI)
		successfulNodesAPI.Unlock()
	} else {
		fmt.Println("Failed to fetch node info from", nodeAddrAPI)

		// Add to unsuccessful nodes
		unsuccessfulNodesAPI.Lock()
		unsuccessfulNodesAPI.nodes = append(unsuccessfulNodesAPI.nodes, nodeAddrAPI)
		unsuccessfulNodesAPI.Unlock()
	}
}
