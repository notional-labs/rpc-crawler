package lib

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/cometbft/cometbft/rpc/client/http"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/fatih/color"
)

var visited = struct {
	sync.RWMutex
	nodes map[string]bool
}{nodes: make(map[string]bool)}

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
	nodeAddrGRPC := strings.Replace(nodeAddr, "26657", "9090", 1)
	nodeAddrGRPC = strings.Replace(nodeAddrGRPC, "http://", "", 1)
	nodeAddrGRPC = strings.Replace(nodeAddrGRPC, "https://", "", 1)
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
	writeStatus("grpc", nodeAddrGRPC, grpcAddr)
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

func FetchNodeInfoGRPC(nodeAddr string) error {
	grpcConn, err := grpc.Dial(
		nodeAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	defer grpcConn.Close()
	serviceClient := tmservice.NewServiceClient(grpcConn)
	_, err = serviceClient.GetNodeInfo(
		context.Background(),
		&tmservice.GetNodeInfoRequest{},
	)
	if err != nil {
		color.Red("[%s] %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
		return err
	}
	return err
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
	WriteSectionToTomlSlice(file, "successful_grpc_nodes", grpcAddr, true)
	WriteSectionToTomlSlice(file, "unsuccessful_grpc_nodes", grpcAddr, false)
	WriteSectionToTomlSlice(file, "successful_api_nodes", apiAddr, true)
	WriteSectionToTomlSlice(file, "unsuccessful_api_nodes", apiAddr, false)
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
