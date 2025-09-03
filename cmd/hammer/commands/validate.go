package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/automa-saga/logx"
	"github.com/leninmehedy/solo-chaos/internal/chaos"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	flagKubeconfig      string
	flagChaosNamespace  string
	flagChaosName       string
	flagChaosType       string
	flagManifestPath    string
	flagTestNamespace   string
	flagWaitTimeout     string
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate chaos experiments and network effects",
	Long:  "Validate chaos experiments, manifest generation, and network effects verification",
}

var validateManifestCmd = &cobra.Command{
	Use:   "manifest",
	Short: "Validate chaos manifest generation",
	Long:  "Validate YAML manifest template generation and variable substitution",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.ParseFlags(args); err != nil {
			logx.As().Error().Err(err).Msg("Failed to parse flags")
			os.Exit(1)
		}
		runValidateManifest(cmd.Context())
	},
}

var validateChaosCmd = &cobra.Command{
	Use:   "chaos",
	Short: "Validate chaos experiment status",
	Long:  "Validate that a chaos experiment has been created and is running correctly",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.ParseFlags(args); err != nil {
			logx.As().Error().Err(err).Msg("Failed to parse flags")
			os.Exit(1)
		}
		runValidateChaos(cmd.Context())
	},
}

var validateNetworkCmd = &cobra.Command{
	Use:   "network",
	Short: "Validate network connectivity and effects",
	Long:  "Validate network connectivity between pods and verify chaos effects",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.ParseFlags(args); err != nil {
			logx.As().Error().Err(err).Msg("Failed to parse flags")
			os.Exit(1)
		}
		runValidateNetwork(cmd.Context())
	},
}

func runValidateManifest(ctx context.Context) {
	logx.As().Info().Str("manifestPath", flagManifestPath).Msg("Validating chaos manifest")

	if flagManifestPath == "" {
		logx.As().Fatal().Msg("Manifest path is required (--manifest-path)")
	}

	// Create test variables
	vars := chaos.ManifestVariables{
		UUID:      "validation-test-" + time.Now().Format("20060102-150405"),
		Namespace: "solo",
		NodeNames: "node1,node2,node3",
		Rate:      "1gbps",
		Limit:     "20971520",
		Buffer:    "102400",
		Region:    "us",
	}

	// Generate and validate manifest
	manifest, err := chaos.GenerateManifestFromTemplate(flagManifestPath, vars)
	if err != nil {
		logx.As().Fatal().Err(err).Msg("Failed to generate manifest from template")
	}

	logx.As().Info().
		Str("name", manifest.Metadata.Name).
		Str("namespace", manifest.Metadata.Namespace).
		Str("action", manifest.Spec.Action).
		Str("mode", manifest.Spec.Mode).
		Msg("✅ Manifest validation successful")

	// Validate pod selector
	err = chaos.ValidatePodSelector(manifest.Spec.Selector)
	if err != nil {
		logx.As().Fatal().Err(err).Msg("Pod selector validation failed")
	}

	logx.As().Info().
		Int("namespaces", len(manifest.Spec.Selector.Namespaces)).
		Int("labelSelectors", len(manifest.Spec.Selector.LabelSelectors)).
		Int("expressionSelectors", len(manifest.Spec.Selector.ExpressionSelectors)).
		Msg("✅ Pod selector validation successful")

	// Print summary
	logx.As().Info().Msg("🎉 Manifest validation completed successfully")
}

func runValidateChaos(ctx context.Context) {
	logx.As().Info().
		Str("name", flagChaosName).
		Str("namespace", flagChaosNamespace).
		Str("type", flagChaosType).
		Msg("Validating chaos experiment status")

	if flagChaosName == "" {
		logx.As().Fatal().Msg("Chaos experiment name is required (--chaos-name)")
	}

	if flagChaosType == "" {
		logx.As().Fatal().Msg("Chaos experiment type is required (--chaos-type)")
	}

	// Get Kubernetes config
	config, err := getKubeConfig()
	if err != nil {
		logx.As().Fatal().Err(err).Msg("Failed to get Kubernetes config")
	}

	// Create chaos validator
	validator, err := chaos.NewChaosValidator(config)
	if err != nil {
		logx.As().Fatal().Err(err).Msg("Failed to create chaos validator")
	}

	// Validate chaos resource creation
	status, err := validator.ValidateChaosResourceCreation(ctx, flagChaosName, flagChaosNamespace, flagChaosType)
	if err != nil {
		logx.As().Fatal().Err(err).Msg("Failed to validate chaos resource")
	}

	logx.As().Info().
		Str("name", status.Name).
		Str("namespace", status.Namespace).
		Str("phase", status.Phase).
		Str("message", status.Message).
		Time("startTime", status.StartTime).
		Msg("✅ Chaos experiment found")

	// Wait for chaos to be ready if specified
	if flagWaitTimeout != "" {
		timeout, err := time.ParseDuration(flagWaitTimeout)
		if err != nil {
			logx.As().Fatal().Err(err).Msg("Invalid wait timeout format")
		}

		logx.As().Info().Dur("timeout", timeout).Msg("Waiting for chaos experiment to be ready...")

		err = validator.WaitForChaosExperimentReady(ctx, flagChaosName, flagChaosNamespace, flagChaosType, timeout)
		if err != nil {
			logx.As().Fatal().Err(err).Msg("Chaos experiment failed to become ready")
		}

		logx.As().Info().Msg("✅ Chaos experiment is ready")
	}

	logx.As().Info().Msg("🎉 Chaos validation completed successfully")
}

func runValidateNetwork(ctx context.Context) {
	logx.As().Info().Str("namespace", flagTestNamespace).Msg("Validating network connectivity")

	if flagTestNamespace == "" {
		logx.As().Fatal().Msg("Test namespace is required (--test-namespace)")
	}

	// Get Kubernetes config
	config, err := getKubeConfig()
	if err != nil {
		logx.As().Fatal().Err(err).Msg("Failed to get Kubernetes config")
	}

	// Create chaos validator
	validator, err := chaos.NewChaosValidator(config)
	if err != nil {
		logx.As().Fatal().Err(err).Msg("Failed to create chaos validator")
	}

	// Example pod selector for network nodes
	selector := chaos.PodSelector{
		Namespaces: []string{flagTestNamespace},
		LabelSelectors: map[string]string{
			"solo.hedera.com/type": "network-node",
		},
	}

	// Validate pod selection
	result, err := validator.ValidatePodSelection(ctx, selector)
	if err != nil {
		logx.As().Fatal().Err(err).Msg("Failed to validate pod selection")
	}

	logx.As().Info().
		Int("totalPods", result.TotalPods).
		Int("matchedPods", len(result.MatchedPods)).
		Str("criteria", result.SelectionCriteria).
		Msg("✅ Pod selection validation completed")

	// Print matched pods
	for _, pod := range result.MatchedPods {
		logx.As().Info().
			Str("podName", pod.Name).
			Str("phase", string(pod.Status.Phase)).
			Str("podIP", pod.Status.PodIP).
			Interface("labels", pod.Labels).
			Msg("Matched pod")
	}

	logx.As().Info().Msg("🎉 Network validation completed successfully")
}

func getKubeConfig() (*rest.Config, error) {
	if flagKubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", flagKubeconfig)
	}

	// Try in-cluster config first
	if config, err := rest.InClusterConfig(); err == nil {
		return config, nil
	}

	// Fall back to default kubeconfig location
	kubeconfigPath := clientcmd.RecommendedHomeFile
	if kubeconfigPath == "" {
		return nil, fmt.Errorf("no kubeconfig found")
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

func init() {
	// Add validate subcommand to root
	rootCmd.AddCommand(validateCmd)

	// Add subcommands to validate
	validateCmd.AddCommand(validateManifestCmd)
	validateCmd.AddCommand(validateChaosCmd)
	validateCmd.AddCommand(validateNetworkCmd)

	// Manifest validation flags
	validateManifestCmd.Flags().StringVarP(&flagManifestPath, "manifest-path", "", "", "Path to chaos manifest template file")
	_ = validateManifestCmd.MarkFlagRequired("manifest-path")

	// Chaos validation flags
	validateChaosCmd.Flags().StringVar(&flagKubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	validateChaosCmd.Flags().StringVar(&flagChaosNamespace, "chaos-namespace", "chaos-mesh", "Namespace where chaos experiment is running")
	validateChaosCmd.Flags().StringVar(&flagChaosName, "chaos-name", "", "Name of the chaos experiment to validate")
	validateChaosCmd.Flags().StringVar(&flagChaosType, "chaos-type", "NetworkChaos", "Type of chaos experiment (NetworkChaos, PodChaos)")
	validateChaosCmd.Flags().StringVar(&flagWaitTimeout, "wait-timeout", "", "Time to wait for chaos to be ready (e.g. 2m, 30s)")
	_ = validateChaosCmd.MarkFlagRequired("chaos-name")

	// Network validation flags
	validateNetworkCmd.Flags().StringVar(&flagKubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	validateNetworkCmd.Flags().StringVar(&flagTestNamespace, "test-namespace", "solo", "Namespace to test network connectivity")
	_ = validateNetworkCmd.MarkFlagRequired("test-namespace")
}