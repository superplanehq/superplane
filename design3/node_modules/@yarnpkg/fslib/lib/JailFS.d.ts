import { FakeFS } from './FakeFS';
import { ProxiedFS } from './ProxiedFS';
import { PortablePath } from './path';
export type JailFSOptions = {
    baseFs?: FakeFS<PortablePath>;
};
export declare class JailFS extends ProxiedFS<PortablePath, PortablePath> {
    private readonly target;
    protected readonly baseFs: FakeFS<PortablePath>;
    constructor(target: PortablePath, { baseFs }?: JailFSOptions);
    getRealPath(): PortablePath;
    getTarget(): PortablePath;
    getBaseFs(): FakeFS<PortablePath>;
    protected mapToBase(p: PortablePath): PortablePath;
    protected mapFromBase(p: PortablePath): PortablePath;
}
