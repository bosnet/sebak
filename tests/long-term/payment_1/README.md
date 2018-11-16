# Payment 1

Create 10 Payment txs

1. Run `request_2~.json` to `request_11~.json` with the following:
```
    curl --insecure \
         --request POST \
         --header "Content-Type: application/json" \
         --data {.json} \
         https://127.0.0.1:${PORT}/api/v1/transactions \
         >/dev/null 2>&1
```
   1. request_2: Payment node1 -> node2, node3
      request_3: Payment node2 -> node1, node3
      request_4: Payment node3 -> node4, node5, node6
      request_5: Payment node4 -> node1, node3
      request_6: Payment node5 -> node6, node7, node8
      request_7: Payment node6 -> node2, node5
      request_8: Payment node7 -> node8, node10
      request_9: Payment node8 -> node7, node5
      request_10: Payment node9 -> node10
      request_11: Payment node10 -> node9, node4

1. Run `payment_1.check`
