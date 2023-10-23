package lib

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/notional-labs/rpc-crawler/types"
)

var client = &http.Client{Timeout: 20000 * time.Millisecond}

var visited = struct {
	sync.RWMutex
	nodes map[string]bool
}{nodes: make(map[string]bool)}

func FetchStatus(nodeAddr string) (*types.StatusResponse, error) {
	url := nodeAddr + "/status"
	resp, err := HttpGet(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var status types.StatusResponse
	err = json.NewDecoder(resp.Body).Decode(&status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

func BuildRPCAddress(peer *types.Peer) string {
	rpcAddr := peer.NodeInfo.Other.RPCAddress
	rpcAddr = strings.TrimPrefix(rpcAddr, "tcp://")

	if len(rpcAddr) >= 9 && (rpcAddr[:9] == "0.0.0.0:" || rpcAddr[:9] == "127.0.0.1:") {
		rpcAddr = peer.RemoteIP + rpcAddr[8:]
	}
	return rpcAddr
}

func WriteSectionToToml(file *os.File, sectionName string, nodes map[string]int) {
	_, err := file.WriteString(fmt.Sprintf("%s = {\n", sectionName))
	if err != nil {
		fmt.Println("cannot write section to toml file")
	}
	for node, blockHeight := range nodes {
		_, err = file.WriteString(fmt.Sprintf("    \"%s\": \"%d\",\n", node, blockHeight))
		if err != nil {
			fmt.Println("cannot write node to toml file")
		}
	}
	_, err = file.WriteString("}\n\n")
	if err != nil {
		fmt.Println("cannot write escape sequence to toml file")
	}
}

// Modify the function signature to:
func ProcessPeer(peer *types.Peer) {
	rpcAddr := BuildRPCAddress(peer)
	rpcAddr = NormalizeAddressWithRemoteIP(rpcAddr, peer.RemoteIP)
	CheckNode("http://" + rpcAddr)
}

func FetchNetInfo(nodeAddr string) (*types.NetInfoResponse, error) {
	url := nodeAddr + "/net_info"
	resp, err := HttpGet(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var netInfo types.NetInfoResponse
	err = json.NewDecoder(resp.Body).Decode(&netInfo)
	return &netInfo, err
}

func NormalizeAddressWithRemoteIP(nodeAddr string, remoteIP string) string {
	nodeAddr = strings.Replace(nodeAddr, "0.0.0.0", remoteIP, -1)
	nodeAddr = strings.Replace(nodeAddr, "127.0.0.1", remoteIP, -1)
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

func HttpGet(url string) (*http.Response, error) {
	return client.Get(url)
}

func WriteNodesToToml(initialNode string) {
	file, err := os.Create("nodes.toml")
	if err != nil {
		fmt.Println("Error creating .toml file:", err)
		return
	}
	defer file.Close()

	// Write the source node to the file
	_, err = file.WriteString(fmt.Sprintf("[sourceNode]\nnode = \"%s\"\n\n", initialNode))
	if err != nil {
		fmt.Println("cannot write source node to toml file")
	}

	_, err = file.WriteString(fmt.Sprintf("totalNodesChecked = %d\n\n", totalNodesChecked))
	if err != nil {
		fmt.Println("cannot write node to toml file")
	}

	// Write sections to the file
	WriteSectionToToml(file, "successfulNodes", successfulNodes.nodes)
	WriteSectionToTomlSlice(file, "unsuccessfulNodes", unsuccessfulNodes.nodes)
	WriteSectionToTomlSlice(file, "archiveNodes", archiveNodes.nodes)

	fmt.Println(".toml file created with node details.")
}

func WriteSectionToTomlSlice(file *os.File, sectionName string, nodes []string) {
	_, err := file.WriteString(fmt.Sprintf("%s = [\n", sectionName))
	if err != nil {
		fmt.Println("cannot write section to toml file")
	}
	for _, node := range nodes {
		_, err = file.WriteString(fmt.Sprintf("    \"%s\",\n", node))
		if err != nil {
			fmt.Println("cannot write node to toml file")
		}
	}
	_, err = file.WriteString("]\n\n")
	if err != nil {
		fmt.Println("cannot write escape sequence to toml file")
	}
}
