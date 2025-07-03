import { NodeFS } from './NodeFS';
import { PortablePath } from './path';
export type XFS = NodeFS & {
    detachTemp(p: PortablePath): void;
    mktempSync(): PortablePath;
    mktempSync<T>(cb: (p: PortablePath) => T): T;
    mktempPromise(): Promise<PortablePath>;
    mktempPromise<T>(cb: (p: PortablePath) => Promise<T>): Promise<T>;
    /**
     * Tries to remove all temp folders created by mktempSync and mktempPromise
     */
    rmtempPromise(): Promise<void>;
    /**
     * Tries to remove all temp folders created by mktempSync and mktempPromise
     */
    rmtempSync(): void;
};
export declare const xfs: XFS;
