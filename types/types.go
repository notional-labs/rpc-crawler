package types

type Peer struct {
	NodeInfo struct {
		Other struct {
			RPCAddress string `json:"rpc_address"`
		} `json:"other"`
	} `json:"node_info"`
	RemoteIP string `json:"remote_ip"`
}

type NetInfoResponse struct {
	Result struct {
		Peers []Peer `json:"peers"`
	} `json:"result"`
}

type StatusResponse struct {
	Result struct {
		NodeInfo struct {
			Network string `json:"network"`
		} `json:"node_info"`
		SyncInfo struct {
			EarliestBlockHeight int `json:"earliest_block_height"`
			LatestBlockHeight   int `json:"latest_block_height"`
		} `json:"sync_info"`
	} `json:"result"`
}
