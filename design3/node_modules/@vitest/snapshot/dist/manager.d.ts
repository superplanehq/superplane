import { S as SnapshotStateOptions, e as SnapshotSummary, b as SnapshotResult } from './index-69d272f6.js';
import 'pretty-format';
import './environment-b0891b0a.js';

declare class SnapshotManager {
    options: Omit<SnapshotStateOptions, 'snapshotEnvironment'>;
    summary: SnapshotSummary;
    resolvedPaths: Set<string>;
    extension: string;
    constructor(options: Omit<SnapshotStateOptions, 'snapshotEnvironment'>);
    clear(): void;
    add(result: SnapshotResult): void;
    resolvePath(testPath: string): string;
    resolveRawPath(testPath: string, rawPath: string): string;
}
declare function emptySummary(options: Omit<SnapshotStateOptions, 'snapshotEnvironment'>): SnapshotSummary;
declare function addSnapshotResult(summary: SnapshotSummary, result: SnapshotResult): void;

export { SnapshotManager, addSnapshotResult, emptySummary };
