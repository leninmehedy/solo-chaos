package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetupClient_InvalidNode(t *testing.T) {
	// Setup: Provide a node name not in config
	nodes := "invalidNode"
	// Mock config.ConsensusNodeInfo to return empty node
	// (Assume config.ConsensusNodeInfo is mockable)
	err := setupClient(nodes)
	assert.Error(t, err)
}

func TestStartCryptoTxWorkers_NoNodes(t *testing.T) {
	ctx := context.Background()
	nodes := ""
	totalBots := 1
	tps := 1
	duration := "1s"
	mirror := ""
	txReceipts := make(chan int, 1)

	err := startCryptoTxWorkers(ctx, nodes, totalBots, tps, duration, mirror, txReceipts)
	assert.Error(t, err)
}

func TestStartCryptoTxWorkers_InvalidDuration(t *testing.T) {
	ctx := context.Background()
	nodes := "node1"
	totalBots := 1
	tps := 1
	duration := "invalid"
	mirror := ""
	txReceipts := make(chan int, 1)

	err := startCryptoTxWorkers(ctx, nodes, totalBots, tps, duration, mirror, txReceipts)
	assert.Error(t, err)
}

func TestSendCryptoTransaction_InvalidAccount(t *testing.T) {
	botId := 1
	toAccount := "invalid"
	amount := 1.0
	traceId := "trace"
	txReceipts := make(chan int, 1)

	err := sendCryptoTransaction(botId, toAccount, amount, traceId, txReceipts)
	assert.Error(t, err)
}
