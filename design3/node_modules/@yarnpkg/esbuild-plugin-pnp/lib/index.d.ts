import type { OnLoadArgs, OnLoadResult, OnResolveArgs, OnResolveResult, Plugin } from 'esbuild';
declare type OnResolveParams = {
    resolvedPath: string | null;
    watchFiles: Array<string>;
    error?: Error;
};
export declare type PluginOptions = {
    baseDir?: string;
    extensions?: Array<string>;
    filter?: RegExp;
    onResolve?: (args: OnResolveArgs, params: OnResolveParams) => Promise<OnResolveResult | null>;
    onLoad?: (args: OnLoadArgs) => Promise<OnLoadResult>;
};
export declare function pnpPlugin({ baseDir, extensions, filter, onResolve, onLoad, }?: PluginOptions): Plugin;
export {};
