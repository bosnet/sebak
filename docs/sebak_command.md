To install `sebak` command, please check the [installation guide](./sebak_Installation.md).

# Options
```
$ sebak
Usage:
  sebak [flags]
  sebak [command]

Available Commands:
  genesis     initialize new network
  help        Help about any command
  key         Keypair management
  node        Run sekbak node
  tls         Generate tls certificate and key file
  version     Print the version
  wallet      CLI for wallet management

Flags:
  -h, --help   help for sebak

Use "sebak [command] --help" for more information about a command.
```

# Commands
## Generate Keypair

With `key generate` you can generate new keypair.

```
$ sebak key generate
       Secret Seed: SCJNVYWXJX4COHWJRPSYXJU6FOI2ZJGDIBDTFHUMKQL6ZNGXOAGK2IZT
    Public Address: GDANB4X55ZATEO7DORBYNLRPQK2REHHGXPFDFI6KDEKZECR2NGEKDR5C
```

## `sebak genesis`

It will create new genesis account and common budget account and it's block.

At first, you should generate new keypairs for genesis account and common budget account.
```
$ sebak key generate
       Secret Seed: SCAVYSGE44WRBXF3U3VU76ARHXFUGZPHXORLAH4TCFXJVACETT3PHDJU
    Public Address: GDUGDDCHYCVOSHG6LI62OP7VQVBW65PMPKKLYL272RBO6QIYDTQTR6HN
$ sebak key generate
       Secret Seed: SD673RWCXYOQ5FPJCTUH73N5YNMXHSCKYDFCNFTRLY7QWDT7QJE2YPMH
    Public Address: GCDGWDPPDOCZ2V3SC7GZ6GWVYPFRL2X64IVZDTXN4PWKEHJ377JX23GV
```
and then, with new public address for genesis account, `GDUGDDCHYCVOSHG6LI62OP7VQVBW65PMPKKLYL272RBO6QIYDTQTR6HN` and new public address for common budget account, `GCDGWDPPDOCZ2V3SC7GZ6GWVYPFRL2X64IVZDTXN4PWKEHJ377JX23GV` you can generate genesis block.
```
$ sebak genesis GDUGDDCHYCVOSHG6LI62OP7VQVBW65PMPKKLYL272RBO6QIYDTQTR6HN GCDGWDPPDOCZ2V3SC7GZ6GWVYPFRL2X64IVZDTXN4PWKEHJ377JX23GV --network-id 'this-is-test-sebak-network'
```
By default, the balance of genesis account will be `1,000,000,000,000`(a hundred billion), of course you can change it with `--balance` option.

```
$ sebak genesis GDUGDDCHYCVOSHG6LI62OP7VQVBW65PMPKKLYL272RBO6QIYDTQTR6HN GCDGWDPPDOCZ2V3SC7GZ6GWVYPFRL2X64IVZDTXN4PWKEHJ377JX23GV --network-id 'this-is-test-sebak-network' --balance 1,000,000,000,000.0000000
```

## `sebak tls`
SEBAK use HTTP2 protocol, so it needs SSL certificates. This command will generate self-signed SSL certificate.

> You could also set your own SSL certificates manually.

```
$ sebak tls -h
Generate tls certificate and key file

Usage:
  ./sebak tls [flags]

Flags:
      --cert string     tls certificate file name (default "sebak.crt")
  -h, --help            help for tls
      --key string      tls key file name (default "sebak.key")
      --output string   tls output path (default ".")
```

## `sebak node`

This command will run the node and will join the network.

```
$ sebak node -h
Run sebak node

Usage:
  sebak node [flags]

Flags:
      --bind string                  bind to listen on (default "https://0.0.0.0:12345")
      --block-time string            block creation time (default "5")
      --debug-pprof                  set debug pprof
      --genesis string               performs the 'genesis' command before running node. Syntax: key[,balance]
  -h, --help                         help for node
      --log string                   set log file
      --log-format string            log format, {terminal, json} (default "terminal")
      --log-level string             log level, {crit, error, warn, info, debug} (default "info")
      --network-id string            network id
      --operations-limit string      operations limit in a transaction (default "1000")
      --publish string               endpoint url for other nodes
      --rate-limit-api list          rate limit for /api: [<ip>=]<limit>-<period>, ex) '10-S' '3.3.3.3=1000-M'
      --rate-limit-node list         rate limit for /node: [<ip>=]<limit>-<period>, ex) '10-S' '3.3.3.3=1000-M'
      --secret-seed string           secret seed of this node
      --storage string               storage uri (default "file:///Users/soonkuk/goworkspace/src/boscoin.io/sebak/db")
      --sync-check-interval string   sync check interval (default "30s")
      --sync-fetch-timeout string    sync fetch timeout (default "1m")
      --sync-pool-size string        sync pool size (default "300")
      --sync-retry-interval string   sync retry interval (default "10s")
      --threshold string             threshold (default "67")
      --timeout-accept string        timeout of the accept state (default "2")
      --timeout-init string          timeout of the init state (default "2")
      --timeout-sign string          timeout of the sign state (default "2")
      --tls-cert string              tls certificate file (default "sebak.crt")
      --tls-key string               tls key file (default "sebak.key")
      --transactions-limit string    transactions limit in a ballot (default "1000")
      --unfreezing-period string     how long freezing must last (default "241920")
      --validators string            set validator: <endpoint url>?address=<public address>[&alias=<alias>] [ <validator>...]
      --verbose                      verbose
```

This is basic example.
```
$ sebak node \
    --network-id 'this-is-test-sebak-network' \
    --bind "https://localhost:12345" \
    --tls-key 'sebak.key' \
    --tls-cert 'sebak.crt' \
    --secret-seed SBA27UP3J5L62KUAMTGXRUTOVMEOE7JCLNU3WVWBERX3V5TCY6DWWEWM \
    --validators "https://localhost:12346?address=GCLIHQDMLWU57FWY7RXHU5ZLMK6K6MMKLXGBYDSUEVA3XFQSZVPDU5YA https://localhost:12347?address=GB546SI45FATUDLT7XHPEDAUDY7YTDIDIC4D3D3LGBFV6DJRAGIDGSCZ"
```

This node will be accessible through `https://localhost:12345` and node have 2 another nodes(https://localhost:12346 and https://localhost:12347) as validator.

## `sebak wallet`

This command allows to interact with the network through the CLI interface.
Currently, payment and unfreezeRequest are implemented, with account creation being an option of payment:
```
Usage:
  ./sebak wallet payment <receiver pubkey> <amount> <sender secret seed> [flags]

Flags:
      --create              Whether or not the account should be created
      --dry-run             Print the transaction instead of sending it
      --endpoint string     endpoint to send the transaction to (https / memory address)
      --freeze              When present, the payment is a frozen account creation. Imply --create.
  -h, --help                help for payment
      --network-id string   network id
      --verbose             Print extra data (transaction sent, before/after balance...)

usage:
  ./sebak wallet unfreezeRequest <sender secret seed> [flags]

Flags:
      --dry-run             Print the transaction instead of sending it
      --endpoint string     endpoint to send the transaction to (https / memory address)
  -h, --help                help for unfreezeRequest
      --network-id string   network id
      --verbose             Print extra data (transaction sent)
```

## `sebak version`

This command for print current version.
```
$ sebak version
0.1.0+proto
```