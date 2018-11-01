By the nature of design, SEBAK should be deployed as composition of nodes and they should tightly connected for consensus. This standalone mode recommend only for testing and debugging. With standalone mode, you do not need to compose quorum and another server. It does not make consensus, just execute incoming messages including transactions, but it do the right work, such as inflation, collecting transaction fee, etc.

## Installation

Please follow the [Installation](./sebak_Installation.md). If you are so busy and you already have golang (1.11 or higher) environment, do like this,

```sh
$ git clone https://github.com/bosnet/sebak.git /tmp/sebak-standalone
$ cd /tmp/sebak-standalone
$ go build boscoin.io/sebak/cmd/sebak
total 31152
drwxr-xr-x  23 spikeekips  wheel       736 Oct  7 13:22 .
drwxrwxrwt  23 root        wheel       736 Oct  7 13:21 ..
...
-rwxr-xr-x   1 spikeekips  staff  15850308 Oct  7 13:22 sebak
...
```

You will get the executable `./sebak`. You can use it to deploy.

## Deploy

> For the detailed deployment instruction, please check [Deploy Network](./sebak_deployment).

For standalone mode, SEBAK already prepared the special command, `self`. You can simply give only `self` in `--validators` option. Then you will get the standalone mode of SEBAK.

1. Create keypairs for genesis block generation , common account and node.

```sh
$ sebak key generate # for genesis block
       Secret Seed: SCAFTDXNCI76JNGT3PNJDYPXZGLMUWX3XKFIM3T2PF5EDPNR2QMBTTWN
    Public Address: GBUQWH7YMPSCNS53CPKBTEFKJONNZNUZWBXQSO7SSPWDT7M34N7OLCD2
$ sebak key generate # for common account
       Secret Seed: SB7QADV4VJK4WWQULQME2SFR3KVJPGCHOO5KYWPZI6KVOXL3RQIS7WPO
    Public Address: GCURMNJVC6ICYJSV7C35FHTNJ3PAIY4RYUL6CPNST3MQVGHK6WWNK7LF
$ sebak key generate # for node
       Secret Seed: SDTZT7PHWQWNZYIQL6KNCVROKPP74LKHEKMNERHAXCLGBPLCOWJIANA6
    Public Address: GCEQ4M573WODOWCAVBKJBVLLT2MIQ6ZDUYCU6SXFGLKYKL5IWU6JTCBW
```

2. Initialize sebak with genesis block. You have to put network id, genesis public address and common account public address in cmd. 
```sh
$ sebak genesis \
    --network-id "this-is-sebak-standalone" \
    GBUQWH7YMPSCNS53CPKBTEFKJONNZNUZWBXQSO7SSPWDT7M34N7OLCD2 \
    GCURMNJVC6ICYJSV7C35FHTNJ3PAIY4RYUL6CPNST3MQVGHK6WWNK7LF
```

3. Deploy

```sh
$ sebak node \
    --network-id "this-is-sebak-standalone" \
    --bind "http://0.0.0.0:12345" \
    --log-level debug \
    --secret-seed SDTZT7PHWQWNZYIQL6KNCVROKPP74LKHEKMNERHAXCLGBPLCOWJIANA6 \
    --validators "self"
INFO[10-07|13:47:20] Starting Sebak                           module=main caller=run.go:281
DBUG[10-07|13:47:20] parsed flags:                            module=main
	network-id=this-is-sebak-standalone
	bind=https://0.0.0.0:12345
	publish=
	storage=file:///tmp/ss/db
	tls-cert=sebak.crt
	tls-key=sebak.key
	log-level=debug
	log-format=terminal
	log=
	threshold=67
	timeout-init=2
	timeout-sign=2
	timeout-accept=2
	block-time=5
	transactions-limit=1000
	validator#0="alias=GCEQ.WU6J address=GCEQ4M573WODOWCAVBKJBVLLT2MIQ6ZDUYCU6SXFGLKYKL5IWU6JTCBW endpoint=https://0.0.0.0:12345" caller=run.go:311
...
```

## Testing

For usage of SEBAK commands, plese check [`sebak` commands](SEBAK-Commands).

### Creating Account

```sh
$ sebak key generate # for new account
       Secret Seed: SAJFV7V75PEJ62RXE7HUEYZPDPTMDEBWE5RSGL36A5MRVMCEUBLGTPGB
    Public Address: GCQ3RMPRKH5OCFNFCUB33GIV3OSCJXSTIXTMZOSWXYZYJGANYDJ6NOA3
```

For Create account, you should put 1 public address for account create and a genesis block secret seed ( or secret key ). 

```sh
$ sebak wallet payment \
    --network-id "this-is-sebak-standalone" \
    --endpoint "http://localhost:12345" \
    --create \
    GCQ3RMPRKH5OCFNFCUB33GIV3OSCJXSTIXTMZOSWXYZYJGANYDJ6NOA3 \
    10000000000000 \
    SCAFTDXNCI76JNGT3PNJDYPXZGLMUWX3XKFIM3T2PF5EDPNR2QMBTTWN \
    --verbose
Account before transaction:  {GBUQWH7YMPSCNS53CPKBTEFKJONNZNUZWBXQSO7SSPWDT7M34N7OLCD2 9999989999999990000 1  [] [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]}
{
  "T": "transaction",
  "H": {
    "version": "",
    "created": "2018-10-07T13:54:21.495102000+09:00",
    "signature": "NMgWv37HiXicJdwEZozXM7LwUjmdyFMcKcT47i2viZH8VtQuqsxouDt7U3YbMJbFRSAawthFALgo46Pdmx6mmrh"
  },
  "B": {
    "source": "GBUQWH7YMPSCNS53CPKBTEFKJONNZNUZWBXQSO7SSPWDT7M34N7OLCD2",
    "fee": "10000",
    "sequenceid": 1,
    "operations": [
      {
        "H": {
          "type": "create-account"
        },
        "B": {
          "target": "GCQ3RMPRKH5OCFNFCUB33GIV3OSCJXSTIXTMZOSWXYZYJGANYDJ6NOA3",
          "amount": "10000000000000"
        }
      }
    ]
  }
}
Receiver account after 5 seconds:  {GCQ3RMPRKH5OCFNFCUB33GIV3OSCJXSTIXTMZOSWXYZYJGANYDJ6NOA3 10000000000000 0  [] [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]}
```

This will create account, it's public address is `GCQ3RMPRKH5OCFNFCUB33GIV3OSCJXSTIXTMZOSWXYZYJGANYDJ6NOA3` and it's initial balance will be `1000000.0000000 BOS`.

### Payment

```sh
$ sebak wallet payment \
    --network-id "this-is-sebak-standalone" \
    --endpoint "http://localhost:12345" \
    GCQ3RMPRKH5OCFNFCUB33GIV3OSCJXSTIXTMZOSWXYZYJGANYDJ6NOA3 \
    1 \
    SCAFTDXNCI76JNGT3PNJDYPXZGLMUWX3XKFIM3T2PF5EDPNR2QMBTTWN \
    --verbose
Account before transaction:  {GBUQWH7YMPSCNS53CPKBTEFKJONNZNUZWBXQSO7SSPWDT7M34N7OLCD2 9999979999999980000 2  [] [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]}
{
  "T": "transaction",
  "H": {
    "version": "",
    "created": "2018-10-07T13:57:11.023884000+09:00",
    "signature": "4gJSJFckF8A1YVKSDAz5S4SfrRunF5mYVLe9Wxv13LMGucCtHwzHnKfdVUtMEoQxDqB8xLAcptpJq53EAqjV1JEy"
  },
  "B": {
    "source": "GBUQWH7YMPSCNS53CPKBTEFKJONNZNUZWBXQSO7SSPWDT7M34N7OLCD2",
    "fee": "10000",
    "sequenceid": 2,
    "operations": [
      {
        "H": {
          "type": "payment"
        },
        "B": {
          "target": "GCQ3RMPRKH5OCFNFCUB33GIV3OSCJXSTIXTMZOSWXYZYJGANYDJ6NOA3",
          "amount": "1"
        }
      }
    ]
  }
}
Receiver account after 5 seconds:  {GCQ3RMPRKH5OCFNFCUB33GIV3OSCJXSTIXTMZOSWXYZYJGANYDJ6NOA3 10000000000001 0  [] [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]}
```

This will send `1BOS` from `GBUQWH7YMPSCNS53CPKBTEFKJONNZNUZWBXQSO7SSPWDT7M34N7OLCD2` to `GCQ3RMPRKH5OCFNFCUB33GIV3OSCJXSTIXTMZOSWXYZYJGANYDJ6NOA3`.