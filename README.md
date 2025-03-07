# Gas Station

Automated gas token management for Cosmos accounts.

![Gas Station][6]

## Reference deployment

In this example a treasury wants to fund multiple bot accounts.

- Treasury account funds Gas Station
- Gas Station checks the balance of accounts it knows about and tops up accounts
  if they fall below a threshold.
- [Prometheus][1] monitors Gas Stations balance via [cosmos-wallets-exporter][2]
  and triggers an alert when it falls below a threshold

![Architecture Diagram][5]

## Installation

Pre-built releases are available on the [Releases][4] page.

```sh
VERSION="v0.0.6"
ARCH="linux_amd64" 
wget -O /tmp/gasstation_${VERSION}_${ARCH}.tar.gz "https://github.com/shapeshed/gasstation/releases/download/${VERSION}/gasstation_${VERSION}_${ARCH}.tar.gz"
tar -xzf /tmp/gasstation_${VERSION}_${ARCH}.tar.gz -C /tmp
sudo mv /tmp/gasstation /usr/local/bin/gasstation
gasstation --version
```

You can also use `go install` if you prefer.

```sh
VERSION=$(git describe --tags --always --dirty --match=v\* 2> /dev/null || echo v0)
DATE=$(date +%FT%T%z)
go install -ldflags="-X main.Version=$VERSION -X main.BuildDate=$DATE" github.com/shapeshed/gasstation/cmd/gasstation@latest
gasstation --version
```

## Configuration

Gas Station accepts a configuration file on startup via the `-c` flag.

```sh
gasstation -c configs/config.toml
```

An example config file is provided in [`configs/config.toml.example`][3]

```toml
[[chains]]
name             = "neutron-testnet"
gas_denom        = "untrn"
gas_prices       = "0.0053untrn"
gas_multiplier   = 1.75
keyring_app_name = "neutron"
keyring_backend  = "pass"
keyring_root_dir = "/home/go"
keyring_uid      = "gasstation"
grpc_url         = "neutron-testnet-grpc.polkachu.com:19190"
accounts         = ["neutron1y3fzmdmlqrhxfjh570cdh74nve5e33apwl2j0t"]
threshold        = 1000000
amount_to_fund   = 5000000
frequency        = 60
```

| Field                | Description                                                                                                                                                  |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **name**             | A descriptive name for the chain (e.g., `neutron-testnet`). This helps identify the chain in logs and outputs.                                               |
| **gas_denom**        | The denomination used for gas fees on the chain (e.g., `untrn` for Neutron Testnet).                                                                         |
| **gas_prices**       | The gas price to use in transactions, specified as an amount and denomination (e.g., `0.0053untrn`).                                                         |
| **gas_multiplier**   | A multiplier applied to the gas estimate, providing a buffer to ensure transactions are not underfunded.                                                     |
| **keyring_app_name** | The application name for the keyring, used to manage account keys.                                                                                           |
| **keyring_backend**  | Specifies the keyring backend, which determines how account keys are stored and accessed. Common options are `pass` or `file`.                               |
| **keyring_root_dir** | The root directory where the keyring is stored. This is typically set to the home directory.                                                                 |
| **keyring_uid**      | The unique identifier for the key used to send transactions. This key must have sufficient funds to top up monitored accounts.                               |
| **grpc_url**         | The gRPC URL for interacting with the chain. This URL is used for querying account balances and submitting transactions.                                     |
| **accounts**         | A list of account addresses to monitor on this chain. Gas Tank will monitor each address and top it up when the balance falls below the specified threshold. |
| **threshold**        | The minimum balance threshold for each account. When an account's balance drops below this level, Gas Tank will send a specified amount to replenish it.     |
| **amount_to_fund**   | The amount to send when topping up an account. The amount is specified in the chain's gas denomination (e.g., `1000000` units of `untrn`).                   |
| **frequency**        | The interval (in seconds) at which Gas Tank checks account balances.                                                                                         |

## Logging

Gas Station logs some information whilst running. Log levels can be increased
with the `LOG_LEVEL` environment variable.

```sh
LOG_LEVEL=debug gasstation -c ~/.config/gasstation/config.toml
```

## Rationale

Gas Tank was developed to address the challenge of running multiple Cosmos-based
bots and ensuring they consistently maintain sufficient funds for gas fees.

While it is possible to monitor account balances with tools like [Prometheus][1]
using [Cosmos Wallet Exporter][2] these tools only generate alerts when an
account balance falls below a specified threshold. In such a setup, a human
operator must manually respond to each alert and send a bank transaction to
replenish each account, making it a time-intensive process prone to delays and
error.

Gas Tank largely automates this process by continuously monitoring account
balances and automatically initiating a bank transaction when an account’s
balance falls below a designated threshold. This automation reduces the need for
manual intervention, helping to keep bots running smoothly and minimizing the
risk of them running out of gas.

> [!NOTE]\
> Ensure that the account used for issuing automated bank transactions is
> actively monitored.

## Contributing

Contributions to the project are welcome!

```sh
git clone https://github.com/shapeshed/gasstation.git
cd gasstation
go mod tidy
make
```

The binary will be created at `./bin/gastation`.

A git pre-commit file is available at `./scripts/pre-commit`.

```sh
cp ./scripts/pre-commit .git/hooks/
```

[1]: https://prometheus.io/
[2]: https://github.com/QuokkaStake/cosmos-wallets-exporter
[3]: configs/config.toml.example
[4]: https://github.com/shapeshed/gasstation/releases
[5]: assets/architecture.png "Reference deployment diagram"
[6]: assets/gasstation.png "A gas station on a lonely road in the desert"
