import { BigNumber } from '@ethersproject/bignumber';
import { JsonRpcProvider, Provider } from '@ethersproject/providers';
import { SingleBar } from 'cli-progress';
import Table from 'cli-table3';
import Logger from '../logger/logger';

class BlockInfo {
    blockNum: number;
    createdAt: number;
    numTxs: number;

    gasUsed: string;
    gasLimit: string;
    gasUtilization: number;

    constructor(
        blockNum: number,
        createdAt: number,
        numTxs: number,
        gasUsed: BigNumber,
        gasLimit: BigNumber
    ) {
        this.blockNum = blockNum;
        this.createdAt = createdAt;
        this.numTxs = numTxs;
        this.gasUsed = gasUsed.toHexString();
        this.gasLimit = gasLimit.toHexString();

        const largeDivision = gasUsed
            .mul(BigNumber.from(10000))
            .div(gasLimit)
            .toNumber();

        this.gasUtilization = largeDivision / 100;
    }
}

class CollectorData {
    tps: number;
    blockInfo: Map<number, BlockInfo>;

    constructor(tps: number, blockInfo: Map<number, BlockInfo>) {
        this.tps = tps;
        this.blockInfo = blockInfo;
    }
}


class StatCollector {

    printBlockData(blockInfoMap: Map<number, BlockInfo>) {
        Logger.info('\nBlock utilization data:');
        const utilizationTable = new Table({
            head: [
                'Block #',
                'Gas Used [wei]',
                'Gas Limit [wei]',
                'Transactions',
                'Utilization',
            ],
        });

        const sortedMap = new Map(
            [...blockInfoMap.entries()].sort((a, b) => a[0] - b[0])
        );

        sortedMap.forEach((info) => {
            utilizationTable.push([
                info.blockNum,
                info.gasUsed,
                info.gasLimit,
                info.numTxs,
                `${info.gasUtilization}%`,
            ]);
        });

        Logger.info(utilizationTable.toString());
    }

    printFinalData(tps: number, blockInfoMap: Map<number, BlockInfo>) {
        // Find average utilization
        let totalUtilization = 0;
        blockInfoMap.forEach((info) => {
            totalUtilization += info.gasUtilization;
        });
        const avgUtilization = totalUtilization / blockInfoMap.size;

        const finalDataTable = new Table({
            head: ['TPS', 'Blocks', 'Avg. Utilization'],
        });

        finalDataTable.push([
            tps,
            blockInfoMap.size,
            `${avgUtilization.toFixed(2)}%`,
        ]);

        Logger.info(finalDataTable.toString());
    }

    async generateStats(
        txHashes: string[],
        mnemonic: string,
        url: string,
        batchSize: number
    ): Promise<CollectorData> {
        if (txHashes.length == 0) {
            Logger.warn('No stat data to display');

            return new CollectorData(0, new Map());
        }

        Logger.title('\n⏱ Statistics calculation initialized ⏱\n');

        const provider = new JsonRpcProvider(url);

        // Wait for transactions to be mined - give it some time
        Logger.info('Waiting for transactions to be mined...');
        await new Promise((resolve) => setTimeout(resolve, 5000));

        // Get the current block number
        const currentBlock = await provider.getBlockNumber();
        Logger.info(`Current block: ${currentBlock}`);

        // Fetch block info and calculate TPS directly from blocks
        const { blockInfoMap, avgTPS } = await this.fetchBlocksAndCalculateTPS(
            provider,
            currentBlock,
            10 // Look back 10 blocks by default
        );

        // Print the block utilization data
        this.printBlockData(blockInfoMap);

        // Print the final TPS and avg. utilization data
        this.printFinalData(avgTPS, blockInfoMap);

        return new CollectorData(avgTPS, blockInfoMap);
    }

    async fetchBlocksAndCalculateTPS(
        provider: Provider,
        currentBlock: number,
        lookbackBlocks: number
    ): Promise<{ blockInfoMap: Map<number, BlockInfo>; avgTPS: number }> {
        Logger.info('\nFetching blocks and calculating TPS...');

        const startBlock = Math.max(1, currentBlock - lookbackBlocks);
        const blocksBar = new SingleBar({
            barCompleteChar: '\u2588',
            barIncompleteChar: '\u2591',
            hideCursor: true,
        });

        const blocksToFetch = currentBlock - startBlock + 1;
        blocksBar.start(blocksToFetch, 0, {
            speed: 'N/A',
        });

        const blocksMap: Map<number, BlockInfo> = new Map<number, BlockInfo>();
        let totalTxs = 0;
        let totalTime = 0;

        for (let blockNum = startBlock; blockNum <= currentBlock; blockNum++) {
            try {
                const fetchedInfo = await provider.getBlock(blockNum);
                blocksBar.increment();

                blocksMap.set(
                    blockNum,
                    new BlockInfo(
                        blockNum,
                        fetchedInfo.timestamp,
                        fetchedInfo.transactions.length,
                        fetchedInfo.gasUsed,
                        fetchedInfo.gasLimit
                    )
                );

                totalTxs += fetchedInfo.transactions.length;

                // Calculate time difference with previous block
                if (blockNum > startBlock) {
                    const prevBlock = await provider.getBlock(blockNum - 1);
                    totalTime += Math.abs(
                        fetchedInfo.timestamp - prevBlock.timestamp
                    );
                }
            } catch (e: any) {
                Logger.error(`Error fetching block ${blockNum}: ${e.message}`);
            }
        }

        blocksBar.stop();
        Logger.success('Fetched block info and calculated TPS');

        const avgTPS = totalTime > 0 ? Math.ceil(totalTxs / totalTime) : 0;

        return { blockInfoMap: blocksMap, avgTPS };
    }
}

class TPSMonitor {
    provider: Provider;
    running: boolean;
    lastBlockNum: number;
    lastBlockTime: number;

    constructor(url: string) {
        this.provider = new JsonRpcProvider(url);
        this.running = false;
        this.lastBlockNum = 0;
        this.lastBlockTime = 0;
    }

    async start() {
        this.running = true;
        Logger.title('\n📊 Real-time TPS Monitor Started 📊\n');

        // Get initial block
        this.lastBlockNum = await this.provider.getBlockNumber();
        const lastBlock = await this.provider.getBlock(this.lastBlockNum);
        this.lastBlockTime = lastBlock.timestamp;

        // Listen for new blocks
        this.provider.on('block', async (blockNumber: number) => {
            if (!this.running) return;

            try {
                const block = await this.provider.getBlock(blockNumber);
                const timeDiff = block.timestamp - this.lastBlockTime;
                const txCount = block.transactions.length;
                const tps = timeDiff > 0 ? (txCount / timeDiff).toFixed(2) : '0';

                const gasUtil =
                    (block.gasUsed.mul(10000).div(block.gasLimit).toNumber() /
                        100).toFixed(2);

                Logger.info(
                    `Block #${blockNumber} | Txs: ${txCount.toString().padStart(4)} | ` +
                        `Time: ${timeDiff}s | TPS: ${tps.padStart(6)} | ` +
                        `Gas: ${gasUtil}%`
                );

                this.lastBlockNum = blockNumber;
                this.lastBlockTime = block.timestamp;
            } catch (e: any) {
                Logger.error(`Error monitoring block: ${e.message}`);
            }
        });
    }

    stop() {
        this.running = false;
        this.provider.removeAllListeners('block');
        Logger.success('\n📊 TPS Monitor Stopped 📊\n');
    }
}

export { StatCollector, CollectorData, BlockInfo, TPSMonitor };
