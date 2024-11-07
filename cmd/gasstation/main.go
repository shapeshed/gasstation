package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/shapeshed/cosmosign"
	"github.com/shapeshed/gasstation/internal/grpc"
	"github.com/shapeshed/gasstation/internal/logger"
	"github.com/shapeshed/gasstation/internal/queries"
	"go.uber.org/zap"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// Chain represents the configuration for each chain in the TOML file.
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
	Frequency      int      `toml:"frequency"`
}

// Config represents the entire configuration file.
type Config struct {
	Chains []Chain `toml:"chains"`
}

// ChainService represents a balance-checking service for a single chain.
type ChainService struct {
	accountQueryClient authtypes.QueryClient
	chain              Chain
	keyring            keyring.Keyring
	bankQueryClient    banktypes.QueryClient
	cosmosignClient    *cosmosign.Cosmosign
	ticker             *time.Ticker
	rand               *rand.Rand
	logger             *zap.Logger
}

var (
	// version and buildDate is set with -ldflags in the Makefile
	Version     string
	BuildDate   string
	configPath  *string
	showVersion *bool
)

func parseFlags() {
	configPath = flag.String("c", "config.toml", "path to config file")
	showVersion = flag.Bool("v", false, "Print the version of the program")
	flag.Parse()
}

// NewChainService initializes a new ChainService with a gRPC client, timer, and Cosmosign client.
func NewChainService(chain Chain, interval time.Duration, l *zap.Logger) (*ChainService, error) {
	ctx := context.Background()

	// Initialize a GRPC connection
	conn, err := grpc.SetupGRPCConnection(chain.GRPCURL, false)
	if err != nil {
		return nil, err
	}

	// Create a bank query client
	bankQueryClient := banktypes.NewQueryClient(conn)

	// Fetch and set the account prefix
	accountQueryClient := authtypes.NewQueryClient(conn)
	prefix, err := accountQueryClient.Bech32Prefix(ctx, &authtypes.Bech32PrefixRequest{})
	if err != nil {
		return nil, err
	}
	chain.AddressPrefix = prefix.Bech32Prefix

	// Initialize the keyring
	encodingConfig := testutil.MakeTestEncodingConfig()
	chainKeyring, err := keyring.New(chain.KeyringAppName, chain.KeyringBackend, chain.KeyringRootDir, nil, encodingConfig.Codec)
	if err != nil {
		return nil, err
	}

	// Initialize a cosmosign client
	cosmosignClient, err := cosmosign.NewClient(context.Background(),
		cosmosign.WithGRPCConn(conn),
		cosmosign.WithGasMultipler(chain.GasMultipler),
		cosmosign.WithGasPrices(chain.GasPrices),
		cosmosign.WithKeyringBackend(chain.KeyringBackend),
		cosmosign.WithKeyringAppName(chain.KeyringAppName),
		cosmosign.WithKeyringRootDir(chain.KeyringRootDir),
		cosmosign.WithKeyringUID(chain.KeyringUID),
	)
	if err != nil {
		return nil, err
	}

	// Return the chain service
	return &ChainService{
		chain:              chain,
		accountQueryClient: accountQueryClient,
		bankQueryClient:    bankQueryClient,
		cosmosignClient:    cosmosignClient,
		keyring:            chainKeyring,
		ticker:             time.NewTicker(interval),
		rand:               rand.New(rand.NewSource(time.Now().UnixNano())),
		logger:             l,
	}, nil
}

// Run starts the periodic balance check for the ChainService, with a randomized delay at the start.
func (cs *ChainService) Run(ctx context.Context) {
	randomDelay := time.Duration(cs.rand.Intn(cs.chain.Frequency)) * time.Second
	cs.logger.Info("Randomized start delay", zap.String("chain", cs.chain.Name), zap.Duration("delay", randomDelay))
	time.Sleep(randomDelay)

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

// checkBalances checks the balance for each account and sends a bank transaction if below the threshold.
func (cs *ChainService) checkBalances(ctx context.Context) {
	for _, account := range cs.chain.Accounts {
		balance, err := queries.GetBalance(ctx, cs.bankQueryClient, account, cs.chain.GasDenom)
		if err != nil {
			cs.logger.Error("Error retrieving balance", zap.String("chain", cs.chain.Name), zap.String("account", account), zap.Error(err))
			continue
		}
		cs.logger.Info("balance", zap.String("chain", cs.chain.Name), zap.String("account", account), zap.Any("balance", balance))

		threshold := sdkmath.NewInt(cs.chain.Threshold)
		if balance.Balance.Amount.LT(threshold) {
			cs.logger.Info("Balance is less than threshold, funding required", zap.String("chain", cs.chain.Name), zap.String("account", account))

			signer, err := cs.keyring.Key(cs.chain.KeyringUID)
			if err != nil {
				cs.logger.Error("Error getting signer", zap.String("chain", cs.chain.Name), zap.String("account", account), zap.Error(err))
				continue
			}

			pk, _ := signer.GetPubKey()
			addressBytes := sdktypes.AccAddress(pk.Address().Bytes())
			signerAddress, _ := sdktypes.Bech32ifyAddressBytes(cs.chain.AddressPrefix, addressBytes)
			cs.logger.Info("address", zap.String("signerAddress", signerAddress), zap.String("account", account), zap.String("prefix", cs.chain.AddressPrefix))

			msg := &banktypes.MsgSend{
				FromAddress: signerAddress,
				ToAddress:   account,
				Amount:      sdktypes.NewCoins(sdktypes.NewCoin(cs.chain.GasDenom, sdkmath.NewInt(cs.chain.AmountToFund))),
			}

			cs.logger.Info("msg", zap.Any("msg", msg))

			cs.logger.Info("cosmosign", zap.Any("cosmosign", cs.cosmosignClient))

			// Send the message using Cosmosign
			res, err := cs.cosmosignClient.SendMessages(msg)
			switch {
			case err != nil:
				cs.logger.Error("Failed to send transaction", zap.Error(err))
			case res.TxResponse.Code == 0:
				cs.logger.Info("Transaction successful", zap.String("transaction hash", res.TxResponse.TxHash), zap.String("chain", cs.chain.Name), zap.String("account", account))
			default:
				cs.logger.Info("Transaction failed", zap.Uint32("code", res.TxResponse.Code), zap.String("raw_log", res.TxResponse.RawLog))
			}
		} else {
			cs.logger.Info("Balance ok", zap.String("chain", cs.chain.Name), zap.String("account", account), zap.String("balance", balance.Balance.Amount.String()))
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
	parseFlags()
	if *showVersion {
		fmt.Printf("Version: %s\nBuild Date: %s\n", Version, BuildDate)
		os.Exit(0)
	}

	// Initialize zap logger
	l, err := logger.Setup()
	if err != nil {
		log.Fatalf("Failed to initialize zap logger: %v", err)
	}

	// Load configuration from TOML
	config, err := loadConfig(*configPath)
	if err != nil {
		l.Error("Failed to load configuration", zap.Error(err))
	}

	ctx := context.Background()
	l.Debug("Starting Gas Station", zap.Any("config", config))

	// Use WaitGroup to manage concurrent startup of ChainServices
	var wg sync.WaitGroup

	// Initialize and run a ChainService for each chain configuration
	for _, chain := range config.Chains {
		wg.Add(1) // Add a goroutine to the WaitGroup

		// Start each ChainService in its own goroutine
		go func(chain Chain) {
			defer wg.Done() // Signal WaitGroup when this goroutine is complete

			// Initialize ChainService
			chainService, err := NewChainService(chain, time.Duration(chain.Frequency)*time.Second, l)
			if err != nil {
				l.Error("Error initializing ChainService", zap.String("chain", chain.Name), zap.Error(err))
				return
			}

			// Run the service with context
			chainService.Run(ctx)
		}(chain)
	}

	// Wait for all goroutines to start
	wg.Wait()

	// Keep the main function running
	select {}
}
