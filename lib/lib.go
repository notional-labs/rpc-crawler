package lib

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/notional-labs/rpc-crawler/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var client = &http.Client{
	Timeout: 300 * time.Millisecond,
	Transport: &http.Transport{
		MaxIdleConns:        500,
		IdleConnTimeout:     90 * time.Second,
		MaxIdleConnsPerHost: 500,
	},
}

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
	tlsCfg := &tls.Config{
		InsecureSkipVerify: true,
	}
	grpcConn, err := grpc.Dial(
		nodeAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)), // The Cosmos SDK doesn't support any transport security mechanism.
		// This instantiates a general gRPC codec which handles proto bytes. We pass in a nil interface registry
		// if the request/response types contain interface instead of 'nil' you should pass the application specific codec.
		grpc.WithDefaultCallOptions(grpc.ForceCodec(codec.NewProtoCodec(nil).GRPCCodec())),
	)
	if err != nil {
		return err
	}

	defer grpcConn.Close()

	serviceClient := tmservice.NewServiceClient(grpcConn)
	nodeInfoRes, err := serviceClient.GetNodeInfo(
		context.Background(),
		&tmservice.GetNodeInfoRequest{},
	)

	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(nodeInfoRes)
	return err
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

	// Write sections to the file
	WriteSectionToTomlSlice(file, "successfulNodesGRPC", successfulNodesGRPC.nodes)
	WriteSectionToTomlSlice(file, "unsuccessfulNodesGRPC", unsuccessfulNodesGRPC.nodes)

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
