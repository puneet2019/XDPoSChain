#!/bin/bash
set -e

# ---- Configuration ----
RPC="http://127.0.0.1:8546"
FROM_ADDR="0x65120334D967b1BD3C4C8c7e3b78bef360cDB73D"
VALUE=100000
DOCKER_NODE="xdc-node1"

# ---- Generate mnemonic & derive address ----
MNEMONIC=$(cast wallet new-mnemonic --json | jq -r '.mnemonic')
PRIV=$(cast wallet private-key --mnemonic "$MNEMONIC" --json)
ADDR=$(cast wallet address --private-key "$PRIV")

echo "Mnemonic: $MNEMONIC"
echo "Derived address: $ADDR"

# ---- Fund address using docker attach ----
TX_CMD="eth.sendTransaction({ from: '$FROM_ADDR', to: '$ADDR', value: web3.toWei($VALUE, 'ether') })"
echo "Sending transaction inside $DOCKER_NODE..."
docker exec -i $DOCKER_NODE /usr/bin/XDC-devnet attach /root/.xdpos/XDC.ipc <<EOF
$TX_CMD
exit
EOF

echo "Done! Funded $ADDR"

pandoras-box -url http://127.0.0.1:8546 -url http://127.0.0.1:8547 -url http://127.0.0.1:8548 -s 10 -t 2000 -b 2000 -o ./myOutput.json -m "$MNEMONIC"