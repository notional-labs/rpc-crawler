package lib

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/cometbft/cometbft/rpc/client/http"
	"github.com/notional-labs/rpc-crawler/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
)

var visited = struct {
	sync.RWMutex
	nodes map[string]bool
}{nodes: make(map[string]bool)}

func FetchClient(nodeAddr string) (client *http.HTTP, err error) {
	client, err = http.New(nodeAddr, "websocket")
	if err != nil {
		return nil, err
	}

	return client, err
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

	// Fetch network info
	netInfo, err := FetchNetInfo("http://" + rpcAddr)
	if err != nil {
		//		fmt.Println("Error fetching network info:", err)
		return
	}

	// Process each peer
	for _, peer := range netInfo.Result.Peers {
		go func(peer types.Peer) {
			if !IsNodeVisited(peer.NodeInfo.Other.RPCAddress) {
				MarkNodeAsVisited(peer.NodeInfo.Other.RPCAddress)
				ProcessPeer(&peer)
			}
		}(peer)
	}
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
		fmt.Println(err)
		return err
	}

	return err
}

func FetchNetInfo(client string) (*types.NetInfoResponse, error) {

	return &netInfo, err
}

func FetchNodeInfoAPI(nodeAddr string) error {
	url := nodeAddr + "/cosmos/base/tendermint/v1beta1/node_info"
	resp, err := HTTPGet(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return err
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

	// Write sections to the file
	WriteSectionToTomlSlice(file, "successfulNodesGRPC", successfulNodesGRPC.nodes)
	WriteSectionToTomlSlice(file, "unsuccessfulNodesGRPC", unsuccessfulNodesGRPC.nodes)
	WriteSectionToTomlSlice(file, "successfulNodesAPI", successfulNodesAPI.nodes)
	WriteSectionToTomlSlice(file, "unsuccessfulNodesAPI", unsuccessfulNodesAPI.nodes)

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
