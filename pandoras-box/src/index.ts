#!/usr/bin/env node
import { Command } from 'commander';
import { Distributor, Runtime } from './distributor/distributor';
import TokenDistributor from './distributor/tokenDistributor';
import Logger from './logger/logger';
import Outputter from './outputter/outputter';
import { Engine, EngineContext } from './runtime/engine';
import EOARuntime from './runtime/eoa';
import ERC20Runtime from './runtime/erc20';
import ERC721Runtime from './runtime/erc721';
import RuntimeErrors from './runtime/errors';
import {
    InitializedRuntime,
    RuntimeType,
    TokenRuntime,
} from './runtime/runtimes';
import { StatCollector, TPSMonitor } from './stats/collector';

async function run() {
    const program = new Command();

    program
        .name('pandoras-box')
        .description(
            'A small and simple stress testing tool for Ethereum-compatible blockchain clients '
        )
        .version('1.0.0');

    program
        .option(
            '-url, --json-rpc <json-rpc-address...>',
            'The URL(s) of the JSON-RPC for the client (can specify multiple)'
        )
        .option(
            '-m, --mnemonic <mnemonic>',
            'The mnemonic used to generate spam accounts'
        )
        .option(
            '-s, -sub-accounts <sub-accounts>',
            'The number of sub-accounts that will send out transactions',
            '10'
        )
        .option(
            '-t, --transactions <transactions>',
            'The total number of transactions to be emitted',
            '2000'
        )
        .option(
            '--mode <mode>',
            'The mode for the stress test. Possible modes: [EOA, ERC20, ERC721]',
            'EOA'
        )
        .option(
            '-o, --output <output-path>',
            'The output path for the results JSON'
        )
        .option(
            '-b, --batch <batch>',
            'The batch size of JSON-RPC transactions',
            '20'
        )
        .option(
            '--only-tps',
            'Only monitor and display real-time TPS (no transaction sending)'
        )
        .option(
            '--monitor-tps',
            'Enable real-time TPS monitoring during transaction sending'
        )
        .parse();

    const options = program.opts();
    const onlyTPS = options.onlyTps;

    // Validate required options based on mode
    if (!options.jsonRpc) {
        Logger.error('Error: --json-rpc is required');
        process.exit(1);
    }

    const urls: string[] = Array.isArray(options.jsonRpc)
        ? options.jsonRpc
        : [options.jsonRpc];
    const primaryUrl = urls[0]; // Use first URL for setup operations

    // If --only-tps flag is set, just run the TPS monitor
    if (onlyTPS) {
        Logger.info(`Monitoring TPS on: ${primaryUrl}`);
        const monitor = new TPSMonitor(primaryUrl);
        await monitor.start();

        // Keep running until user interrupts (Ctrl+C)
        await new Promise(() => {}); // Never resolves
        return;
    }

    // For normal operation, mnemonic is required
    const mnemonic = options.mnemonic;
    if (!mnemonic) {
        Logger.error('Error: --mnemonic is required for transaction sending');
        process.exit(1);
    }

    const transactionCount = options.transactions;
    const mode = options.mode;
    const subAccountsCount = options.SubAccounts;
    const batchSize = options.batch;
    const output = options.output;
    const monitorTPS = options.monitorTps;

    Logger.info(`Using ${urls.length} RPC endpoint(s): ${urls.join(', ')}`);

    let runtime: Runtime;
    switch (mode) {
        case RuntimeType.EOA:
            runtime = new EOARuntime(mnemonic, primaryUrl);

            break;
        case RuntimeType.ERC20:
            runtime = new ERC20Runtime(mnemonic, primaryUrl);

            // Initialize the runtime
            await (runtime as InitializedRuntime).Initialize();

            break;
        case RuntimeType.ERC721:
            runtime = new ERC721Runtime(mnemonic, primaryUrl);

            // Initialize the runtime
            await (runtime as InitializedRuntime).Initialize();

            break;
        default:
            throw RuntimeErrors.errUnknownRuntime;
    }

    // Distribute the native currency funds
    const distributor = new Distributor(
        mnemonic,
        subAccountsCount,
        transactionCount,
        runtime,
        primaryUrl
    );

    const accountIndexes: number[] = await distributor.distribute();

    // Distribute the token funds, if any
    if (mode === RuntimeType.ERC20) {
        const tokenDistributor = new TokenDistributor(
            mnemonic,
            accountIndexes,
            transactionCount,
            runtime as TokenRuntime
        );

        // Start the distribution
        await tokenDistributor.distributeTokens();
    }

    // Start real-time TPS monitoring if requested
    let tpsMonitor: TPSMonitor | undefined;
    if (monitorTPS) {
        tpsMonitor = new TPSMonitor(primaryUrl);
        await tpsMonitor.start();
    }

    // Run the specific runtime with all URLs
    const txHashes = await Engine.Run(
        runtime,
        new EngineContext(
            accountIndexes,
            transactionCount,
            batchSize,
            mnemonic,
            urls
        )
    );

    // Stop TPS monitor if it was running
    if (tpsMonitor) {
        tpsMonitor.stop();
    }

    // Collect the data
    const collectorData = await new StatCollector().generateStats(
        txHashes,
        mnemonic,
        primaryUrl,
        batchSize
    );

    // Output the data if needed
    if (output) {
        Outputter.outputData(collectorData, output);
    }
}

run()
    .then()
    .catch((err) => {
        Logger.error(err);
    });
