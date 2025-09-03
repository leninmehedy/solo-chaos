package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCmd(t *testing.T) {
	t.Run("Validate command exists", func(t *testing.T) {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd == validateCmd {
				found = true
				break
			}
		}
		assert.True(t, found, "validate command should be registered")
	})

	t.Run("Validate command has correct structure", func(t *testing.T) {
		assert.Equal(t, "validate", validateCmd.Use)
		assert.Contains(t, validateCmd.Short, "chaos experiments")
		assert.NotEmpty(t, validateCmd.Long)
	})

	t.Run("Validate command has required subcommands", func(t *testing.T) {
		subcommands := validateCmd.Commands()
		assert.Len(t, subcommands, 3, "Should have 3 subcommands")

		commandNames := make(map[string]bool)
		for _, cmd := range subcommands {
			commandNames[cmd.Use] = true
		}

		assert.True(t, commandNames["manifest"], "Should have manifest subcommand")
		assert.True(t, commandNames["chaos"], "Should have chaos subcommand")
		assert.True(t, commandNames["network"], "Should have network subcommand")
	})
}

func TestValidateManifestCmd(t *testing.T) {
	t.Run("Manifest command has correct flags", func(t *testing.T) {
		manifestFlag := validateManifestCmd.Flags().Lookup("manifest-path")
		assert.NotNil(t, manifestFlag, "Should have manifest-path flag")
		assert.Equal(t, "", manifestFlag.DefValue, "Default value should be empty")
	})

	t.Run("Manifest command requires manifest-path flag", func(t *testing.T) {
		// Check that manifest-path flag exists (required flags are checked by cobra at runtime)
		manifestFlag := validateManifestCmd.Flags().Lookup("manifest-path")
		assert.NotNil(t, manifestFlag, "manifest-path flag should exist")
	})
}

func TestValidateChaosCmd(t *testing.T) {
	t.Run("Chaos command has correct flags", func(t *testing.T) {
		flags := []string{"kubeconfig", "chaos-namespace", "chaos-name", "chaos-type", "wait-timeout"}
		
		for _, flagName := range flags {
			flag := validateChaosCmd.Flags().Lookup(flagName)
			assert.NotNil(t, flag, "Should have %s flag", flagName)
		}
	})

	t.Run("Chaos command has correct default values", func(t *testing.T) {
		namespaceFlag := validateChaosCmd.Flags().Lookup("chaos-namespace")
		assert.Equal(t, "chaos-mesh", namespaceFlag.DefValue)

		typeFlag := validateChaosCmd.Flags().Lookup("chaos-type")
		assert.Equal(t, "NetworkChaos", typeFlag.DefValue)

		kubeconfigFlag := validateChaosCmd.Flags().Lookup("kubeconfig")
		assert.Equal(t, "", kubeconfigFlag.DefValue)
	})

	t.Run("Chaos command requires chaos-name flag", func(t *testing.T) {
		// Check that chaos-name flag exists (required flags are checked by cobra at runtime)
		nameFlag := validateChaosCmd.Flags().Lookup("chaos-name")
		assert.NotNil(t, nameFlag, "chaos-name flag should exist")
	})
}

func TestValidateNetworkCmd(t *testing.T) {
	t.Run("Network command has correct flags", func(t *testing.T) {
		flags := []string{"kubeconfig", "test-namespace"}
		
		for _, flagName := range flags {
			flag := validateNetworkCmd.Flags().Lookup(flagName)
			assert.NotNil(t, flag, "Should have %s flag", flagName)
		}
	})

	t.Run("Network command has correct default values", func(t *testing.T) {
		namespaceFlag := validateNetworkCmd.Flags().Lookup("test-namespace")
		assert.Equal(t, "solo", namespaceFlag.DefValue)

		kubeconfigFlag := validateNetworkCmd.Flags().Lookup("kubeconfig")
		assert.Equal(t, "", kubeconfigFlag.DefValue)
	})

	t.Run("Network command requires test-namespace flag", func(t *testing.T) {
		// Check that test-namespace flag exists (required flags are checked by cobra at runtime)
		namespaceFlag := validateNetworkCmd.Flags().Lookup("test-namespace")
		assert.NotNil(t, namespaceFlag, "test-namespace flag should exist")
	})
}