const axios = require('axios');

const visitedNodes = new Set();
const timeout = 3000; // Timeout for requests in milliseconds

async function fetchNetInfo(url) {
    try {
        const response = await axios.get(url, { timeout });
        return response.data.result;
    } catch (error) {
        console.error(`Error fetching ${url}:`, error.message);
        return null;
    }
}

async function fetchStatus(url) {
    try {
        const response = await axios.get(url, { timeout });
        return response.data.result;
    } catch (error) {
        console.error(`Error fetching ${url}:`, error.message);
        return null;
    }
}

async function crawlNetwork(url, maxDepth, currentDepth = 0) {
    if (currentDepth > maxDepth) {
        return;
    }

    const netInfo = await fetchNetInfo(url);
    if (!netInfo) {
        return;
    }

    const statusUrl = url.replace('/net_info', '/status');
    const statusInfo = await fetchStatus(statusUrl);
    if (statusInfo) {
        const earliestBlockHeight = statusInfo.sync_info.earliest_block_height;
        const earliestBlockTime = statusInfo.sync_info.earliest_block_time;
        console.log(`Node: ${url}`);
        console.log(`Earliest Block Height: ${earliestBlockHeight}`);
        console.log(`Earliest Block Time: ${earliestBlockTime}`);
    }

    const peers = netInfo.peers;
    const crawlPromises = peers.map(async (peer) => {
        const remoteIp = peer.remote_ip;
        const rpcAddress = peer.node_info.other.rpc_address.replace('tcp://', 'http://').replace('0.0.0.0', remoteIp);

        if (!visitedNodes.has(rpcAddress)) {
            visitedNodes.add(rpcAddress);
            console.log(`Crawling: ${rpcAddress}`);
            await crawlNetwork(`${rpcAddress}/net_info`, maxDepth, currentDepth + 1);
        }
    });

    await Promise.all(crawlPromises);
}

const initialUrl = 'https://sifchain-rpc.polkachu.com:443/net_info';
const maxDepth = 3; // Set the maximum depth for recursion

crawlNetwork(initialUrl, maxDepth).then(() => {
    console.log('Crawling complete.');
}).catch((error) => {
    console.error('Error during crawling:', error);
});

