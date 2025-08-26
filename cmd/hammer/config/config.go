package config

import (
	"fmt"
	"github.com/automa-saga/logx"
	hiero "github.com/hiero-ledger/hiero-sdk-go/v2/sdk"
	"github.com/joomcode/errorx"
	"github.com/spf13/viper"
	"os"
	"strings"
)

var cfg *Config
var network map[string]hiero.AccountID
var consensusNodes map[string]ConsensusNode
var mirrorNodes map[string]MirrorNode

type ConsensusNode struct {
	Name     string `yaml:"name"`
	Account  string `yaml:"account"`
	Endpoint string `yaml:"endpoint"`
}

type MirrorNode struct {
	Name     string `yaml:"name"`
	Endpoint string `yaml:"endpoint"`
}

type Operator struct {
	Account string `yaml:"account"`
	Key     string `yaml:"key"`
}

type Config struct {
	Log            logx.LoggingConfig
	ConsensusNodes []ConsensusNode `yaml:"consensusNodes"`
	MirrorNodes    []MirrorNode    `yaml:"mirrorNodes"`
	Operator       Operator        `yaml:"operator"`
}

func init() {
	cfg = &Config{
		Log: logx.LoggingConfig{
			Level:          "debug",
			FileLogging:    false,
			ConsoleLogging: true,
		},
	}
}

func Initialize(path string) error {
	viper.Reset()
	viper.SetConfigFile(path)
	viper.SetEnvPrefix("cheetah")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	if network == nil {
		network = make(map[string]hiero.AccountID)
	}

	if consensusNodes == nil {
		consensusNodes = make(map[string]ConsensusNode)
	}

	if mirrorNodes == nil {
		mirrorNodes = make(map[string]MirrorNode)
	}

	for _, n := range cfg.ConsensusNodes {
		consensusNodes[n.Name] = n

		logx.As().Debug().Str("name", n.Name).Str("account", n.Account).Str("endpoint", n.Endpoint).
			Msg("Parsing Consensus node info")

		accountId, err := hiero.AccountIDFromString(n.Account)
		if err != nil {
			return errorx.IllegalFormat.Wrap(err, "illegal account Id")
		}

		network[n.Endpoint] = accountId
	}

	for _, m := range cfg.MirrorNodes {
		logx.As().Debug().Str("name", m.Name).Str("endpoint", m.Endpoint).Msg("Parsing Mirror node info")
		mirrorNodes[m.Name] = m
	}

	if cfg.Operator.Account == "" && os.Getenv("OPERATOR_ID") != "" {
		cfg.Operator.Account = os.Getenv("OPERATOR_ID")
	}

	if cfg.Operator.Key == "" && os.Getenv("OPERATOR_KEY") != "" {
		cfg.Operator.Key = os.Getenv("OPERATOR_KEY")
	}

	return nil
}

func Get() *Config {
	return cfg
}

func Network() map[string]hiero.AccountID {
	if network == nil {
		network = make(map[string]hiero.AccountID)
	}

	return network
}

func ConsensusNodeInfo(name string) ConsensusNode {
	if c, ok := consensusNodes[name]; ok {
		return c
	}

	return ConsensusNode{}
}

func MirrorNodeInfo(name string) MirrorNode {
	if m, ok := mirrorNodes[name]; ok {
		return m
	}

	return MirrorNode{}
}
