exit 1


docker exec -it xdc-node1 /usr/bin/XDC-devnet attach /root/.xdpos/XDC.ipc


eth.sendTransaction({ from: "0x65120334D967b1BD3C4C8c7e3b78bef360cDB73D", to: "0xF62242F0e2adCd6FBA36292A073824Fc7746d9bC", value: web3.toWei(100000, "ether") })
eth.sendTransaction({ from: "0x65120334D967b1BD3C4C8c7e3b78bef360cDB73D", to: "0xf1B41746829b1710b07974e913C6EDf38B211F55", value: web3.toWei(100000, "ether") })
eth.sendTransaction({ from: "0x65120334D967b1BD3C4C8c7e3b78bef360cDB73D", to: "0xDA9723aF829db6DFEaD10D330b0308500D04505b", value: web3.toWei(100000, "ether") })
eth.sendTransaction({ from: "0x65120334D967b1BD3C4C8c7e3b78bef360cDB73D", to: "0x0BBEE48B064f8e676e41387e63d4A4D022dA5045", value: web3.toWei(100000, "ether") })
eth.sendTransaction({ from: "0x65120334D967b1BD3C4C8c7e3b78bef360cDB73D", to: "0x5bac833ea17aB7b483aC41e3200d687F647648c0", value: web3.toWei(100000, "ether") })
eth.sendTransaction({ from: "0x65120334D967b1BD3C4C8c7e3b78bef360cDB73D", to: "0x04F0A0455846e3Dca1Adc2088A4f98bB11e3e59E", value: web3.toWei(100000, "ether") })
eth.sendTransaction({ from: "0x65120334D967b1BD3C4C8c7e3b78bef360cDB73D", to: "0x9F46c3a79C492C65219e267992ac03BAf1a1fA59", value: web3.toWei(100000, "ether") })
eth.sendTransaction({ from: "0x65120334D967b1BD3C4C8c7e3b78bef360cDB73D", to: "0x989864afc2339D9CCE5699e45A5E67Eca5cd092A", value: web3.toWei(100000, "ether") })
eth.getTransaction("")

#0xF62242F0e2adCd6FBA36292A073824Fc7746d9bC
pandoras-box -url http://127.0.0.1:8546 -url http://127.0.0.1:8547 -url http://127.0.0.1:8548 -s 2000 -t 20000 -b 10000 -o ./myOutput.json -m "erupt oven loud noise rug proof sunset gas table era dizzy vault" &
#0xf1B41746829b1710b07974e913C6EDf38B211F55
pandoras-box -url http://127.0.0.1:8546 -url http://127.0.0.1:8547 -url http://127.0.0.1:8548 -s 2000 -t 20000 -b 10000 -o ./myOutput.json -m "banana give mushroom east universe clever best side pretty certain hospital draft"  &
#0xDA9723aF829db6DFEaD10D330b0308500D04505b
pandoras-box -url http://127.0.0.1:8546 -url http://127.0.0.1:8547 -url http://127.0.0.1:8548 -s 2000 -t 20000 -b 10000 -o ./myOutput.json -m "scatter menu warrior sister spare balcony sound gospel awake enlist sniff old"  &
#0x0BBEE48B064f8e676e41387e63d4A4D022dA5045
pandoras-box -url http://127.0.0.1:8546 -url http://127.0.0.1:8547 -url http://127.0.0.1:8548 -s 2000 -t 20000 -b 10000 -o ./myOutput.json -m "venture intact prefer clinic funny nut pledge pact erode match copper garage"  &
#0x5bac833ea17aB7b483aC41e3200d687F647648c0
pandoras-box -url http://127.0.0.1:8546 -url http://127.0.0.1:8547 -url http://127.0.0.1:8548 -s 2000 -t 20000 -b 10000 -o ./myOutput.json -m "snow tomato service subject actress century any can whisper acquire leave recall"  &
#0x04F0A0455846e3Dca1Adc2088A4f98bB11e3e59E
pandoras-box -url http://127.0.0.1:8546 -url http://127.0.0.1:8547 -url http://127.0.0.1:8548 -s 2000 -t 20000 -b 10000 -o ./myOutput.json -m "ocean life common escape ride robust live cabbage arena album staff brush"  &
#0x9F46c3a79C492C65219e267992ac03BAf1a1fA59
pandoras-box -url http://127.0.0.1:8546 -url http://127.0.0.1:8547 -url http://127.0.0.1:8548 -s 2000 -t 20000 -b 10000 -o ./myOutput.json -m "magnet clay doctor square spin dignity subway legend solution nation amazing eternal"  &
#0x989864afc2339D9CCE5699e45A5E67Eca5cd092A
pandoras-box -url http://127.0.0.1:8546 -url http://127.0.0.1:8547 -url http://127.0.0.1:8548 -s 2000 -t 20000 -b 10000 -o ./myOutput.json -m "april creek drum sword trap injury duck wash crystal social cream holiday"  &
wait

for i in {1..20}; do
  ./autoscript.sh &
done
wait