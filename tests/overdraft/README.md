# Overdraft tests

If you need to regenerate the test files (because some data changed),
here is how to do it.

## Prerequisite

- Be at the root of the project (the `git` root)
- Build the docker image: `docker build . -t sebak`
- Open 3 more terminals and start 3 nodes:
  - `docker run -it --rm --network host --env-file=docker/node1.env sebak`
  - `docker run -it --rm --network host --env-file=docker/node2.env sebak`
  - `docker run -it --rm --network host --env-file=docker/node3.env sebak`
- For simplicity, the following variables are assumed to be defined:
  ```sh
  export SEBAK_NETWORK_ID=sebak-test-network
  export SECRET1=SBECGI3FSCYHNQIMANNCWQSVA6S5C6L4BXFKAPMBAMI5V47NWXNE37MN
  export SECRET2=SABNXHXHIISL6NK3CZCOMKF6G7JMRFC5Z3C7DMMHSICWW736VKUWSJIA
  export SECRET3=SDMF6777DZEFVNLEKYWXVE7NZGYAPVK5JSU2N66ALTRV74RCASPV5A6V
  export PUBKEY1=GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ
  export PUBKEY2=GAYGELM74WJMKSLDN5YP2VAMP64WC4IXIGICUNK2SCVIT7KPTLY7M3MW
  export PUBKEY3=GDTEPFWEITKFHSUO44NQABY2XHRBBH2UBVGJ2ZJPDREIOL2F6RAEBJE4
  ```

## Commands used

```sh
docker run --network host -it sebak wallet payment ${PUBKEY3} 10_000 ${SECRET1} --endpoint=https://127.0.0.1:2821 --create --verbose
docker run --network host -it sebak wallet payment ${PUBKEY3} 0 ${SECRET1} --endpoint=https://127.0.0.1:2821 --verbose
docker run --network host -it sebak wallet payment ${PUBKEY3} 1 ${SECRET1} --endpoint=https://127.0.0.1:2821 --verbose
docker run --network host -it sebak wallet payment ${PUBKEY1} 100 ${SECRET3} --endpoint=https://127.0.0.1:2821 --verbose
docker run --network host -it sebak wallet payment ${PUBKEY1} 2 ${SECRET3} --endpoint=https://127.0.0.1:2821 --verbose
docker run --network host -it sebak wallet payment ${PUBKEY1} 1 ${SECRET3} --endpoint=https://127.0.0.1:2821 --verbose
```
