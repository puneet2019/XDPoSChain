this repo belongs to: 

https://github.com/sig-0/pandoras-box

readme found in README.md

on top edits have been made to not wait for tx receipts, allow for multiple rpc urls to fire txs.
And tps to be calculated based on information from querying node info - block height, block time, block tx count.
TPS = block tx count / (block time - last block time)

Additional service:
`pandoras-box -url http://127.0.0.1:8546 --only-tps` 
is to be run which keeps querying blocks, txs and hence TPS. 

These edits have been asked to be done by an ai-agent, and it suffices the purpose of the project.