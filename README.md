# Gas Station

Automated gas token management for Cosmos accounts.

![Gas Station](assets/gasstation.png)

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

## Releases

Pre-built releases are available on the [Releases][4] page.

## Installation

```sh
go install github.com/shapeshed/gasstation/cmd/gasstation@latest
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
LOG_LEVEL=debug gasstation -c configs/config.toml
```

## Contributing

If you want to hack on gas station and contribute to the project you can get the
source and build the project as follows.

```sh
git clone https://github.com/shapeshed/gasstation.git
go mod tidy
make
```

The binary will be created at `./bin/gastation`.

[1]: https://prometheus.io/
[2]: https://github.com/QuokkaStake/cosmos-wallets-exporter
[3]: configs/config.toml.example
[4]: https://github.com/shapeshed/gasstation/releases
