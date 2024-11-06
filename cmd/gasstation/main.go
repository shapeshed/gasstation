package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/BurntSushi/toml"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/shapeshed/gasstation/internal/grpc"
	"github.com/shapeshed/gasstation/internal/queries"
)

// ChainConfig represents the configuration for each chain in the TOML file.
type ChainConfig struct {
	Name     string   `toml:"name"`
	GRPCURL  string   `toml:"grpc_url"`
	Accounts []string `toml:"accounts"`
}

// Config represents the entire configuration file.
type Config struct {
	Chains []ChainConfig `toml:"chains"`
}

func loadConfig(path string) (*Config, error) {
	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func main() {
	// Load configuration from TOML
	config, err := loadConfig("config.toml")
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx := context.Background()

	// Loop through each chain configuration
	for _, chain := range config.Chains {
		slog.Info("Checking balances for chain", "chain", chain.Name)

		// Initialize a GRPC Client for the chain
		conn, err := grpc.SetupGRPCConnection(chain.GRPCURL, false)
		if err != nil {
			slog.Error("Error initializing gRPC connection", "chain", chain.Name, "error", err)
			continue
		}
		defer conn.Close()

		bankQueryClient := banktypes.NewQueryClient(conn)

		// Check balance for each account in the chain
		for _, account := range chain.Accounts {
			balance, err := queries.GetBalance(ctx, bankQueryClient, account, "untrn")
			if err != nil {
				slog.Error("Error retrieving balance", "chain", chain.Name, "account", account, "error", err)
				continue
			}

			slog.Info("Balance checked", "chain", chain.Name, "account", account, "balance", balance)
		}
	}
}
