import { t as SequenceHooks, u as SequenceSetupFiles, F as File, a as Test, S as Suite, l as TaskResultPack, s as TestContext } from './tasks-e594cd24.js';
export { D as DoneCallback, o as Fixtures, p as HookCleanupCallback, H as HookListener, O as OnTestFailedHandler, R as RunMode, r as RuntimeContext, d as SuiteAPI, f as SuiteCollector, q as SuiteFactory, g as SuiteHooks, T as Task, i as TaskBase, b as TaskCustom, j as TaskMeta, k as TaskResult, h as TaskState, e as TestAPI, m as TestFunction, n as TestOptions } from './tasks-e594cd24.js';
import '@vitest/utils';

/**
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */


declare type CompareKeys =
  | ((a: string, b: string) => number)
  | null
  | undefined;

/**
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

type DiffOptionsColor = (arg: string) => string;
interface DiffOptions {
    aAnnotation?: string;
    aColor?: DiffOptionsColor;
    aIndicator?: string;
    bAnnotation?: string;
    bColor?: DiffOptionsColor;
    bIndicator?: string;
    changeColor?: DiffOptionsColor;
    changeLineTrailingSpaceColor?: DiffOptionsColor;
    commonColor?: DiffOptionsColor;
    commonIndicator?: string;
    commonLineTrailingSpaceColor?: DiffOptionsColor;
    contextLines?: number;
    emptyFirstOrLastLinePlaceholder?: string;
    expand?: boolean;
    includeChangeCounts?: boolean;
    omitAnnotationLines?: boolean;
    patchColor?: DiffOptionsColor;
    compareKeys?: CompareKeys;
}

interface VitestRunnerConfig {
    root: string;
    setupFiles: string[] | string;
    name: string;
    passWithNoTests: boolean;
    testNamePattern?: RegExp;
    allowOnly?: boolean;
    sequence: {
        shuffle?: boolean;
        concurrent?: boolean;
        seed: number;
        hooks: SequenceHooks;
        setupFiles: SequenceSetupFiles;
    };
    chaiConfig?: {
        truncateThreshold?: number;
    };
    maxConcurrency: number;
    testTimeout: number;
    hookTimeout: number;
    retry: number;
    diffOptions?: DiffOptions;
}
type VitestRunnerImportSource = 'collect' | 'setup';
interface VitestRunnerConstructor {
    new (config: VitestRunnerConfig): VitestRunner;
}
type CancelReason = 'keyboard-input' | 'test-failure' | string & Record<string, never>;
interface VitestRunner {
    /**
     * First thing that's getting called before actually collecting and running tests.
     */
    onBeforeCollect?(paths: string[]): unknown;
    /**
     * Called after collecting tests and before "onBeforeRun".
     */
    onCollected?(files: File[]): unknown;
    /**
     * Called when test runner should cancel next test runs.
     * Runner should listen for this method and mark tests and suites as skipped in
     * "onBeforeRunSuite" and "onBeforeRunTest" when called.
     */
    onCancel?(reason: CancelReason): unknown;
    /**
     * Called before running a single test. Doesn't have "result" yet.
     */
    onBeforeRunTest?(test: Test): unknown;
    /**
     * Called before actually running the test function. Already has "result" with "state" and "startTime".
     */
    onBeforeTryTest?(test: Test, options: {
        retry: number;
        repeats: number;
    }): unknown;
    /**
     * Called after result and state are set.
     */
    onAfterRunTest?(test: Test): unknown;
    /**
     * Called right after running the test function. Doesn't have new state yet. Will not be called, if the test function throws.
     */
    onAfterTryTest?(test: Test, options: {
        retry: number;
        repeats: number;
    }): unknown;
    /**
     * Called before running a single suite. Doesn't have "result" yet.
     */
    onBeforeRunSuite?(suite: Suite): unknown;
    /**
     * Called after running a single suite. Has state and result.
     */
    onAfterRunSuite?(suite: Suite): unknown;
    /**
     * If defined, will be called instead of usual Vitest suite partition and handling.
     * "before" and "after" hooks will not be ignored.
     */
    runSuite?(suite: Suite): Promise<void>;
    /**
     * If defined, will be called instead of usual Vitest handling. Useful, if you have your custom test function.
     * "before" and "after" hooks will not be ignored.
     */
    runTest?(test: Test): Promise<void>;
    /**
     * Called, when a task is updated. The same as "onTaskUpdate" in a reporter, but this is running in the same thread as tests.
     */
    onTaskUpdate?(task: TaskResultPack[]): Promise<void>;
    /**
     * Called before running all tests in collected paths.
     */
    onBeforeRun?(files: File[]): unknown;
    /**
     * Called right after running all tests in collected paths.
     */
    onAfterRun?(files: File[]): unknown;
    /**
     * Called when new context for a test is defined. Useful, if you want to add custom properties to the context.
     * If you only want to define custom context, consider using "beforeAll" in "setupFiles" instead.
     */
    extendTestContext?(context: TestContext): TestContext;
    /**
     * Called, when files are imported. Can be called in two situations: when collecting tests and when importing setup files.
     */
    importFile(filepath: string, source: VitestRunnerImportSource): unknown;
    /**
     * Publicly available configuration.
     */
    config: VitestRunnerConfig;
}

export { CancelReason, File, SequenceHooks, SequenceSetupFiles, Suite, TaskResultPack, Test, TestContext, VitestRunner, VitestRunnerConfig, VitestRunnerConstructor, VitestRunnerImportSource };
