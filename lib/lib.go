package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/notional-labs/rpc-crawler/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
)

var client = &http.Client{
	Timeout: 3000 * time.Millisecond,
	Transport: &http.Transport{
		MaxIdleConns:        500,
		IdleConnTimeout:     30 * time.Second,
		MaxIdleConnsPerHost: 500,
	},
}

var visited = struct {
	sync.RWMutex
	nodes map[string]bool
}{nodes: make(map[string]bool)}

func FetchStatus(nodeAddr string) (*types.StatusResponse, error) {
	url := nodeAddr + "/status"
	resp, err := HTTPGet(url)
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

func WriteSectionToToml(file *os.File, sectionName string) {
	nodeAddrGRPC = strings.Replace(sectionName, "26657", "9090", 1)
	nodeAddrGRPC = strings.Replace(nodeAddrGRPC, "http://", "", 1)
	nodeAddrGRPC = strings.Replace(nodeAddrGRPC, "https://", "", 1)
	nodeAddrAPI = strings.Replace(sectionName, "26657", "1317", 1)
	_, err := file.WriteString(fmt.Sprintf("Starting node = %s \n", sectionName))
	if err != nil {
		fmt.Println("cannot write starting node to toml file")
	}
	_, err = file.WriteString(fmt.Sprint("[\n"))
	if err != nil {
		fmt.Println("cannot write section to toml file")
	}
	_, err = file.WriteString(fmt.Sprintf("    earliest_block = \"%d\",\n", earliest_block[sectionName]))
	if err != nil {
		fmt.Println("cannot write node to toml file")
	}
	rpc := "unsuccessful"
	if rpc_addr[sectionName] {
		rpc = "successful"
	}
	grpc := "unsuccessful"
	if grpc_addr[sectionName] {
		grpc = "successful"
	}
	api := "unsuccessful"
	if api_addr[sectionName] {
		api = "successful"
	}
	_, err = file.WriteString(fmt.Sprintf("    rpc = \"%s\",\n", sectionName))
	if err != nil {
		fmt.Println("cannot write node to toml file")
	}
	_, err = file.WriteString(fmt.Sprintf("    rpc_status = \"%s\",\n", rpc))
	if err != nil {
		fmt.Println("cannot write node to toml file")
	}
	_, err = file.WriteString(fmt.Sprintf("    grpc = \"%s\",\n", nodeAddrGRPC))
	if err != nil {
		fmt.Println("cannot write node to toml file")
	}
	_, err = file.WriteString(fmt.Sprintf("    grpc_status = \"%s\",\n", grpc))
	if err != nil {
		fmt.Println("cannot write node to toml file")
	}
	_, err = file.WriteString(fmt.Sprintf("    api = \"%s\",\n", nodeAddrAPI))
	if err != nil {
		fmt.Println("cannot write node to toml file")
	}
	_, err = file.WriteString(fmt.Sprintf("    api_status = \"%s\",\n", api))
	if err != nil {
		fmt.Println("cannot write node to toml file")
	}
	_, err = file.WriteString("]\n\n")
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

func FetchNetInfo(nodeAddr string) (*types.NetInfoResponse, error) {
	url := nodeAddr + "/net_info"
	resp, err := HTTPGet(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var netInfo types.NetInfoResponse
	err = json.NewDecoder(resp.Body).Decode(&netInfo)
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

func HTTPGet(url string) (*http.Response, error) {
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
	for key := range rpc_addr {
		WriteSectionToToml(file, key)
	}

	fmt.Println(".toml file created with node details.")
}
