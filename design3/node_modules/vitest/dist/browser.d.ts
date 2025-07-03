export { startTests } from '@vitest/runner';
import { R as ResolvedConfig, e as CoverageOptions, f as CoverageProvider, g as CoverageProviderModule } from './reporters-5f784f42.js';
import { VitestExecutor } from './execute.js';
import 'vite';
import 'vite-node';
import '@vitest/snapshot';
import '@vitest/expect';
import '@vitest/runner/utils';
import '@vitest/utils';
import 'tinybench';
import 'vite-node/client';
import '@vitest/snapshot/manager';
import 'vite-node/server';
import 'node:worker_threads';
import 'rollup';
import 'node:fs';
import 'chai';
import 'node:vm';

type Formatter = (input: string | number | null | undefined) => string;

interface DiffOptions {
    aAnnotation?: string;
    aColor?: Formatter;
    aIndicator?: string;
    bAnnotation?: string;
    bColor?: Formatter;
    bIndicator?: string;
    changeColor?: Formatter;
    changeLineTrailingSpaceColor?: Formatter;
    commonColor?: Formatter;
    commonIndicator?: string;
    commonLineTrailingSpaceColor?: Formatter;
    contextLines?: number;
    emptyFirstOrLastLinePlaceholder?: string;
    expand?: boolean;
    includeChangeCounts?: boolean;
    omitAnnotationLines?: boolean;
    patchColor?: Formatter;
    compareKeys?: any;
}

declare function setupCommonEnv(config: ResolvedConfig): Promise<void>;
declare function loadDiffConfig(config: ResolvedConfig, executor: VitestExecutor): Promise<DiffOptions | undefined>;

interface Loader {
    executeId: (id: string) => Promise<{
        default: CoverageProviderModule;
    }>;
}
declare function getCoverageProvider(options: CoverageOptions | undefined, loader: Loader): Promise<CoverageProvider | null>;
declare function startCoverageInsideWorker(options: CoverageOptions | undefined, loader: Loader): Promise<unknown>;
declare function takeCoverageInsideWorker(options: CoverageOptions | undefined, loader: Loader): Promise<unknown>;
declare function stopCoverageInsideWorker(options: CoverageOptions | undefined, loader: Loader): Promise<unknown>;

export { getCoverageProvider, loadDiffConfig, setupCommonEnv, startCoverageInsideWorker, stopCoverageInsideWorker, takeCoverageInsideWorker };
