import { FakeFS, ExtractHintOptions } from './FakeFS';
import { ProxiedFS } from './ProxiedFS';
import { Filename, PortablePath } from './path';
export type VirtualFSOptions = {
    baseFs?: FakeFS<PortablePath>;
    folderName?: Filename;
};
export declare class VirtualFS extends ProxiedFS<PortablePath, PortablePath> {
    protected readonly baseFs: FakeFS<PortablePath>;
    static makeVirtualPath(base: PortablePath, component: Filename, to: PortablePath): PortablePath;
    static resolveVirtual(p: PortablePath): PortablePath;
    constructor({ baseFs }?: VirtualFSOptions);
    getExtractHint(hints: ExtractHintOptions): boolean;
    getRealPath(): PortablePath;
    realpathSync(p: PortablePath): PortablePath;
    realpathPromise(p: PortablePath): Promise<PortablePath>;
    mapToBase(p: PortablePath): PortablePath;
    mapFromBase(p: PortablePath): PortablePath;
}
