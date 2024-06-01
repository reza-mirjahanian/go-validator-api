


REST Ethereum Validator API


## For "/syncduties/:slot" :
I've called two APIs sequentially. First, I find the validator indexes, and then I get their public keys.

```
https://RPC_ENDPOINT/eth/v1/beacon/states/8765432/sync_committees
https://RPC_ENDPOINT/eth/v1/beacon/states/8765432/validators?id=943501&id=1239839&id=513381&id=482814&id=1082134&id=462664&id=752468
```

More info:
[https://docs.blastapi.io/blast-documentation/apis-documentation/core-api/ethereum/holesky-beacon/beacon/eth-v1-beacon-states-state_id-sync_committees](https://docs.blastapi.io/blast-documentation/apis-documentation/core-api/ethereum/holesky-beacon/beacon/eth-v1-beacon-states-state_id-sync_committees)


[https://docs.blastapi.io/blast-documentation/apis-documentation/core-api/ethereum/holesky-beacon/beacon/eth-v1-beacon-states-state_id-validators](https://docs.blastapi.io/blast-documentation/apis-documentation/core-api/ethereum/holesky-beacon/beacon/eth-v1-beacon-states-state_id-validators)


## For "/blockreward/:slot" :
1. Checks if the slot is before the Paris merge slot. If yes, returns
   an error.

2. Compares the slot with the current slot to ensure it's not in the
   future.

3. Fetch Block Hash

4. Fetch Block by Hash( BlockByHash function)

5. Calculate Fees

6. Process Transactions:
   -   Iterates over each transaction in the block.
   -   Fetches transaction receipts.
   -   Calculates the cost and checks if the transaction is a MEV transaction by comparing gas price with three times the base fee.

7. Calculate Reward

```bash
### fetch current head slot blocks
https://RPC_ENDPOINT/eth/v1/beacon/headers


https://RPC_ENDPOINT/eth/v2/beacon/blocks/8765432
```


More info:


https://docs.blastapi.io/blast-documentation/apis-documentation/core-api/ethereum/holesky-beacon/beacon/eth-v1-beacon-headers


[https://docs.blastapi.io/blast-documentation/apis-documentation/core-api/ethereum/holesky-beacon/beacon/eth-v2-beacon-blocks-block_id](https://docs.blastapi.io/blast-documentation/apis-documentation/core-api/ethereum/holesky-beacon/beacon/eth-v2-beacon-blocks-block_id)

---------------


![alt text](docs/blockreward.png)
![alt text](docs/syncduties.png)
![alt text](docs/env.png)
![alt text](docs/compose.png)

--------------------------
--------------------------

## Running the app with Docker ( Default port is :9090)
```bash
docker compose -f docker-compose.dev.yml  up
```



## Running the app locally( you need Go installed - default port is :8080)

- Create an ".env" file, similar to ".env.sample". Fix the value of "RPC_ENDPOINT".
```bash
go run .
```

## Example requests:
check requests.http file
https://marketplace.visualstudio.com/items?itemName=humao.rest-client

## Unit Tests

```bash
@todo
```

#### Swagger UI
```bash
@todo
```

## Info:
- 📌 gin
- 📌 go-ethereum
- 📌 spf13/viper



#### Done:
- ✅ GET /blockreward/{slot}
- ✅ GET /syncduties/{slot}




#### Todo:
- 💡 Better naming conventions and folder structure
- 💡 Caching with Redis
- 💡 Include API versioning,


