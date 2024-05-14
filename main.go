package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cometbft/cometbft/rpc/client/http"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/fatih/color"
)

var (
	visited = struct {
		sync.RWMutex
		nodes map[string]bool
	}{nodes: make(map[string]bool)}
	earliestBlock     map[string]int
	earliestBlockMu   sync.RWMutex
	rpcAddr           map[string]bool
	rpcAddrMu         sync.RWMutex
	moniker           map[string]string
	monikerMu         sync.RWMutex
	initialNode       string
	totalNodesChecked int
	initialChainID    string
	archiveNodes      map[string]bool
	archiveNodesMu    sync.RWMutex
)

func main() {
	initialNodes := []string{"http://localhost:26657"}
	if len(os.Args) > 1 {
		initialNodes = strings.Split(os.Args[1], " ")
	}

	start := time.Now()
	color.Green("[%s] Starting RPC crawler...\n", start.Format("2006-01-02 15:04:05"))

	for _, initialNode := range initialNodes {
		go CheckNode(initialNode)
	}

	// Wait for all goroutines to finish
	time.Sleep(180 * time.Second)

	elapsed := time.Since(start)
	color.Green("[%s] RPC crawler completed in %s\n", time.Now().Format("2006-01-02 15:04:05"), elapsed)

	WriteNodesToToml(initialNodes[0])
}

func CheckNode(nodeAddr string) {
	commonRPCPorts := []string{
		"26657",
		"36657",
		"22257",
		"14657",
		"58657",
		"33657",
		"53657",
		"37657",
		"31657",
		"10157",
		"27957",
		"2401",
		"15957",
	}

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

			// Iterate through common RPC ports if the initial connection attempt fails
			for _, port := range commonRPCPorts {
				altNodeAddr := fmt.Sprintf("%s:%s", strings.Split(nodeAddr, ":")[0], port)
				client, err = FetchClient(altNodeAddr)
				if err == nil {
					nodeAddr = altNodeAddr
					break
				}
			}

			if err != nil {
				return
			}
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
		// Add to unsuccessful nodes
		rpcAddr[nodeAddr] = false
		return
	}
}

func FetchClient(nodeAddr string) (client *http.HTTP, err error) {
	client, err = http.NewWithTimeout(nodeAddr, "websocket", 500)
	if err != nil {
		return nil, err
	}
	return client, err
}

func BuildRPCAddress(peer coretypes.Peer) string {
	rpcAddr := peer.NodeInfo.Other.RPCAddress
	rpcAddr = strings.TrimPrefix(rpcAddr, "tcp://")
	if len(rpcAddr) >= 9 && (rpcAddr[:9] == "0.0.0.0:" || rpcAddr[:9] == "127.0.0.1:") {
		rpcAddr = peer.RemoteIP + rpcAddr[8:]
	}
	return rpcAddr
}

func WriteSectionToToml(file *os.File, nodeAddr string) {
	sectionName, ok := moniker[nodeAddr]
	if !ok || sectionName == "" {
		sectionName = "default"
	}
	_, err := file.WriteString("[[nodes]]\n")
	if err != nil {
		color.Red("[%s] cannot write section to toml file\n", time.Now().Format("2006-01-02 15:04:05"))
	}
	_, err = file.WriteString(fmt.Sprintf("name = \"%s\"\n", sectionName))
	if err != nil {
		color.Red("[%s] cannot write section to toml file\n", time.Now().Format("2006-01-02 15:04:05"))
	}
	_, err = file.WriteString(fmt.Sprintf("earliest_block = %d\n", earliestBlock[nodeAddr]))
	if err != nil {
		color.Red("[%s] cannot write node to toml file\n", time.Now().Format("2006-01-02 15:04:05"))
	}
	writeStatus := func(service string, addr string, statusMap map[string]bool) {
		status := "unsuccessful"
		if statusMap[nodeAddr] {
			status = "successful"
		}
		_, err = file.WriteString(fmt.Sprintf("%s = \"%s\"\n", service, addr))
		if err != nil {
			color.Red("[%s] cannot write node to toml file\n", time.Now().Format("2006-01-02 15:04:05"))
		}
		_, err = file.WriteString(fmt.Sprintf("%s_status = \"%s\"\n", service, status))
		if err != nil {
			color.Red("[%s] cannot write node to toml file\n", time.Now().Format("2006-01-02 15:04:05"))
		}
	}
	writeStatus("rpc", nodeAddr, rpcAddr)
	_, err = file.WriteString("\n")
	if err != nil {
		color.Red("[%s] cannot write escape sequence to toml file\n", time.Now().Format("2006-01-02 15:04:05"))
	}
}

func ProcessPeer(peer coretypes.Peer) {
	rpcAddr := BuildRPCAddress(peer)
	rpcAddr = NormalizeAddressWithRemoteIP(rpcAddr, peer.RemoteIP)
	CheckNode("http://" + rpcAddr)
}

func FetchNetInfo(client *http.HTTP) (*coretypes.ResultNetInfo, error) {
	ctx := context.Background()
	netinfo, err := client.NetInfo(ctx)
	return netinfo, err
}

func NormalizeAddressWithRemoteIP(nodeAddr string, remoteIP string) string {
	nodeAddr = strings.ReplaceAll(nodeAddr, "0.0.0.0", remoteIP)
	nodeAddr = strings.ReplaceAll(nodeAddr, "127.0.0.1", remoteIP)
	return nodeAddr
}

func IsNodeVisited(nodeAddr string) bool {
	visited.RLock()
	defer visited.RUnlock()
	_, ok := visited.nodes[nodeAddr]
	return ok
}

func MarkNodeAsVisited(nodeAddr string) {
	visited.Lock()
	defer visited.Unlock()
	visited.nodes[nodeAddr] = true
}

func WriteNodesToToml(initialNode string) {
	file, err := os.Create("nodes.toml")
	if err != nil {
		color.Red("[%s] Error creating .toml file: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
		return
	}
	defer file.Close()
	// Write the source node to the file
	_, err = file.WriteString(fmt.Sprintf("source_node = \"%s\"\n\n", initialNode))
	if err != nil {
		color.Red("[%s] cannot write source node to toml file\n", time.Now().Format("2006-01-02 15:04:05"))
	}
	_, err = file.WriteString(fmt.Sprintf("total_nodes_checked = %d\n\n", totalNodesChecked))
	if err != nil {
		color.Red("[%s] cannot write node to toml file\n", time.Now().Format("2006-01-02 15:04:05"))
	}
	// Write sections to the file
	for key := range rpcAddr {
		WriteSectionToToml(file, key)
	}
	WriteSectionToTomlSlice(file, "successful_rpc_nodes", rpcAddr, true)
	WriteSectionToTomlSlice(file, "unsuccessful_rpc_nodes", rpcAddr, false)
	color.Green("[%s] .toml file created with node details.\n", time.Now().Format("2006-01-02 15:04:05"))
}

func WriteSectionToTomlSlice(file *os.File, sectionName string, nodes map[string]bool, status bool) {
	_, err := file.WriteString(fmt.Sprintf("%s = [\n", sectionName))
	if err != nil {
		color.Red("[%s] cannot write node to toml file\n", time.Now().Format("2006-01-02 15:04:05"))
	}
	for key, val := range nodes {
		if val == status {
			_, err = file.WriteString(fmt.Sprintf("  \"%s\",\n", key))
			if err != nil {
				color.Red("[%s] cannot write node to toml file\n", time.Now().Format("2006-01-02 15:04:05"))
			}
		}
	}
	_, err = file.WriteString("]\n\n")
	if err != nil {
		color.Red("[%s] cannot write node to toml file\n", time.Now().Format("2006-01-02 15:04:05"))
	}
}
