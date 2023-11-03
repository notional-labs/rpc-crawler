package lib

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"

	"github.com/cometbft/cometbft/rpc/client/http"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
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

	_, err := file.WriteString(fmt.Sprintf("[%s]\n", sectionName))
	if err != nil {
		fmt.Println("cannot write section to toml file")
	}

	_, err = file.WriteString(fmt.Sprintf("    earliest_block = \"%d\",\n", earliestBlock[nodeAddr]))
	if err != nil {
		fmt.Println("cannot write node to toml file")
	}

	writeStatus := func(service string, addr string, statusMap map[string]bool) {
		status := "unsuccessful"
		if statusMap[nodeAddr] {
			status = "successful"
		}
		_, err = file.WriteString(fmt.Sprintf("    %s = \"%s\",\n", service, addr))
		if err != nil {
			fmt.Println("cannot write node to toml file")
		}
		_, err = file.WriteString(fmt.Sprintf("    %s_status = \"%s\",\n", service, status))
		if err != nil {
			fmt.Println("cannot write node to toml file")
		}
	}

	writeStatus("rpc", nodeAddr, rpcAddr)
	writeStatus("grpc", nodeAddrGRPC, grpcAddr)

	_, err = file.WriteString("]\n\n")
	if err != nil {
		fmt.Println("cannot write escape sequence to toml file")
	}
}

func ProcessPeer(peer coretypes.Peer) {
	rpcAddr := BuildRPCAddress(peer)
	rpcAddr = NormalizeAddressWithRemoteIP(rpcAddr, peer.RemoteIP)
	CheckNode("http://" + rpcAddr)

	// Fetch network info
	client, err := FetchClient("http://" + rpcAddr)
	if err != nil {
		//		fmt.Println("Error fetching network info:", err)
		return
	}
	netInfo, err := FetchNetInfo(client)
	if err != nil {
		//		fmt.Println("Error fetching network info:", err)
		return
	}

	// Process each peer
	for _, peer := range netInfo.Peers {
		go func(peer coretypes.Peer) {
			if !IsNodeVisited(peer.NodeInfo.Other.RPCAddress) {
				MarkNodeAsVisited(peer.NodeInfo.Other.RPCAddress)
				ProcessPeer(peer)
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
	for key := range rpcAddr {
		WriteSectionToToml(file, key)
	}

	WriteSectionToTomlSlice(file, "successfulRPCNodes", rpcAddr, true)
	WriteSectionToTomlSlice(file, "unsuccessfulRPCNodes", rpcAddr, false)
	WriteSectionToTomlSlice(file, "successfulGRPCNodes", grpcAddr, true)
	WriteSectionToTomlSlice(file, "unsuccessfulGRPCNodes", grpcAddr, false)
	WriteSectionToTomlSlice(file, "successfulAPINodes", apiAddr, true)
	WriteSectionToTomlSlice(file, "unsuccessfulAPINodes", apiAddr, false)

	fmt.Println(".toml file created with node details.")
}

func WriteSectionToTomlSlice(file *os.File, sectionName string, nodes map[string]bool, status bool) {
	_, err := file.WriteString(fmt.Sprintf("%s = [\n", sectionName))
	if err != nil {
		fmt.Println("cannot write node to toml file")
	}

	for key, val := range nodes {
		if val == status {
			_, err = file.WriteString(fmt.Sprintf("    %s\n", key))
			if err != nil {
				fmt.Println("cannot write node to toml file")
			}
		}
	}

	_, err = file.WriteString("]\n")
	if err != nil {
		fmt.Println("cannot write node to toml file")
	}
}
