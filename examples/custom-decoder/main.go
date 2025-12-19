package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/84hero/evm-scanner/pkg/decoder"
	"github.com/84hero/evm-scanner/pkg/rpc"
	"github.com/84hero/evm-scanner/pkg/scanner"
	"github.com/84hero/evm-scanner/pkg/storage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Minimal ERC20 ABI for Transfer event
const erc20ABI = `[{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Setup
	rpcCfg := []rpc.NodeConfig{{URL: "https://rpc.ankr.com/eth", Priority: 1}}
	client, _ := rpc.NewClient(ctx, rpcCfg)
	store := storage.NewMemoryStore("decoder_demo_")

	// 2. Setup Decoder
	erc20Decoder, err := decoder.NewFromJSON(erc20ABI)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Define Filter (USDT)
	usdtAddr := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	transferTopic := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	filter := scanner.NewFilter().AddContract(usdtAddr).SetTopic(0, transferTopic)

	// 4. Scanner with Decoding Logic
	s := scanner.New(client, store, scanner.Config{
		ChainID: "ethereum", Rewind: 5, Interval: 5 * time.Second,
	}, filter)

	s.SetHandler(func(ctx context.Context, logs []types.Log) error {
		for _, l := range logs {
			// Try to decode the log using our ERC20 decoder
			decoded, err := erc20Decoder.Decode(l)
			if err != nil {
				fmt.Printf("Decode failed for tx %s: %v\n", l.TxHash.Hex(), err)
				continue
			}

			// Access decoded fields in a type-safe way
			from := decoded.Inputs["from"].(common.Address)
			to := decoded.Inputs["to"].(common.Address)
			value := decoded.Inputs["value"]

			fmt.Printf("🚀 [%s] Transfer Detected:\n", l.TxHash.Hex()[:10])
			fmt.Printf("   From:  %s\n", from.Hex())
			fmt.Printf("   To:    %s\n", to.Hex())
			fmt.Printf("   Value: %v\n", value)
		}
		return nil
	})

	fmt.Println("Starting USDT Transfer decoder demo...")
	go s.Start(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
