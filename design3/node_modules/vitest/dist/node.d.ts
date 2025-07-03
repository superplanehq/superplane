import { V as VitestRunMode, U as UserConfig, b as Vitest, T as TestSequencer, W as WorkspaceSpec } from './reporters-5f784f42.js';
export { d as TestSequencerConstructor, c as VitestWorkspace, s as startVitest } from './reporters-5f784f42.js';
import { UserConfig as UserConfig$1, Plugin } from 'vite';
import '@vitest/runner';
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

declare function createVitest(mode: VitestRunMode, options: UserConfig, viteOverrides?: UserConfig$1): Promise<Vitest>;

declare function VitestPlugin(options?: UserConfig, ctx?: Vitest): Promise<Plugin[]>;

declare function registerConsoleShortcuts(ctx: Vitest): () => void;

declare class BaseSequencer implements TestSequencer {
    protected ctx: Vitest;
    constructor(ctx: Vitest);
    shard(files: WorkspaceSpec[]): Promise<WorkspaceSpec[]>;
    sort(files: WorkspaceSpec[]): Promise<WorkspaceSpec[]>;
}

export { BaseSequencer, TestSequencer, Vitest, VitestPlugin, WorkspaceSpec, createVitest, registerConsoleShortcuts };
