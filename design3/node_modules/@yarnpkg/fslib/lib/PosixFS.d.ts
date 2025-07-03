import { FakeFS } from './FakeFS';
import { ProxiedFS } from './ProxiedFS';
import { NativePath, PortablePath } from './path';
export declare class PosixFS extends ProxiedFS<NativePath, PortablePath> {
    protected readonly baseFs: FakeFS<PortablePath>;
    constructor(baseFs: FakeFS<PortablePath>);
    protected mapFromBase(path: PortablePath): NativePath;
    protected mapToBase(path: NativePath): PortablePath;
}
