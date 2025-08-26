package commands

import (
	"context"
	"fmt"
	"github.com/automa-saga/logx"
	hiero "github.com/hiero-ledger/hiero-sdk-go/v2/sdk"
	"github.com/joomcode/errorx"
	"github.com/leninmehedy/solo-chaos/cmd/hammer/config"
	"github.com/spf13/cobra"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	TxTypeCrypto = "crypto"
)

var client *hiero.Client

var txCmd = &cobra.Command{
	Use:   "tx",
	Short: "Create transaction",
	Long:  "Create transaction and set to node randomly",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.ParseFlags(args); err != nil {
			logx.As().Error().Err(err).Msg("Failed to parse flags")
			os.Exit(1)
		}
		if cmd.Context() == nil {
			logx.As().Error().Msg("Context is nil")
			os.Exit(1)
		}
		runTx(cmd.Context())
	},
}

func runTx(ctx context.Context) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	if err := config.Initialize(flagConfig); err != nil {
		logx.As().Fatal().Err(err).Msg("Failed to initialize config")
	}
	logx.As().Info().Msg("Configuration initialized")

	logx.As().Info().
		Str("nodes", flagNodes).
		Int("total_bots", flagBots).
		Int("tps", flagTps).
		Str("duration", flagDuration).
		Str("mirror-node", flagMirror).
		Str("tx-type", flagTxType).
		Msg("Starting transaction load generation")

	if err := setupClient(flagNodes); err != nil {
		logx.As().Fatal().Err(err).Msg("Failed to setup client")
	}

	var wg sync.WaitGroup
	var err error

	// compute total tps
	totalBots := flagBots
	txReceipts := make(chan int, totalBots)
	var totalTps float64
	var totalTime float64
	var totalTx int64
	defer close(txReceipts)
	go func() {
		start := time.Now()
		for {
			select {
			case <-ctx.Done():
				return
			case <-txReceipts:
				totalTx++
				totalTime = time.Since(start).Seconds()
				totalTps = math.Round(float64(totalTx) / totalTime)
				logx.As().Info().
					Int64("total_tx", totalTx).
					Float64("total_time_sec", totalTime).
					Float64("tps", totalTps).
					Msg("Received a transaction receipt")
			}
		}
	}()

	// start transaction workers based on tx type
	switch flagTxType {
	case TxTypeCrypto:
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = startCryptoTxWorkers(ctx, flagNodes, flagBots, flagTps, flagDuration, flagMirror, txReceipts)
		}()
	default:
		logx.As().Error().Str("tx-type", flagTxType).Msg("Unsupported transaction type")
		os.Exit(1)
	}

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		wg.Wait()
		cancelFunc()
		logx.As().Info().
			Int64("total_tx", totalTx).
			Float64("total_time_sec", totalTime).
			Float64("tps", totalTps).
			Msg("All transaction bots completed")
	}()

	select {
	case <-sigCh:
		logx.As().Trace().Msg("Received exit signal, stopping pipelines...")
		cancelFunc()
	case <-ctx.Done():
	}

	if err != nil {
		logx.As().Error().Err(err).Msg("Transaction workers failed")
		os.Exit(1)
	}
}

func setupClient(nodes string) error {
	if client != nil {
		return nil
	}

	network := config.Network()

	// if nodes flag is provided, filter the network settings based on the provided nodes
	nodeList := strings.Split(nodes, ",")
	if len(nodeList) > 0 {
		network = make(map[string]hiero.AccountID)
		for _, nodeName := range nodeList {
			node := config.ConsensusNodeInfo(nodeName)
			if node.Name == "" {
				return errorx.IllegalArgument.New("node %s not found in config", nodeName)
			}
			accountID, err := hiero.AccountIDFromString(node.Account)
			if err != nil {
				return errorx.IllegalArgument.Wrap(err, "error converting string to AccountID: %s", node.Account)
			}
			network[node.Endpoint] = accountID
		}
	}

	var err error
	client, err = hiero.ClientForNetworkV2(network)
	if err != nil {
		return errorx.IllegalState.Wrap(err, "error creating client")
	}

	operatorAccountID, err := hiero.AccountIDFromString(config.Get().Operator.Account)
	if err != nil {
		return errorx.IllegalState.Wrap(err, "error converting string to operator AccountID")
	}

	operatorKey, err := hiero.PrivateKeyFromString(config.Get().Operator.Key)
	if err != nil {
		return errorx.IllegalState.Wrap(err, "error converting string to operator key")
	}

	client.SetOperator(operatorAccountID, operatorKey)

	return nil
}

func startCryptoTxWorkers(ctx context.Context, nodes string, totalBots, tps int, duration, mirror string, txReceipts chan int) error {
	d, err := time.ParseDuration(duration)
	if err != nil {
		return err
	}

	nodesList := strings.Split(nodes, ",")
	if len(nodesList) == 0 {
		return errorx.IllegalArgument.New("no nodes provided")
	}

	logx.As().Info().
		Dur("duration", d).
		Int("tps", tps).
		Int("total_bots", totalBots).
		Int("total_tps", tps*totalBots).
		Strs("nodes", nodesList).
		Str("mirror_node", mirror).
		Msg("Starting crypto transaction bots")

	tickerDuration := time.Second / time.Duration(tps)
	errCh := make(chan error, totalBots)
	var wg sync.WaitGroup

	botFunc := func(botId int) {
		defer wg.Done()
		ticker := time.NewTicker(tickerDuration)
		timer := time.NewTimer(d)
		defer ticker.Stop()
		defer timer.Stop()
		counter := int64(0)
		for {
			select {
			case <-ticker.C:
				nodeName := nodesList[rand.Intn(len(nodesList))]
				node := config.ConsensusNodeInfo(nodeName)
				if node.Name == "" {
					errCh <- errorx.IllegalState.New("bot %d: node %s not found in config", botId, nodeName)
					return
				}

				traceId := fmt.Sprintf("tx-crypto-%d", time.Now().UnixNano())

				logx.As().Info().Int("bot_id", botId).
					Any("node", node).
					Str("mirror_node", mirror).
					Int64("tx_total", counter).
					Str("interval", tickerDuration.String()).
					Str("duration", duration).
					Str("trace_id", traceId).
					Msgf("Transferring %d hbar from %v to %s", 1, client.GetOperatorAccountID(), node.Account)

				if ex := sendCryptoTransaction(botId, node.Account, 1, traceId, txReceipts); ex != nil {
					errCh <- errorx.IllegalState.New("bot %d: failed to send transaction: %v", botId, ex)
					return
				}

				counter++
			case <-timer.C:
				return
			case <-ctx.Done():
				return
			}
		}
	}

	for i := 0; i < totalBots; i++ {
		wg.Add(1)
		go botFunc(i)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for e := range errCh {
		if e != nil {
			errs = append(errs, e)
		}
	}
	if len(errs) > 0 {
		return errorx.IllegalState.New("one or more bots failed, errors: %v", errs)
	}
	return nil
}

func sendCryptoTransaction(botId int, toAccount string, amount float64, traceId string, txReceipts chan int) error {
	to, err := hiero.AccountIDFromString(toAccount)
	if err != nil {
		return errorx.IllegalArgument.Wrap(err, fmt.Sprintf("error converting string to AccountID: %s", toAccount))
	}

	transactionResponse, err := hiero.NewTransferTransaction().
		AddHbarTransfer(client.GetOperatorAccountID(), hiero.NewHbar(-1*amount)).
		AddHbarTransfer(to, hiero.NewHbar(amount)).
		SetTransactionMemo(fmt.Sprintf("hammer - bot %d", botId)).
		Execute(client)
	if err != nil {
		return errorx.IllegalState.Wrap(err, "error creating transaction")
	}

	transactionReceipt, err := transactionResponse.GetReceipt(client)
	if err != nil {
		return errorx.IllegalState.Wrap(err, "error retrieving transfer receipt")
	}

	logx.As().Info().
		Int("bot_id", botId).
		Str("to", to.String()).
		Str("from", client.GetOperatorAccountID().String()).
		Float64("amount", amount).
		Str("status", transactionReceipt.Status.String()).
		Str("transaction_id", transactionResponse.TransactionID.String()).
		Str("trace_id", traceId).
		Msgf("Crypto transfer status: %v", transactionReceipt.Status)

	// add tx receipt count for TPS calculation
	txReceipts <- 1

	return nil
}
