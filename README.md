# SEBAK

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](http://godoc.org/github.com/bosnet/sebak/lib) [![CircleCI](https://circleci.com/gh/bosnet/sebak.svg?style=svg&circle-token=fd8cbd27a7594539b58dd3c46363a2c693f25edb)](https://circleci.com/gh/bosnet/sebak)

Sebak is the core node for crypto-currency blockchain.

## Installation

Before installing, you must install Go 1.9 or above and 'dep'. You can check how to install 'dep' in [dep installation](https://github.com/golang/dep#installation).

To start sebak:

```
$ mkdir sebak
$ cd sebak
$ export GOPATH=$(pwd)
$ go get boscoin.io/sebak
$ cd src/boscoin.io/sebak
$ dep ensure
$ go install boscoin.io/sebak/cmd/sebak
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
$ cd cmd/sebak
$ sebak node \
    --network-id 'this-is-test-sebak-network' \
    --endpoint "https://localhost:12345" \
    --tls-key 'sebak.key' \
    --tls-cert 'sebak.crt' \
    --log-level debug \
    --secret-seed SCN4NSV5SVHIZWUDJFT4Z5FFVHO3TFRTOIBQLHMNPAZJ37K5A2YFSCBM \
    --validator GBWCMWDUZK67YNUZ44UPNVFYZRSCCS4OLE6ORWD4ZLI2MVGY4KJDPHMO,https://localhost:12346 \ --validator GCZG7MBKRSS6MJVZOALYBJB5C223FSZ43MDTPX2O4UGQTCXTHWBDNUB6,https://localhost:12347
```

To run multiple sebak nodes on the local machine, then you must set up the different db paths each:

- node12345
```
$ sebak node \
    --network-id 'this-is-test-sebak-network' \
    --endpoint "https://localhost:12345" \
    --tls-key 'sebak.key' \
    --tls-cert 'sebak.crt' \
    --log-level debug \
    --storage=file:///tmp/db5 \
    --secret-seed SCN4NSV5SVHIZWUDJFT4Z5FFVHO3TFRTOIBQLHMNPAZJ37K5A2YFSCBM \
    --validator GDPQ2LBYP3RL3O675H2N5IEYM6PRJNUA5QFMKXIHGTKEB5KS5T3KHFA2,https://localhost:12346 \
    --validator GCZG7MBKRSS6MJVZOALYBJB5C223FSZ43MDTPX2O4UGQTCXTHWBDNUB6,https://localhost:12347
```

- node12346
```
$ sebak node \
    --network-id 'this-is-test-sebak-network' \
    --endpoint "https://localhost:12346" \
    --tls-key 'sebak.key' \
    --tls-cert 'sebak.crt' \
    --log-level debug \
    --storage=file:///tmp/db6 \
    --secret-seed SBGJDQ2J4PIYU7JVGKIBLNF6X3DOEVW3I4W2T77M2B47X2MPSUNXZ7T7 \
    --validator GBNUTWSM4FRSEULVMHZF7NFQWIBGEDF5X5OHXFOZJB6SH5MIEDEJEJ2F,https://localhost:12345 \
    --validator GCZG7MBKRSS6MJVZOALYBJB5C223FSZ43MDTPX2O4UGQTCXTHWBDNUB6,https://localhost:12347
```

- node12347
```
$ sebak node \
    --network-id 'this-is-test-sebak-network' \
    --endpoint "https://localhost:12347" \
    --tls-key 'sebak.key' \
    --tls-cert 'sebak.crt' \
    --log-level debug \
    --storage=file:///tmp/db7 \
    --secret-seed SDQKKG2MBSAXVLUE5JFNM7MXQ7MV7WPRIEOS7U7KLWFDKYDKXTLSSRTC \
    --validator GBNUTWSM4FRSEULVMHZF7NFQWIBGEDF5X5OHXFOZJB6SH5MIEDEJEJ2F,https://localhost:12345 \
    --validator GDPQ2LBYP3RL3O675H2N5IEYM6PRJNUA5QFMKXIHGTKEB5KS5T3KHFA2,https://localhost:12346
```

## Spinning a test net using Docker

To spawn a simple network, first build the docker image:
```sh
docker build . -t sebak
# The build process creates a rather large image, which is regenarated every time
# To clean up all orphaned images, you can run the following command
docker rmi -f $(docker images -f "dangling=true" -q)
```

Then you can spawn 3 nodes using the following commands:
```sh
docker run --net host --rm -it --env-file=docker/node1.env sebak
docker run --net host --rm -it --env-file=docker/node2.env sebak
docker run --net host --rm -it --env-file=docker/node3.env sebak
```

Changing the number of test nodes should be fairly simple once #46 is solved.
