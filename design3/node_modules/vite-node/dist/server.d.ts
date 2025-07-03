import { TransformResult, ViteDevServer } from 'vite';
import { D as DebuggerOptions, c as DepsHandlingOptions, V as ViteNodeServerOptions, F as FetchResult, d as ViteNodeResolveId } from './index-6fb787b2.js';
import { E as EncodedSourceMap } from './trace-mapping.d-e677e8f4.js';

declare class Debugger {
    options: DebuggerOptions;
    dumpDir: string | undefined;
    initPromise: Promise<void> | undefined;
    externalizeMap: Map<string, string>;
    constructor(root: string, options: DebuggerOptions);
    clearDump(): Promise<void>;
    encodeId(id: string): string;
    recordExternalize(id: string, path: string): Promise<void>;
    dumpFile(id: string, result: TransformResult | null): Promise<void>;
    loadDump(id: string): Promise<TransformResult | null>;
    writeInfo(): Promise<void>;
}

declare function guessCJSversion(id: string): string | undefined;
declare function shouldExternalize(id: string, options?: DepsHandlingOptions, cache?: Map<string, Promise<string | false>>): Promise<string | false>;

declare class ViteNodeServer {
    server: ViteDevServer;
    options: ViteNodeServerOptions;
    private fetchPromiseMap;
    private transformPromiseMap;
    private existingOptimizedDeps;
    fetchCache: Map<string, {
        duration?: number | undefined;
        timestamp: number;
        result: FetchResult;
    }>;
    externalizeCache: Map<string, Promise<string | false>>;
    debugger?: Debugger;
    constructor(server: ViteDevServer, options?: ViteNodeServerOptions);
    shouldExternalize(id: string): Promise<string | false>;
    private ensureExists;
    resolveId(id: string, importer?: string, transformMode?: 'web' | 'ssr'): Promise<ViteNodeResolveId | null>;
    getSourceMap(source: string): EncodedSourceMap | null;
    fetchModule(id: string, transformMode?: 'web' | 'ssr'): Promise<FetchResult>;
    transformRequest(id: string, filepath?: string): Promise<TransformResult | null | undefined>;
    transformModule(id: string, transformMode?: 'web' | 'ssr'): Promise<{
        code: string | undefined;
    }>;
    getTransformMode(id: string): "web" | "ssr";
    private _fetchModule;
    protected processTransformResult(filepath: string, result: TransformResult): Promise<TransformResult>;
    private _transformRequest;
}

export { ViteNodeServer, guessCJSversion, shouldExternalize };
