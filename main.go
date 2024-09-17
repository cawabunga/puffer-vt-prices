package main

import (
	"context"
	"encoding/json"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const EVENTS_JSON_PATH = "docs/events.json"

func main() {
	pufferOracleV2Bytes, err := os.ReadFile("abi/PufferOracleV2.json")
	if err != nil {
		log.Fatal(err)
	}

	pufferOracleV2Abi, err := abi.JSON(strings.NewReader(string(pufferOracleV2Bytes)))
	if err != nil {
		log.Fatal(err)
	}

	eventParserInstance, err := NewEventParse(pufferOracleV2Abi, "ValidatorTicketMintPriceUpdated")
	if err != nil {
		log.Fatal(err)
	}

	logs := make(chan types.Log)
	entriesCh := make(chan entry)
	priceUpdateEvents := make(chan struct {
		types.Log
		priceUpdateEvent
	})

	go func() {
		defer close(logs)

		lastBlock := 0

		if bytes, err := os.ReadFile(EVENTS_JSON_PATH); err == nil {
			var entries []entry
			err = json.Unmarshal(bytes, &entries)
			if err != nil {
				log.Fatal(err)
			}
			for _, e := range entries {
				lastBlock = int(e.BlockNumber)
				entriesCh <- e
			}
		}

		fetchLogs(lastBlock+1, logs)
	}()

	go func() {
		defer close(priceUpdateEvents)

		for a := range logs {
			if !eventParserInstance.CanParse(a) {
				continue
			}

			var parsedEvent priceUpdateEvent

			err := eventParserInstance.Parse(a, &parsedEvent)
			if err != nil {
				log.Fatal(err)
			}

			priceUpdateEvents <- struct {
				types.Log
				priceUpdateEvent
			}{a, parsedEvent}
		}
	}()

	go func() {
		defer close(entriesCh)

		for event := range priceUpdateEvents {
			// send requests twice in a second
			throttleTimer := time.After(500 * time.Millisecond)

			client, err := ethclient.Dial(RPC_URL)
			if err != nil {
				log.Fatal(err)
			}

			block, err := client.HeaderByNumber(context.Background(), big.NewInt(int64(event.BlockNumber)))

			if err != nil {
				log.Fatal(err)
			}

			entriesCh <- entry{
				Timestamp:   uint32(block.Time),
				BlockNumber: uint32(event.BlockNumber),
				Event:       event.priceUpdateEvent,
			}

			// wait for while, in order to not to get banned
			<-throttleTimer
		}
	}()

	entries := make([]entry, 0)
	for entry := range entriesCh {
		data, err := json.Marshal(entry)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(string(data))
		entries = append(entries, entry)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(EVENTS_JSON_PATH, data, 0644); err != nil {
		log.Fatal(err)
	}
	log.Printf("Writting to %s\n", EVENTS_JSON_PATH)
}

type entry struct {
	Timestamp   uint32           `json:"timestamp"`
	BlockNumber uint32           `json:"blockNumber"`
	Event       priceUpdateEvent `json:"event"`
}

type priceUpdateEvent struct {
	OldPrice *big.Int `json:"oldPrice"`
	NewPrice *big.Int `json:"newPrice"`
}

const RPC_URL = "https://eth.llamarpc.com"
const ORACLE_ADDRESS = "0x0BE2aE0edbeBb517541DF217EF0074FC9a9e994f"
const ORACLE_DEPLOY_BLOCK = 19845895

func fetchLogs(fromBlock int, logs chan<- types.Log) {
	client, err := ethclient.Dial(RPC_URL)
	if err != nil {
		log.Fatal(err)
	}

	lastBlock, err := client.BlockNumber(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	maxSize := 10000
	start := max(fromBlock, ORACLE_DEPLOY_BLOCK)

	for i := start; i <= int(lastBlock); i += maxSize {
		// send requests twice in a second
		throttleTimer := time.After(500 * time.Millisecond)
		log.Printf("Fetching logs, left %d\n", int(lastBlock)-i)

		a, err := client.FilterLogs(context.Background(), ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(i)),
			ToBlock:   big.NewInt(int64(i + maxSize - 1)),
			Addresses: []common.Address{common.HexToAddress(ORACLE_ADDRESS)},
		})

		if err != nil {
			log.Fatal(err)
		}

		for _, log := range a {
			logs <- log
		}

		// wait for while, in order to not to get banned
		<-throttleTimer
	}
}
