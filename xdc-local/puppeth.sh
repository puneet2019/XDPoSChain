#!/usr/bin/expect -f

spawn puppeth
expect "Please specify a network name"
send "genesis\r"

expect "What would you like to do?"
send "2\r"

expect "Which consensus engine"
send "3\r"

expect "How many seconds should blocks take?"
send "2\r"

expect "How many Ethers should be rewarded to masternode?"
send "100\r"

expect "Which block number start v2 consesus?"
send "\r"

expect "How long is the v2 timeout period?"
send "\r"

expect "How many v2 timeout reach to send Synchronize message?"
send "\r"

expect "Proportion of total masternodes v2 vote collection to generate a QC (float value), should be two thirds of masternodes?"
send "\r"

expect "Who own the first masternodes?"
send "65120334d967b1bd3c4c8c7e3b78bef360cdb73d\r"

expect "Which accounts are allowed to seal (signers)?"
send "65120334d967b1bd3c4c8c7e3b78bef360cdb73d\r"
send "c5139c89013235289d4628243bf40935afb0acd7\r"
send "332bdfddf2575ff83ae3a996c24893fbabf5a0c9\r"
send "\r"

expect "How many blocks per epoch?"
send "9000\r"

expect "How many blocks before checkpoint need to prepare new set of masternodes?"
send "4500\r"

expect "What is foundation wallet address?"
send "65120334d967b1bd3c4c8c7e3b78bef360cdb73d\r"

expect "Which accounts are allowed to confirm in Foudation MultiSignWallet?"
send "65120334d967b1bd3c4c8c7e3b78bef360cdb73d\r\r"

expect "How many require for confirm tx in Foudation MultiSignWallet?"
send "1\r"

expect "Which accounts are allowed to confirm in Team MultiSignWallet?"
send "65120334d967b1bd3c4c8c7e3b78bef360cdb73d\r\r"

expect "How many require for confirm tx in Team MultiSignWallet?"
send "1\r"

expect "What is swap wallet address for fund 55m XDC?"
send "65120334d967b1bd3c4c8c7e3b78bef360cdb73d\r"

expect "Which accounts should be pre-funded?"
send "65120334d967b1bd3c4c8c7e3b78bef360cdb73d\r"
send "c5139c89013235289d4628243bf40935afb0acd7\r"
send "332bdfddf2575ff83ae3a996c24893fbabf5a0c9\r"
send "\r"

expect "Specify your chain/network ID if you want an explicit one"
send "2025\r"

expect "What would you like to do?"
send "2\r"

expect "What would you like to do?"
send "2\r"

expect "Which file to save the genesis int"
send "\r"

interact
