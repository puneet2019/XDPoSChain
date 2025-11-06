#!/bin/bash


sudo chmod -R +777 .

rm -r keys/*/XDC*
rm -r keys/*/genesis.json
rm -r keys/*/xdc.log
rm -r keys/

#-------#

mkdir -p xdc-local/keys
cd ~/xdc-local

## GENERATE genesis.json in ~/xdc-local

mkdir -p  keys/bootnode keys/node1 keys/node2 keys/node3
# Put the private key strings in files (no 0x prefix)
echo "e7064463b0d8670605f55f2ee4f7718eb17f4827b6b7f72ddaa88e8d03df15c2" > keys/bootnode/key

echo "27a2b34095c9f755317ed1c9352684cff841f7a9349e940216408fad6e2a7509" > keys/node1/key
echo "0a2bf44acc2606290162c6bae4f86a8f391b69af5355ed5a720f3c9d009f7c72" > keys/node2/key
echo "0b3931890f1db7d7a865f05ec58f6785bac0ec15d960803846f6520b2f7c6a36" > keys/node3/key

touch keys/.pwd
cp keys/.pwd keys/bootnode/
cp keys/.pwd keys/node1/
cp keys/.pwd keys/node2/
cp keys/.pwd keys/node3/

# generate bootnode.key (32 bytes hex)
openssl rand -hex 32 > keys/bootnode/nodekey
# lock down permissions so only you can read it
chmod 600 keys/bootnode/nodekey

## for no perms in docker created files
export MYUID=$(id -u)
export MYGID=$(id -g)
echo $MYUID $MYGID


# bootnode
docker run --rm -it \
  -v "$(pwd)/keys/bootnode":/work/xdcchain \
  -v "$(pwd)/keys/bootnode/key":/tmp/key:ro \
  -v "$(pwd)/keys/.pwd":/work/.pwd:ro \
  --entrypoint "" \
  --user "${MYUID}:${MYGID}" \
  xinfinorg/xdposchain:latest \
  /usr/bin/XDC-devnet account import --password /work/.pwd --datadir /work/xdcchain /tmp/key

# Node1
docker run --rm -it \
  -v "$(pwd)/keys/node1":/work/xdcchain \
  -v "$(pwd)/keys/node1/key":/tmp/key:ro \
  -v "$(pwd)/keys/.pwd":/work/.pwd:ro \
  --entrypoint "" \
  --user "${MYUID}:${MYGID}" \
  xinfinorg/xdposchain:latest \
  /usr/bin/XDC-devnet account import --password /work/.pwd --datadir /work/xdcchain /tmp/key

# Node2
docker run --rm -it \
  -v "$(pwd)/keys/node2":/work/xdcchain \
  -v "$(pwd)/keys/node2/key":/tmp/key:ro \
  -v "$(pwd)/keys/.pwd":/work/.pwd:ro \
  --entrypoint "" \
  --user "${MYUID}:${MYGID}" \
  xinfinorg/xdposchain:latest \
  /usr/bin/XDC-devnet account import --password /work/.pwd --datadir /work/xdcchain /tmp/key

# Node3
docker run --rm -it \
  -v "$(pwd)/keys/node3":/work/xdcchain \
  -v "$(pwd)/keys/node3/key":/tmp/key:ro \
  -v "$(pwd)/keys/.pwd":/work/.pwd:ro \
  --entrypoint "" \
  --user "${MYUID}:${MYGID}" \
  xinfinorg/xdposchain:latest \
  /usr/bin/XDC-devnet account import --password /work/.pwd --datadir /work/xdcchain /tmp/key

# boot, 1, 2, 3
xdc777a7ec18c7787d9cfed77349445927a73791233
xdc65120334d967b1bd3c4c8c7e3b78bef360cdb73d
xdcc5139c89013235289d4628243bf40935afb0acd7
xdc332bdfddf2575ff83ae3a996c24893fbabf5a0c9

777a7ec18c7787d9cfed77349445927a73791233
65120334d967b1bd3c4c8c7e3b78bef360cdb73d
c5139c89013235289d4628243bf40935afb0acd7
332bdfddf2575ff83ae3a996c24893fbabf5a0c9
# bootnode initialization (we’ll also mount nodekey for persistent enode)
export MYUID=$(id -u)
export MYGID=$(id -g)
echo $MYUID $MYGID
docker run --rm -it \
  -v "$(pwd)/genesis.json":/work/genesis.json:ro \
  -v "$(pwd)/keys/bootnode":/root/keys:ro \
  -v "$(pwd)/keys/node1":/work/xdcchain \
  --entrypoint "" \
  --user "${MYUID}:${MYGID}" \
  xinfinorg/xdposchain:latest \
  /usr/bin/XDC-devnet init /work/genesis.json --datadir /work/xdcchain

# bootnode initialization (we’ll also mount nodekey for persistent enode)
docker run --rm -it \
  -v "$(pwd)/genesis.json":/work/genesis.json:ro \
  -v "$(pwd)/keys/bootnode":/root/keys:ro \
  -v "$(pwd)/keys/node2":/work/xdcchain \
  --entrypoint "" \
  --user "${MYUID}:${MYGID}" \
  xinfinorg/xdposchain:latest \
  /usr/bin/XDC-devnet init /work/genesis.json --datadir /work/xdcchain

# bootnode initialization (we’ll also mount nodekey for persistent enode)
docker run --rm -it \
  -v "$(pwd)/genesis.json":/work/genesis.json:ro \
  -v "$(pwd)/keys/bootnode":/root/keys:ro \
  -v "$(pwd)/keys/node3":/work/xdcchain \
  --entrypoint "" \
  --user "${MYUID}:${MYGID}" \
  xinfinorg/xdposchain:latest \
  /usr/bin/XDC-devnet init /work/genesis.json --datadir /work/xdcchain

docker run --rm -d \
  -v "$(pwd)/genesis.json":/root/genesis.json:ro \
  -v "$(pwd)/keys/bootnode/nodekey":/root/nodekey:ro \
  -v "$(pwd)/keys/node1":/work/xdcchain \
  -e NETWORK=devnet \
  --name xdc-bootnode-temp \
  xinfinorg/xdposchain:latest \
  /usr/bin/XDC-devnet --datadir /work/xdcchain --networkid 2025 --nodekey /root/nodekey --nodiscover --nat none --port 30301 --http --http.addr 0.0.0.0 --http.port 8545 --http.api eth,net,web3,txpool,personal --etherbase xdc777a7ec18c7787d9cfed77349445927a73791233 --allow-insecure-unlock

sleep 4
docker exec -it xdc-bootnode-temp /bin/sh -c "/usr/bin/XDC-devnet attach /work/xdcchain/XDC.ipc --exec 'admin.nodeInfo.enode'"
#enode://4f9d9ccd7ec1658f07bcbbe0482c541e9bf7c00e889246e3921356d5da58e935a1429570046df896662ad7752b39b7e0b82706d0244ce38a129dc1acb4eecb2b@51.38.184.188:30303

sudo chmod -R +777 .

docker rm -f xdc-bootnode-temp

## generate docker-compose, edit enode and addresses

# make sure MYUID / MYGID are exported
export MYUID=$(id -u)
export MYGID=$(id -g)

#----
#
#docker compose up -d
#
#docker logs -f xdc-bootnode
#docker logs -f xdc-node1
#
#
#sudo chmod -R +777 .
#
#rm -r keys/*/XDC*
#rm -r keys/*/genesis.json
#rm -r keys/*/xdc.log
#rm -r keys/

