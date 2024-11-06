package main

import (
	"context"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/shapeshed/cosmosign"
	"github.com/shapeshed/gasstation/internal/grpc"
	"github.com/shapeshed/gasstation/internal/queries"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// ChainConfig represents the configuration for each chain in the TOML file.
type Chain struct {
	Name           string   `toml:"name"`
	AddressPrefix  string   `toml:"address_prefix"`
	GRPCURL        string   `toml:"grpc_url"`
	GasDenom       string   `toml:"gas_denom"`
	GasPrices      string   `toml:"gas_prices"`
	GasMultipler   float64  `toml:"gas_multiplier"`
	KeyringBackend string   `toml:"keyring_backend"`
	KeyringAppName string   `toml:"keyring_app_name"`
	KeyringRootDir string   `toml:"keyring_root_dir"`
	KeyringUID     string   `toml:"keyring_uid"`
	Accounts       []string `toml:"accounts"`
	Threshold      int64    `toml:"threshold"`
	AmountToFund   int64    `toml:"amount_to_fund"`
}

// Config represents the entire configuration file.
type Config struct {
	Chains []Chain `toml:"chains"`
}

// ChainService represents a balance-checking service for a single chain.
type ChainService struct {
	chain           Chain
	keyring         keyring.Keyring
	bankQueryClient banktypes.QueryClient
	cosmosignClient *cosmosign.Cosmosign
	ticker          *time.Ticker
	rand            *rand.Rand
}

// NewChainService initializes a new ChainService with a gRPC client, timer, and Cosmosign client.
func NewChainService(chain Chain, interval time.Duration) (*ChainService, error) {
	conn, err := grpc.SetupGRPCConnection(chain.GRPCURL, false)
	if err != nil {
		return nil, err
	}

	bankQueryClient := banktypes.NewQueryClient(conn)

	encodingConfig := testutil.MakeTestEncodingConfig()
	chainKeyring, err := keyring.New(chain.KeyringAppName, chain.KeyringBackend, chain.KeyringRootDir, nil, encodingConfig.Codec)
	if err != nil {
		return nil, err
	}

	slog.Info("chain", slog.Any("chain", chain))

	cosmosignClient, err := cosmosign.NewClient(context.Background(),
		cosmosign.WithGRPCConn(conn),
		cosmosign.WithGasMultipler(chain.GasMultipler),
		cosmosign.WithGasPrices(chain.GasPrices),
		cosmosign.WithKeyringBackend(chain.KeyringBackend),
		cosmosign.WithKeyringAppName(chain.KeyringAppName),
		cosmosign.WithKeyringRootDir(chain.KeyringRootDir),
		cosmosign.WithKeyringUID(chain.KeyringUID),
		cosmosign.WithAddressPrefix(chain.AddressPrefix),
	)
	if err != nil {
		return nil, err
	}

	return &ChainService{
		chain:           chain,
		bankQueryClient: bankQueryClient,
		cosmosignClient: cosmosignClient,
		keyring:         chainKeyring,
		ticker:          time.NewTicker(interval),
		rand:            rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

// Run starts the periodic balance check for the ChainService, with a randomized delay at the start.
func (cs *ChainService) Run(ctx context.Context) {
	// Generate a random delay up to 10 seconds
	randomDelay := time.Duration(cs.rand.Intn(10)) * time.Second
	slog.Info("Randomized start delay", "chain", cs.chain.Name, "delay", randomDelay)
	time.Sleep(randomDelay) // Initial random delay

	for {
		select {
		case <-cs.ticker.C:
			cs.checkBalances(ctx)
		case <-ctx.Done():
			cs.ticker.Stop()
			return
		}
	}
}

// checkBalances checks the balance for each account and sends a transaction if below the threshold.
func (cs *ChainService) checkBalances(ctx context.Context) {
	for _, account := range cs.chain.Accounts {
		balance, err := queries.GetBalance(ctx, cs.bankQueryClient, account, cs.chain.GasDenom)
		if err != nil {
			slog.Error("Error retrieving balance", "chain", cs.chain.Name, "account", account, "error", err)
			continue
		}
		slog.Info("balance", "balance", balance)

		threshold := sdkmath.NewInt(cs.chain.Threshold)
		if balance.Balance.Amount.LT(threshold) {
			slog.Info("Balance is less than threshold, funding required", "chain", cs.chain.Name, "account", account)

			_ = cs.cosmosignClient.ApplyOptions(
				cosmosign.WithAddressPrefix(cs.chain.AddressPrefix),
			)

			signer, err := cs.keyring.Key(cs.chain.KeyringUID)
			if err != nil {
				slog.Error("Error getting signer", "chain", cs.chain.Name, "account", account, "error", err)
				continue
			}

			signerAddr, err := signer.GetAddress()
			if err != nil {
				slog.Error("Error getting signer address", "chain", cs.chain.Name, "account", account, "error", err)
				continue
			}

			// Create MsgSend transaction message
			msg := banktypes.NewMsgSend(
				signerAddr,
				sdktypes.MustAccAddressFromBech32(account),
				sdktypes.NewCoins(sdktypes.NewCoin(cs.chain.GasDenom, sdkmath.NewInt(cs.chain.AmountToFund))),
			)

			// Send the message using Cosmosign
			res, err := cs.cosmosignClient.SendMessages(msg)
			switch {
			case err != nil:
				slog.Error("Failed to send transaction", "error", err)
			case res.TxResponse.Code == 0:
				slog.Info("Transaction successful", "transaction hash", res.TxResponse.TxHash)
			default:
				slog.Info("Transaction failed", "code", res.TxResponse.Code, "raw_log", res.TxResponse.RawLog)
			}
		} else {
			slog.Info("Balance is sufficient", "chain", cs.chain.Name, "account", account, "balance", balance.Balance.Amount)
		}
	}
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
	config, err := loadConfig("configs/config.toml")
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx := context.Background()

	// Initialize and run a ChainService for each chain configuration
	for _, chain := range config.Chains {
		chainService, err := NewChainService(chain, 10*time.Second) // Adjust the interval as needed
		if err != nil {
			slog.Error("Error initializing ChainService", "chain", chain.Name, "error", err)
			continue
		}

		// Run the service in a separate goroutine with a randomized start delay
		go chainService.Run(ctx)
	}

	// Block forever (or use `select {}`) to keep the main function running
	select {}
}
