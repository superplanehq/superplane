import { FakeFS } from './FakeFS';
import { ProxiedFS } from './ProxiedFS';
import { Path, PathUtils } from './path';
export type AliasFSOptions<P extends Path> = {
    baseFs: FakeFS<P>;
    pathUtils: PathUtils<P>;
};
export declare class AliasFS<P extends Path> extends ProxiedFS<P, P> {
    private readonly target;
    protected readonly baseFs: FakeFS<P>;
    constructor(target: P, { baseFs, pathUtils }: AliasFSOptions<P>);
    getRealPath(): P;
    getBaseFs(): FakeFS<P>;
    protected mapFromBase(p: P): P;
    protected mapToBase(p: P): P;
}
