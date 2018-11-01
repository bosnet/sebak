### *Testnet*
As its name implies, *Testnet* is used for testing purpose. You can create new accounts without any permission or restriction and send payment to other accounts. This network is isolated from the *existing token-net network*.

SEBAK, the code name of the node powering *Testnet* is under active development. Be aware that as a consequence, *Testnet* might experience some periods of instability.

*Testnet* will be running for testing various kind of experimental features and implementations. You can test all the available features of SEBAK without any restrictions in *Testnet*.

> NOTICE : Your accounts and histories can be changed or deleted without notification.

# Constants

| name | value |
| --: | -- |
| *network-id* | `sebak-testnet` |
| *genesis account* | `GDMZMF2EAK4E6NSZNSCJQQHQGMAOZ6UI3XQVVLMEJRFDPYHLY7PPHKLP` |
| *total balance* | `1,000,000,000,000.0000000`BOS |
| *Minimum Balance* | `0.1`BOS(`1,000,000`GON) |
| *Base Fee* | `0.001`BOS(`10,000`GON) |

### *Minimum Balance*
The minimum balance of new account.

### *Base Fee*
When you send a transaction, creating an account or sending a payment, *Testnet* network charges an extra *fee* to handle your transaction. **Base Fee** is the amount charged.

# Nodes

| node | endpoint | validators |
| -- | -- | -- |
| *dev-0* | https://dev-0.sebak-testnet.dev.boscoin.io | *dev-1* *dev-2* *dev-3* |
| *dev-1* | https://dev-1.sebak-testnet.dev.boscoin.io | *dev-0* *dev-2* *dev-3* |
| *dev-2* | https://dev-2.sebak-testnet.dev.boscoin.io | *dev-0* *dev-1* *dev-3* |
| *dev-3* | https://dev-3.sebak-testnet.dev.boscoin.io | *dev-0* *dev-1* *dev-2* |

# Initial Accounts

| account | initial balance | public address | secret seed |
| -- | -- | -- |  -- |
| `a0` | `5,000,000` | `GA5ZQ37YVVRAVKCWPFRTS6CF2LH76UNRFY76G75Q2ABDQ6QZ37G24GHO` | `SCDMSX3ITG2OAI4CYFZQKWPWU2PZDJUDOQKIUWQSDEUZ6NZDTOLOFBWO` |
| `a1` | `5,000,000` | `GCZD2VEVDAQEYSGOHMPOTYEWTNIJBM5JS3LZCUIL23FU4PZMRTWP6ZCS` | `SBDTZW6YCQPHLW2DIMV3SSPC7LQSBJ7IKCVV6CGUOBVDXAQPOJY5WDKB` |
| `a2` | `5,000,000` | `GB67C5RHN6EHS7MVO4ZLL3RJ3INIUXIXKZ74NVYHWW6HU3YQHMXONMXN` | `SAICU2PX2FMPRPABHU73LNDEFCMF37YNNKKDR6BSFEWREK52BXKBWKAX` |
| `a3` | `5,000,000` | `GDKMOAHEZULSJM55MXUJ5LLHBW4QZPQHERU3ACD6MUCGB4Q2QC2E3MSJ` | `SA5YHDBS57DULW7GEGILQLSTMMNH7L5LWMAH5GDDSAYAEVTOZ3YFDVTU` |

The network is initialized with 4 accounts. You can test the network with these accounts by `sebak wallet` command. For more details, please see [`sebak wallet`](./sebak_command.md#sebak-wallet).

> ## Creating accounts will be supported as soon as possible.