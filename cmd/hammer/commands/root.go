package commands

import (
	"fmt"
	"github.com/automa-saga/logx"
	"github.com/leninmehedy/solo-chaos/cmd/hammer/config"
	"github.com/spf13/cobra"
	_ "net/http/pprof"
)

var (
	// Used for flags.
	flagConfig       string
	flagNodes        string
	flagTotalWorkers int
	flagTps          int
	flagDuration     string
	flagMirror       string
	flagTxType       string

	rootCmd = &cobra.Command{
		Use:   "hammer",
		Short: "A fast and efficient transaction load generator",
		Long:  "Hammer - A fast and efficient transaction load generator",
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&flagConfig, "config", "c", "", "config file path (required)")
	rootCmd.PersistentFlags().StringVarP(&flagNodes, "nodes", "n", "node1,node2,node3", "comma separated list of node names to connect")
	rootCmd.PersistentFlags().IntVarP(&flagTotalWorkers, "workers", "w", 1, "number of worker")
	rootCmd.PersistentFlags().IntVarP(&flagTps, "tps", "t", 10, "tps per worker")
	rootCmd.PersistentFlags().StringVarP(&flagDuration, "duration", "d", "60s", "duration of the test")
	rootCmd.PersistentFlags().StringVarP(&flagMirror, "mirror-node", "m", "", "mirror node name to connect")
	rootCmd.PersistentFlags().StringVarP(&flagTxType, "tx-type", "", "crypto", "transaction type (crypto, file, contract)")

	// make flags mandatory
	_ = rootCmd.MarkPersistentFlagRequired("config")

	rootCmd.AddCommand(txCmd)
}

func initConfig() {
	var err error
	err = config.Initialize(flagConfig)
	if err != nil {
		fmt.Println("failed to initialize config")
		cobra.CheckErr(err)
	}

	err = logx.Initialize(config.Get().Log)
	if err != nil {
		fmt.Println(err)
		cobra.CheckErr(err)
	}
}
