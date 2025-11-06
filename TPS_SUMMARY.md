The setup consists of 3 validator nodes and 1 bootnode.

NOTE: Time is sometimes negative -> its because blocks are synced in random order 
(probably the script doesnt poll for blocks, rather just listens for coming blocks, but it's not important as we can draw conclusions without it)

Initially with 2 sec blocktime: 

Block #120 | Txs:    0 | Time: 2s | TPS:   0.00 | Gas: 0.00%
Block #121 | Txs:    3 | Time: 2s | TPS:   1.50 | Gas: 0.31%
Block #122 | Txs:  902 | Time: 2s | TPS: 451.00 | Gas: 22.55%
Block #123 | Txs: 1658 | Time: 2s | TPS: 829.00 | Gas: 41.45%
Block #124 | Txs: 1389 | Time: 2s | TPS: 694.50 | Gas: 34.72%
Block #125 | Txs: 1557 | Time: 2s | TPS: 778.50 | Gas: 38.92%
Block #126 | Txs: 1353 | Time: 2s | TPS: 676.50 | Gas: 33.82%
Block #127 | Txs: 1275 | Time: 2s | TPS: 637.50 | Gas: 31.87%
Block #129 | Txs:  339 | Time: 4s | TPS:  84.75 | Gas: 8.47%
Block #128 | Txs: 1527 | Time: -2s | TPS:      0 | Gas: 38.17% ~750
Block #130 | Txs:    0 | Time: 4s | TPS:   0.00 | Gas: 0.00%
Block #131 | Txs:  630 | Time: 2s | TPS: 315.00 | Gas: 15.75%
Block #132 | Txs: 1324 | Time: 2s | TPS: 662.00 | Gas: 33.10%
Block #133 | Txs: 1274 | Time: 2s | TPS: 637.00 | Gas: 31.85%
Block #134 | Txs:  875 | Time: 4s | TPS: 218.75 | Gas: 21.87%
Block #135 | Txs: 1759 | Time: 2s | TPS: 879.50 | Gas: 43.97%
Block #136 | Txs: 2206 | Time: 8s | TPS: 275.75 | Gas: 55.38%
Block #137 | Txs: 2518 | Time: 5s | TPS: 503.60 | Gas: 62.95%
Block #138 | Txs: 4000 | Time: 10s | TPS: 400.00 | Gas: 100.00%
Block #139 | Txs:  168 | Time: 24s | TPS:   7.00 | Gas: 4.20%
Block #140 | Txs: 2157 | Time: 12s | TPS: 179.75 | Gas: 53.92%
Block #141 | Txs:    0 | Time: 26s | TPS:   0.00 | Gas: 0.00%
Block #142 | Txs: 1078 | Time: 2s | TPS: 539.00 | Gas: 26.95%

After reducing blocktime to 1 sec
Block #899 | Txs:  430 | Time: 2s | TPS: 215.00 | Gas: 10.75%
Block #900 | Txs: 1809 | Time: 1s | TPS: 1809.00 | Gas: 45.22%
Block #898 | Txs:  439 | Time: -2s | TPS:      0 | Gas: 10.97%
Block #901 | Txs:  369 | Time: 4s | TPS:  92.25 | Gas: 9.46%
Block #902 | Txs:  190 | Time: 1s | TPS: 190.00 | Gas: 4.75%
Block #903 | Txs: 1405 | Time: 1s | TPS: 1405.00 | Gas: 35.12%
Block #904 | Txs: 1116 | Time: 3s | TPS: 372.00 | Gas: 27.90%
Block #905 | Txs: 1091 | Time: 2s | TPS: 545.50 | Gas: 27.27%
Block #906 | Txs: 4000 | Time: 3s | TPS: 1333.33 | Gas: 100.00%
Block #907 | Txs: 2721 | Time: 7s | TPS: 388.71 | Gas: 68.02%
Block #908 | Txs: 1700 | Time: 7s | TPS: 242.86 | Gas: 42.50%
Block #909 | Txs: 4000 | Time: 4s | TPS: 1000.00 | Gas: 100.00%
Block #911 | Txs:  111 | Time: 9s | TPS:  12.33 | Gas: 2.77%
Block #910 | Txs: 2211 | Time: -4s | TPS:      0 | Gas: 55.27%
Block #912 | Txs: 4000 | Time: 6s | TPS: 666.67 | Gas: 100.00%
Block #913 | Txs:  926 | Time: 18s | TPS:  51.44 | Gas: 23.15%
Block #914 | Txs: 3919 | Time: 9s | TPS: 435.44 | Gas: 97.97%
Block #915 | Txs:  448 | Time: 14s | TPS:  32.00 | Gas: 11.20%
Block #916 | Txs:  839 | Time: 1s | TPS: 839.00 | Gas: 21.21%
Block #917 | Txs: 2527 | Time: 2s | TPS: 1263.50 | Gas: 63.17%
Block #918 | Txs:  519 | Time: 5s | TPS: 103.80 | Gas: 12.97%
Block #919 | Txs:  528 | Time: 9s | TPS:  58.67 | Gas: 13.20%
Block #920 | Txs: 2188 | Time: 2s | TPS: 1094.00 | Gas: 54.70%
Block #921 | Txs: 1524 | Time: 12s | TPS: 127.00 | Gas: 38.10%
Block #922 | Txs:  547 | Time: 4s | TPS: 136.75 | Gas: 13.67%
Block #923 | Txs: 2958 | Time: 2s | TPS: 1479.00 | Gas: 73.95%
Block #924 | Txs:    0 | Time: 15s | TPS:   0.00 | Gas: 0.00%
Block #925 | Txs: 3200 | Time: 8s | TPS: 400.00 | Gas: 80.00%
Block #926 | Txs:    0 | Time: 30s | TPS:   0.00 | Gas: 0.00%
Block #927 | Txs:  957 | Time: 2s | TPS: 478.50 | Gas: 23.92%
.
.
.
Block #1122 | Txs:  110 | Time: 1s | TPS: 110.00 | Gas: 2.75%
Block #1123 | Txs:  542 | Time: 1s | TPS: 542.00 | Gas: 13.55%
Block #1124 | Txs: 1303 | Time: 2s | TPS: 651.50 | Gas: 32.57%
Block #1125 | Txs:   21 | Time: 3s | TPS:   7.00 | Gas: 0.52%
Block #1126 | Txs: 1911 | Time: 1s | TPS: 1911.00 | Gas: 48.01%
Block #1127 | Txs: 3356 | Time: 6s | TPS: 559.33 | Gas: 83.90%
Block #1128 | Txs: 4000 | Time: 10s | TPS: 400.00 | Gas: 100.00%
Block #1129 | Txs:  275 | Time: 14s | TPS:  19.64 | Gas: 6.87%
Block #1130 | Txs: 1239 | Time: 2s | TPS: 619.50 | Gas: 30.97%
Block #1131 | Txs: 3203 | Time: 10s | TPS: 320.30 | Gas: 80.07%
Block #1132 | Txs:  253 | Time: 17s | TPS:  14.88 | Gas: 6.32%
Block #1133 | Txs: 1491 | Time: 4s | TPS: 372.75 | Gas: 37.27%
Block #1134 | Txs:  349 | Time: 2s | TPS: 174.50 | Gas: 8.72%
Block #1135 | Txs:   84 | Time: 11s | TPS:   7.64 | Gas: 2.10%
Block #1136 | Txs:  656 | Time: 3s | TPS: 218.67 | Gas: 16.40%
Block #1137 | Txs:  453 | Time: 2s | TPS: 226.50 | Gas: 11.32%
Block #1138 | Txs:  152 | Time: 8s | TPS:  19.00 | Gas: 3.80%
Block #1139 | Txs: 1401 | Time: 13s | TPS: 107.77 | Gas: 35.02%
Block #1140 | Txs:    2 | Time: 15s | TPS:   0.13 | Gas: 0.05%
Block #1141 | Txs:  547 | Time: 14s | TPS:  39.07 | Gas: 13.91%


After increasing blocktime to 3 sec

Block #346 | Txs: 2315 | Time: 3s | TPS: 771.67 | Gas: 58.11%
Block #347 | Txs: 1821 | Time: 3s | TPS: 607.00 | Gas: 45.52%
Block #348 | Txs: 2019 | Time: 3s | TPS: 673.00 | Gas: 50.47%
Block #349 | Txs: 2061 | Time: 3s | TPS: 687.00 | Gas: 51.52%
Block #350 | Txs: 2203 | Time: 3s | TPS: 734.33 | Gas: 55.07%
Block #351 | Txs: 2951 | Time: 7s | TPS: 421.57 | Gas: 73.77%
Block #352 | Txs: 2429 | Time: 7s | TPS: 347.00 | Gas: 60.72%
Block #353 | Txs: 4000 | Time: 10s | TPS: 400.00 | Gas: 100.00%
Block #354 | Txs: 1322 | Time: 15s | TPS:  88.13 | Gas: 33.05%
Block #355 | Txs: 3185 | Time: 6s | TPS: 530.83 | Gas: 79.62%


Conclusions: Lightly reproducable:
Reducing blocktime to 1 sec -> gives higher TPS in general till the time network gets clogged, 
after which the blocktime increases and TPS decreases.
Increasing blocktime to 3 sec -> gives more consistent higher tps, but the network gets clogged sooner, and blocks get delayed.

In most tests, blocks did not hit block gas limits, and it is not considered as a constraint.

All tests (3nodes +1bootnode + scripts) were done on a local machine. As the TPS scripts ran- machine constantly hit CPU limits.