package commands

import (
	"context"
	"fmt"
	"github.com/automa-saga/logx"
	hiero "github.com/hiero-ledger/hiero-sdk-go/v2/sdk"
	"github.com/joomcode/errorx"
	"github.com/leninmehedy/solo-chaos/cmd/hammer/config"
	"github.com/spf13/cobra"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const TxTypeCrypto = "crypto"

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

	// Initialize configuration
	if err := config.Initialize(flagConfig); err != nil {
		logx.As().Fatal().Err(err).Msg("Failed to initialize config")
	}
	logx.As().Info().Msg("Configuration initialized")
	// Here you can add the logic to create and send transactions based on the flags
	logx.As().Info().
		Str("nodes", flagNodes).
		Int("worker", flagTotalWorkers).
		Int("tps", flagTps).
		Str("duration", flagDuration).
		Str("mirror-node", flagMirror).
		Str("tx-type", flagTxType).
		Msg("Starting transaction load generation")

	// setup client
	err := setupClient()
	if err != nil {
		logx.As().Fatal().Err(err).Msg("Failed to setup client")
	}

	var wg sync.WaitGroup
	// based on the transaction type, create different types of transactions
	switch flagTxType {
	case TxTypeCrypto:
		wg.Add(1)
		err := startCryptoTxWorkers(ctx, flagNodes, flagTotalWorkers, flagTps, flagDuration, flagMirror)
		wg.Done()
		if err != nil {
			logx.As().Error().Err(err).Msg("Failed to create crypto transactions")
			os.Exit(1)
		}
	default:
		logx.As().Error().Str("tx-type", flagTxType).Msg("Unsupported transaction type")
		os.Exit(1)
	}

	go func() {
		wg.Wait()
		logx.As().Info().Msg("All transaction bots completed")
		cancelFunc()
	}()

	// Handle OS signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
		logx.As().Trace().Msg("Received exit signal, stopping pipelines...")
		cancelFunc()
	case <-ctx.Done():
	}
}

func setupClient() error {
	if client != nil {
		return nil
	}

	var err error
	client, err = hiero.ClientForNetworkV2(config.Network())
	if err != nil {
		return errorx.IllegalState.Wrap(err, "error creating client")
	}

	// operatorAccountID is any account that has some hbar to pay for the transaction fees and we know the private key
	operatorAccountID, err := hiero.AccountIDFromString(config.Get().Operator.Account)
	if err != nil {
		return errorx.IllegalState.Wrap(err, "error converting string to operator AccountID")
	}

	operatorKey, err := hiero.PrivateKeyFromString(config.Get().Operator.Key)
	if err != nil {
		return errorx.IllegalState.Wrap(err, "error converting string to operator key")
	}

	// Setting the client operator ID and key
	client.SetOperator(operatorAccountID, operatorKey)

	return nil
}

func startCryptoTxWorkers(ctx context.Context, nodes string, totalWorkers int, tps int, duration string, mirror string) error {
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
		Int("total_workers", totalWorkers).
		Int("total_tps", tps*totalWorkers).
		Strs("nodes", nodesList).
		Str("mirror_node", mirror).
		Msg("Starting crypto transaction bots")

	tickerDuration := time.Second / time.Duration(tps)
	errCh := make(chan error, totalWorkers) // Buffered channel to avoid blocking
	var wg sync.WaitGroup

	workerFunc := func(workerId int) {
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
					errCh <- errorx.IllegalState.New("totalWorkers %d: node %s not found in config", workerId, nodeName)
					return
				}

				logx.As().Info().Int("bot_id", workerId).
					Any("node", node).
					Str("mirror_node", mirror).
					Int64("tx_total", counter).
					Str("interval", tickerDuration.String()).
					Str("duration", duration).
					Any("node", node).
					Msgf("Transferring %d hbar from %v to %s", 1, client.GetOperatorAccountID(), node.Account)
				if ex := sendCryptoTransaction(workerId, node.Account, 1); ex != nil {
					errCh <- errorx.IllegalState.New("totalWorkers %d: failed to send transaction to any node", workerId)
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

	// Start totalWorkers pool
	numWorkers := totalWorkers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go workerFunc(i)
	}

	// Wait for all workers to finish
	wg.Wait()
	close(errCh)

	// Collect errors
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

func sendCryptoTransaction(botId int, toAccount string, amount float64) error {
	to, err := hiero.AccountIDFromString(toAccount)
	if err != nil {
		return errorx.IllegalArgument.Wrap(err, fmt.Sprintf("error converting string to AccountID: %s", toAccount))
	}

	transactionResponse, err := hiero.NewTransferTransaction().
		// Hbar has to be negated to denote we are taking out from that account
		AddHbarTransfer(client.GetOperatorAccountID(), hiero.NewHbar(-1*amount)).
		// If the amount of these 2 transfers is not the same, the transaction will throw an error
		AddHbarTransfer(to, hiero.NewHbar(amount)).
		SetTransactionMemo(fmt.Sprintf("hammer - worker %d", botId)).
		Execute(client)

	if err != nil {
		return errorx.IllegalState.Wrap(err, "error creating transaction")
	}

	// Retrieve the receipt to make sure the transaction went through
	transactionReceipt, err := transactionResponse.GetReceipt(client)

	if err != nil {
		return errorx.IllegalState.Wrap(err, "error retrieving transfer receipt")
	}

	logx.As().Info().
		Int("worker", botId).
		Str("to", to.String()).
		Str("from", client.GetOperatorAccountID().String()).
		Msgf("Crypto transfer status: %v\n", transactionReceipt.Status)
	return nil
}
