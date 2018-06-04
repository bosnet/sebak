# SEBAK

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](http://godoc.org/github.com/owlchain/sebak/lib) [![Build Status](https://travis-ci.org/owlchain/sebak.svg?branch=master)](https://travis-ci.org/owlchain/sebak)

Sebak is the core node for crypto-currency blockchain.

## Installation

Before installing, you must install Go 1.9 or above and 'dep'. You can check how to install 'dep' in [dep installation](https://github.com/golang/dep#installation).

To start sebak:

```
$ mkdir sebak
$ cd sebak
$ export GOPATH=$(pwd)
$ go get github.com/owlchain/sebak
$ cd src/github.com/owlchain/sebak
$ dep ensure
$ go install github.com/owlchain/sebak/cmd/sebak
```

## Test

You can test sebak.

```
$ go test ./...
```

You can see the detailed logs:
```
$ go test ./... -v
```

## Generating Keypair

sebak can make keypair, 'secret seed' and 'public address'.
```
$ sebak key generate
       Secret Seed: SCN4NSV5SVHIZWUDJFT4Z5FFVHO3TFRTOIBQLHMNPAZJ37K5A2YFSCBM
    Public Address: GALQG5SCKCPXUG4ODPMFZJGZ6XBVJTLAJFR7OJKJOJVARA7M4H5SGSOG
```

## Create Genesis Block

Before running node, you must generate genesis block.

```
$ sebak genesis GALQG5SCKCPXUG4ODPMFZJGZ6XBVJTLAJFR7OJKJOJVARA7M4H5SGSOG --balance 1,000,000,000,000.0000000
successfully created genesis block
```

## Deploying Node

To run sebak, you need SSL certificates for HTTP2 protocol. To create self-signed SSL certificates, see [Generating a self-signed certificate using OpenSSL](https://www.ibm.com/support/knowledgecenter/en/SSWHYP_4.0.0/com.ibm.apimgmt.cmc.doc/task_apionprem_gernerate_self_signed_openSSL.html).

After you successfully create genesis block and your own SSL certificates, run sebak:
```
$ sebak node --network-id 'this-is-test-sebak-network' --endpoint "https://localhost:12345" --tls-key 'sebak.key' --tls-cert 'sebak.crt' --log-level debug --secret-seed SCN4NSV5SVHIZWUDJFT4Z5FFVHO3TFRTOIBQLHMNPAZJ37K5A2YFSCBM --validator GBWCMWDUZK67YNUZ44UPNVFYZRSCCS4OLE6ORWD4ZLI2MVGY4KJDPHMO,https://localhost:12346 --validator GCHCJZQ3F6XJCO6EP47XVWX6UT53GTBZ5T6BQZSODGJWOU3LOORQ4TN6,https://localhost:12347
```
