import { FakeFS } from './FakeFS';
import { ProxiedFS } from './ProxiedFS';
import { Path, PathUtils } from './path';
export type LazyFSFactory<P extends Path> = () => FakeFS<P>;
export declare class LazyFS<P extends Path> extends ProxiedFS<P, P> {
    private readonly factory;
    private instance;
    constructor(factory: LazyFSFactory<P>, pathUtils: PathUtils<P>);
    protected get baseFs(): FakeFS<P>;
    protected set baseFs(value: FakeFS<P>);
    protected mapFromBase(p: P): P;
    protected mapToBase(p: P): P;
}
