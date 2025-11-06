import { TransactionRequest } from '@ethersproject/providers';
import Logger from '../logger/logger';
import Batcher from './batcher';
import { Runtime } from './runtimes';
import { senderAccount, Signer } from './signer';

class EngineContext {
    accountIndexes: number[];
    numTxs: number;
    batchSize: number;

    mnemonic: string;
    urls: string[];

    constructor(
        accountIndexes: number[],
        numTxs: number,
        batchSize: number,
        mnemonic: string,
        urls: string[]
    ) {
        this.accountIndexes = accountIndexes;
        this.numTxs = numTxs;
        this.batchSize = batchSize;

        this.mnemonic = mnemonic;
        this.urls = urls;
    }
}

class Engine {
    static async Run(runtime: Runtime, ctx: EngineContext): Promise<string[]> {
        // Initialize transaction signer (use first URL for signing)
        const signer: Signer = new Signer(ctx.mnemonic, ctx.urls[0]);

        // Get the account metadata
        const accounts: senderAccount[] = await signer.getSenderAccounts(
            ctx.accountIndexes,
            ctx.numTxs
        );

        // Construct the transactions
        const rawTransactions: TransactionRequest[] =
            await runtime.ConstructTransactions(accounts, ctx.numTxs);

        // Sign the transactions
        const signedTransactions = await signer.signTransactions(
            accounts,
            rawTransactions
        );

        Logger.title(runtime.GetStartMessage());

        // Send the transactions in batches across multiple URLs
        return Batcher.batchTransactions(
            signedTransactions,
            ctx.batchSize,
            ctx.urls
        );
    }
}

export { Engine, EngineContext };
