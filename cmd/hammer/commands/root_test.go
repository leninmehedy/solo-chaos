package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCmdFlags(t *testing.T) {
	cmd := rootCmd

	// Check default values
	assert.Equal(t, "node1,node2,node3", cmd.PersistentFlags().Lookup("nodes").Value.String())
	assert.Equal(t, "crypto", cmd.PersistentFlags().Lookup("tx-type").Value.String())
	assert.Equal(t, "60s", cmd.PersistentFlags().Lookup("duration").Value.String())
	assert.Equal(t, "100", cmd.PersistentFlags().Lookup("bots").Value.String())
	assert.Equal(t, "1", cmd.PersistentFlags().Lookup("tps").Value.String())

	// Check required flag
	flag := cmd.PersistentFlags().Lookup("config")
	assert.True(t, flag != nil)
}

func TestRootCmdHasTxSubcommand(t *testing.T) {
	cmd := rootCmd
	found := false
	for _, c := range cmd.Commands() {
		if c == txCmd {
			found = true
			break
		}
	}
	assert.True(t, found, "txCmd should be registered as a subcommand")
}
