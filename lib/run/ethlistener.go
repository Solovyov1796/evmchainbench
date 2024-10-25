package run

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type BlockInfo struct {
	Time     int64
	TxCount  int64
	GasUsed  int64
	GasLimit int64
}

type EthereumListener struct {
	wsURL         string
	client        *ethclient.Client
	rpcClient     *rpc.Client
	headerChan    chan *types.Header
	pendingTxChan chan common.Hash
	quit          chan struct{}
}

func NewEthereumListener(wsURL string, limiter *RateLimiter) *EthereumListener {
	return &EthereumListener{
		wsURL:         wsURL,
		headerChan:    make(chan *types.Header),
		pendingTxChan: make(chan common.Hash),
		quit:          make(chan struct{}),
	}
}

func (el *EthereumListener) Connect() error {
	var err error
	el.client, err = ethclient.Dial(el.wsURL)
	if err != nil {
		return fmt.Errorf("dial error: %v", err)
	}

	el.rpcClient, err = rpc.Dial(el.wsURL)
	if err != nil {
		return fmt.Errorf("dial error: %v", err)
	}
	return nil
}

func (el *EthereumListener) Subscribe() {
	// go func() {
	subNewHeads, err := el.client.SubscribeNewHead(context.Background(), el.headerChan)
	if err != nil {
		log.Fatalf("subscribe new heads error: %v", err)
	}

	subPendingTxn, err := el.rpcClient.EthSubscribe(context.Background(), el.pendingTxChan, "newPendingTransactions")
	if err != nil {
		log.Fatalf("subscribe pending tx error: %v", err)
	}

	for {
		select {
		case err := <-subNewHeads.Err():
			log.Fatalf("Error in newHeads subscription: %v", err)
		case header := <-el.headerChan:
			jsonData, _ := json.MarshalIndent(header, "", "  ")
			fmt.Printf("New Block Header: %s\n", jsonData)
		case err := <-subPendingTxn.Err():
			log.Fatalf("Error in newPendingTransactions subscription: %v", err)
		case txHash := <-el.pendingTxChan:
			fmt.Printf("New Pending Transaction: %s\n", txHash.Hex())
		}
	}
	// }()
}

func (el *EthereumListener) Close() {
	if el.client != nil {
		el.client.Close()
	}

	if el.rpcClient != nil {
		el.rpcClient.Close()
	}
	close(el.headerChan)
	close(el.pendingTxChan)
	close(el.quit)
}
