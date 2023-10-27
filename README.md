# rpc-crawler
rpc crawler for researching cosmos networks

This software will take an argument of any open rpc and it will query net_info.  Then it will recursively search all open rpc's net_info.

It will output to nodes.toml:

* total number of nodes
* successful node urls
* unsuccessful node urls
* Open GRPC
* Open API


### Install
```bash
go install github.com/notional-labs/rpc-crawler
```

### Usage

to check a local node on 26657:
```bash
rpc-crawler
```

to check a node of your choice:
```bash
rpc-crawler https://notionalapi.com/cosmos
```





