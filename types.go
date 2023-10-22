package main

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
		SyncInfo struct {
			EarliestBlockHeight string `json:"earliest_block_height"`
			LatestBlockHeight   string `json:"latest_block_height"`
		} `json:"sync_info"`
	} `json:"result"`
}
