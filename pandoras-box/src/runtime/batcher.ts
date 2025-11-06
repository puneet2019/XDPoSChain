import axios from 'axios';
import { SingleBar } from 'cli-progress';
import Logger from '../logger/logger';

class Batcher {
    // Generates batches of items based on the passed in
    // input set
    static generateBatches<ItemType>(
        items: ItemType[],
        batchSize: number
    ): ItemType[][] {
        const batches: ItemType[][] = [];

        // Find the required number of batches
        let numBatches: number = Math.ceil(items.length / batchSize);
        if (numBatches == 0) {
            numBatches = 1;
        }

        // Initialize empty batches
        for (let i = 0; i < numBatches; i++) {
            batches[i] = [];
        }

        let currentBatch = 0;
        for (const item of items) {
            batches[currentBatch].push(item);

            if (batches[currentBatch].length % batchSize == 0) {
                currentBatch++;
            }
        }

        return batches;
    }

    static async batchTransactions(
        signedTxs: string[],
        batchSize: number,
        urls: string[]
    ): Promise<string[]> {
        // Generate the transaction hash batches
        const batches: string[][] = Batcher.generateBatches<string>(
            signedTxs,
            batchSize
        );

        Logger.info(
            `Sending transactions continuously across ${urls.length} RPC endpoint(s)...`
        );

        const batchBar = new SingleBar({
            barCompleteChar: '\u2588',
            barIncompleteChar: '\u2591',
            hideCursor: true,
        });

        batchBar.start(batches.length, 0, {
            speed: 'N/A',
        });

        const txHashes: string[] = [];
        const batchErrors: string[] = [];
        let nextIndx = 0;
        let urlIndex = 0;

        // Send batches continuously without waiting for all to complete
        // Use round-robin across multiple URLs
        for (const batch of batches) {
            try {
                let singleRequests = '';
                for (let i = 0; i < batch.length; i++) {
                    singleRequests += JSON.stringify({
                        jsonrpc: '2.0',
                        method: 'eth_sendRawTransaction',
                        params: [batch[i]],
                        id: nextIndx++,
                    });

                    if (i != batch.length - 1) {
                        singleRequests += ',\n';
                    }
                }

                // Round-robin URL selection
                const currentUrl = urls[urlIndex % urls.length];
                urlIndex++;

                // Send immediately without awaiting
                axios({
                    url: currentUrl,
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    data: '[' + singleRequests + ']',
                })
                    .then((response) => {
                        const content = response.data;
                        for (const cnt of content) {
                            // eslint-disable-next-line no-prototype-builtins
                            if (cnt.hasOwnProperty('error')) {
                                batchErrors.push(cnt.error.message);
                            } else {
                                txHashes.push(cnt.result);
                            }
                        }
                    })
                    .catch((e: any) => {
                        Logger.error(
                            `Error with ${currentUrl}: ${e.message}`
                        );
                    });

                batchBar.increment();
            } catch (e: any) {
                Logger.error(e.message);
            }
        }

        batchBar.stop();

        if (batchErrors.length > 0) {
            Logger.warn('Errors encountered during batch sending:');

            for (const err of batchErrors) {
                Logger.error(err);
            }
        }

        Logger.success(
            `${batches.length} ${batches.length > 1 ? 'batches' : 'batch'} sent continuously`
        );

        return txHashes;
    }
}

export default Batcher;
