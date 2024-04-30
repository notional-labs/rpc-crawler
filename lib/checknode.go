package lib

import (
	"context"
	"strings"
	"sync"
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/fatih/color"
)

var (
	earliestBlock     map[string]int
	earliestBlockMu   sync.RWMutex
	rpcAddr           map[string]bool
	rpcAddrMu         sync.RWMutex
	grpcAddr          map[string]bool
	grpcAddrMu        sync.RWMutex
	apiAddr           map[string]bool
	apiAddrMu         sync.RWMutex
	moniker           map[string]string
	monikerMu         sync.RWMutex
	initialNode       string
	nodeAddrGRPC      string
	totalNodesChecked int
	initialChainID    string
	archiveNodes      map[string]bool
	archiveNodesMu    sync.RWMutex
)

func CheckNode(nodeAddr string) {
	if IsNodeVisited(nodeAddr) {
		return
	}
	MarkNodeAsVisited(nodeAddr)
	// Check if the node is the initial node
	if initialNode == "" {
		earliestBlockMu.Lock()
		earliestBlock = map[string]int{}
		earliestBlockMu.Unlock()

		rpcAddrMu.Lock()
		rpcAddr = map[string]bool{}
		rpcAddrMu.Unlock()

		grpcAddrMu.Lock()
		grpcAddr = map[string]bool{}
		grpcAddrMu.Unlock()

		apiAddrMu.Lock()
		apiAddr = map[string]bool{}
		apiAddrMu.Unlock()

		archiveNodesMu.Lock()
		archiveNodes = map[string]bool{}
		archiveNodesMu.Unlock()

		monikerMu.Lock()
		moniker = map[string]string{}
		monikerMu.Unlock()

		initialNode = nodeAddr
		client, err := FetchClient(nodeAddr)
		if err != nil {
			color.Red("[%s] Failed to fetch status from %s\n", time.Now().Format("2006-01-02 15:04:05"), nodeAddr)
			return
		}
		if client == nil {
			color.Red("[%s] Client is nil for %s\n", time.Now().Format("2006-01-02 15:04:05"), nodeAddr)
			return
		}

		ctx := context.TODO()
		status, err := client.Status(ctx)
		if err != nil {
			color.Red("[%s] cannot fetch status\n", time.Now().Format("2006-01-02 15:04:05"))
			return
		}
		if status == nil {
			color.Red("[%s] Status is nil for %s\n", time.Now().Format("2006-01-02 15:04:05"), nodeAddr)
			return
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
		color.Red("[%s] Failed to fetch status from %s\n", time.Now().Format("2006-01-02 15:04:05"), nodeAddr)
		return
	}
	if client == nil {
		color.Red("[%s] Client is nil for %s\n", time.Now().Format("2006-01-02 15:04:05"), nodeAddr)
		return
	}
	netinfo, err := FetchNetInfo(client)
	if err == nil {
		color.Green("[%s] Got net info from %s\n", time.Now().Format("2006-01-02 15:04:05"), nodeAddr)
		CheckNodeGRPC(nodeAddr)
		ctx := context.TODO()
		status, err := client.Status(ctx)
		moniker[nodeAddr] = status.NodeInfo.Moniker
		if err != nil {
			color.Red("[%s] Failed to fetch client from %s\n", time.Now().Format("2006-01-02 15:04:05"), nodeAddr)
			return
		}
		if status == nil {
			color.Red("[%s] Status is nil for %s\n", time.Now().Format("2006-01-02 15:04:05"), nodeAddr)
			return
		}
		// Verify chain_id
		if status.NodeInfo.Network != initialChainID {
			color.Red("[%s] Node %s is on a different chain_id\n", time.Now().Format("2006-01-02 15:04:05"), nodeAddr)
			return
		}
		// Add to successful nodes
		earliestBlock[nodeAddr] = int(status.SyncInfo.EarliestBlockHeight)
		rpcAddr[nodeAddr] = true
		// If the node has block 1, it's an archive node
		if int(status.SyncInfo.EarliestBlockHeight) == 1 {
			archiveNodes[nodeAddr] = true
		}
		var wg sync.WaitGroup
		for _, peer := range netinfo.Peers {
			wg.Add(1)
			go func(peer coretypes.Peer) {
				defer wg.Done()
				ProcessPeer(peer)
			}(peer)
		}
		wg.Wait()
	} else {
		color.Red("[%s] Failed to fetch net_info from %s\n", time.Now().Format("2006-01-02 15:04:05"), nodeAddr)
		CheckNodeGRPC(nodeAddr)
		// Add to unsuccessful nodes
		rpcAddr[nodeAddr] = false
		return
	}
}

func CheckNodeGRPC(nodeAddr string) {
	nodeAddrGRPC = strings.Replace(nodeAddr, "26657", "9090", 1)
	nodeAddrGRPC = strings.Replace(nodeAddrGRPC, "http://", "", 1)
	nodeAddrGRPC = strings.Replace(nodeAddrGRPC, "https://", "", 1)
	err := FetchNodeInfoGRPC(nodeAddrGRPC)
	if err == nil {
		color.Green("[%s] Got node info GRPC from %s\n", time.Now().Format("2006-01-02 15:04:05"), nodeAddrGRPC)
		// Add to successful nodes
		grpcAddr[nodeAddr] = true
	} else {
		color.Red("[%s] Failed to fetch node info GRPC from %s\n", time.Now().Format("2006-01-02 15:04:05"), nodeAddrGRPC)
		// Add to unsuccessful nodes
		grpcAddr[nodeAddr] = false
	}
}
