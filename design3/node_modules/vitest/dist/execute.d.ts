import vm from 'node:vm';
import { ViteNodeRunner } from 'vite-node/client';
import { ViteNodeRunnerOptions } from 'vite-node';
import { ag as PendingSuiteMock, ah as MockFactory, Z as WorkerGlobalState, ai as MockMap, K as RuntimeRPC } from './reporters-5f784f42.js';
import 'vite';
import '@vitest/runner';
import '@vitest/snapshot';
import '@vitest/expect';
import '@vitest/runner/utils';
import '@vitest/utils';
import 'tinybench';
import '@vitest/snapshot/manager';
import 'vite-node/server';
import 'node:worker_threads';
import 'rollup';
import 'node:fs';
import 'chai';

type Key = string | symbol;
declare class VitestMocker {
    executor: VitestExecutor;
    static pendingIds: PendingSuiteMock[];
    private spyModule?;
    private resolveCache;
    private primitives;
    private filterPublicKeys;
    constructor(executor: VitestExecutor);
    private get root();
    private get mockMap();
    private get moduleCache();
    private get moduleDirectories();
    initializeSpyModule(): Promise<void>;
    private deleteCachedItem;
    private isAModuleDirectory;
    getSuiteFilepath(): string;
    private createError;
    getMocks(): {
        [x: string]: string | MockFactory | null;
    };
    private resolvePath;
    resolveMocks(): Promise<void>;
    private callFunctionMock;
    getMockPath(dep: string): string;
    getDependencyMock(id: string): string | MockFactory | null;
    normalizePath(path: string): string;
    resolveMockPath(mockPath: string, external: string | null): string | null;
    mockObject(object: Record<Key, any>, mockExports?: Record<Key, any>): Record<Key, any>;
    unmockPath(path: string): void;
    mockPath(originalId: string, path: string, external: string | null, factory?: MockFactory): void;
    importActual<T>(rawId: string, importee: string): Promise<T>;
    importMock(rawId: string, importee: string): Promise<any>;
    requestWithMock(url: string, callstack: string[]): Promise<any>;
    queueMock(id: string, importer: string, factory?: MockFactory): void;
    queueUnmock(id: string, importer: string): void;
}

interface ModuleEvaluateOptions {
    timeout?: vm.RunningScriptOptions['timeout'] | undefined;
    breakOnSigint?: vm.RunningScriptOptions['breakOnSigint'] | undefined;
}
type ModuleLinker = (specifier: string, referencingModule: VMModule, extra: {
    assert: Object;
}) => VMModule | Promise<VMModule>;
type ModuleStatus = 'unlinked' | 'linking' | 'linked' | 'evaluating' | 'evaluated' | 'errored';
declare class VMModule {
    dependencySpecifiers: readonly string[];
    error: any;
    identifier: string;
    context: vm.Context;
    namespace: Object;
    status: ModuleStatus;
    evaluate(options?: ModuleEvaluateOptions): Promise<void>;
    link(linker: ModuleLinker): Promise<void>;
}

declare class FileMap {
    private fsCache;
    private fsBufferCache;
    readFile(path: string): string;
    readBuffer(path: string): Buffer;
}

interface ExternalModulesExecutorOptions extends ExecuteOptions {
    context: vm.Context;
    fileMap: FileMap;
    packageCache: Map<string, any>;
}
declare class ExternalModulesExecutor {
    private options;
    private cjs;
    private esm;
    private vite;
    private context;
    private fs;
    private resolvers;
    constructor(options: ExternalModulesExecutorOptions);
    importModuleDynamically: (specifier: string, referencer: VMModule) => Promise<VMModule>;
    resolveModule: (specifier: string, referencer: string) => Promise<VMModule>;
    resolve(specifier: string, parent: string): Promise<string>;
    private findNearestPackageData;
    private wrapCoreSynteticModule;
    private wrapCommonJsSynteticModule;
    private getModuleInformation;
    private createModule;
    import(identifier: string): Promise<Object>;
    require(identifier: string): any;
    createRequire(identifier: string): NodeRequire;
}

interface ExecuteOptions extends ViteNodeRunnerOptions {
    mockMap: MockMap;
    packageCache: Map<string, string>;
    moduleDirectories?: string[];
    context?: vm.Context;
    state: WorkerGlobalState;
    transform: RuntimeRPC['transform'];
}
declare class VitestExecutor extends ViteNodeRunner {
    options: ExecuteOptions;
    mocker: VitestMocker;
    externalModules?: ExternalModulesExecutor;
    private primitives;
    constructor(options: ExecuteOptions);
    protected getContextPrimitives(): {
        Object: ObjectConstructor;
        Reflect: typeof Reflect;
        Symbol: SymbolConstructor;
    };
    get state(): WorkerGlobalState;
    shouldResolveId(id: string, _importee?: string | undefined): boolean;
    originalResolveUrl(id: string, importer?: string): Promise<[url: string, fsPath: string]>;
    resolveUrl(id: string, importer?: string): Promise<[string, string]>;
    protected runModule(context: Record<string, any>, transformed: string): Promise<void>;
    importExternalModule(path: string): Promise<any>;
    dependencyRequest(id: string, fsPath: string, callstack: string[]): Promise<any>;
    prepareContext(context: Record<string, any>): Record<string, any>;
}

export { VitestExecutor };
