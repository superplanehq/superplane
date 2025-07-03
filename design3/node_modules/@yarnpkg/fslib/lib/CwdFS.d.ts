import { FakeFS } from './FakeFS';
import { ProxiedFS } from './ProxiedFS';
import { PortablePath } from './path';
export type CwdFSOptions = {
    baseFs?: FakeFS<PortablePath>;
};
export declare class CwdFS extends ProxiedFS<PortablePath, PortablePath> {
    private readonly target;
    protected readonly baseFs: FakeFS<PortablePath>;
    constructor(target: PortablePath, { baseFs }?: CwdFSOptions);
    getRealPath(): PortablePath;
    resolve(p: PortablePath): PortablePath;
    mapFromBase(path: PortablePath): PortablePath;
    mapToBase(path: PortablePath): PortablePath;
}
