import { FakeFS } from '../FakeFS';
import { Path } from '../path';
export declare enum LinkStrategy {
    Allow = "allow",
    ReadOnly = "readOnly"
}
export type CopyOptions = {
    linkStrategy: LinkStrategy | null;
    stableTime: boolean;
    stableSort: boolean;
    overwrite: boolean;
};
export type Operations = Array<() => Promise<void>>;
export type LUTimes<P extends Path> = Array<[P, Date | number, Date | number]>;
export declare function copyPromise<P1 extends Path, P2 extends Path>(destinationFs: FakeFS<P1>, destination: P1, sourceFs: FakeFS<P2>, source: P2, opts: CopyOptions): Promise<void>;
