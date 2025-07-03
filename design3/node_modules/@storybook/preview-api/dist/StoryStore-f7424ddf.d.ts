import * as _storybook_types from '@storybook/types';
import { StoryId, StoryName, ComponentTitle, StoryIndex, IndexEntry, Path, Args, PreparedStory, Globals, GlobalTypes, Renderer, ModuleExports, CSFFile, NormalizedStoryAnnotations, NormalizedComponentAnnotations, NormalizedProjectAnnotations, ModuleExport, PreparedMeta, StoryContextForLoaders, ModuleImportFn, ProjectAnnotations, StoryContextForEnhancers, Parameters, StoryIndexV3, API_PreparedStoryIndex, BoundStory } from '@storybook/types';
import { SynchronousPromise } from 'synchronous-promise';
import { H as HooksContext } from './hooks-655fa363.js';

type StorySpecifier = StoryId | {
    name: StoryName;
    title: ComponentTitle;
} | '*';
declare class StoryIndexStore {
    entries: StoryIndex['entries'];
    constructor({ entries }?: StoryIndex);
    entryFromSpecifier(specifier: StorySpecifier): IndexEntry | undefined;
    storyIdToEntry(storyId: StoryId): IndexEntry;
    importPathToEntry(importPath: Path): IndexEntry;
}

declare class ArgsStore {
    initialArgsByStoryId: Record<StoryId, Args>;
    argsByStoryId: Record<StoryId, Args>;
    get(storyId: StoryId): Args;
    setInitial(story: PreparedStory<any>): void;
    updateFromDelta(story: PreparedStory<any>, delta: Args): void;
    updateFromPersisted(story: PreparedStory<any>, persisted: Args): void;
    update(storyId: StoryId, argsUpdate: Partial<Args>): void;
}

declare class GlobalsStore {
    allowedGlobalNames: Set<string>;
    initialGlobals: Globals;
    globals: Globals;
    constructor({ globals, globalTypes, }: {
        globals?: Globals;
        globalTypes?: GlobalTypes;
    });
    set({ globals, globalTypes }: {
        globals?: Globals;
        globalTypes?: GlobalTypes;
    }): void;
    filterAllowedGlobals(globals: Globals): Globals;
    updateFromPersisted(persisted: Globals): void;
    get(): Globals;
    update(newGlobals: Globals): void;
}

declare function processCSFFile<TRenderer extends Renderer>(moduleExports: ModuleExports, importPath: Path, title: ComponentTitle): CSFFile<TRenderer>;

declare function prepareStory<TRenderer extends Renderer>(storyAnnotations: NormalizedStoryAnnotations<TRenderer>, componentAnnotations: NormalizedComponentAnnotations<TRenderer>, projectAnnotations: NormalizedProjectAnnotations<TRenderer>): PreparedStory<TRenderer>;
declare function prepareMeta<TRenderer extends Renderer>(componentAnnotations: NormalizedComponentAnnotations<TRenderer>, projectAnnotations: NormalizedProjectAnnotations<TRenderer>, moduleExport: ModuleExport): PreparedMeta<TRenderer>;
declare function prepareContext<TRenderer extends Renderer, TContext extends Pick<StoryContextForLoaders<TRenderer>, 'args' | 'argTypes' | 'globals'>>(context: TContext): TContext & Pick<StoryContextForLoaders<TRenderer>, 'allArgs' | 'argsByTarget' | 'unmappedArgs'>;

declare class StoryStore<TRenderer extends Renderer> {
    storyIndex?: StoryIndexStore;
    importFn?: ModuleImportFn;
    projectAnnotations?: NormalizedProjectAnnotations<TRenderer>;
    globals?: GlobalsStore;
    args: ArgsStore;
    hooks: Record<StoryId, HooksContext<TRenderer>>;
    cachedCSFFiles?: Record<Path, CSFFile<TRenderer>>;
    processCSFFileWithCache: typeof processCSFFile;
    prepareMetaWithCache: typeof prepareMeta;
    prepareStoryWithCache: typeof prepareStory;
    initializationPromise: SynchronousPromise<void>;
    resolveInitializationPromise: () => void;
    constructor();
    setProjectAnnotations(projectAnnotations: ProjectAnnotations<TRenderer>): void;
    initialize({ storyIndex, importFn, cache, }: {
        storyIndex?: StoryIndex;
        importFn: ModuleImportFn;
        cache?: boolean;
    }): Promise<void>;
    onStoriesChanged({ importFn, storyIndex, }: {
        importFn?: ModuleImportFn;
        storyIndex?: StoryIndex;
    }): Promise<void>;
    storyIdToEntry(storyId: StoryId): Promise<IndexEntry>;
    loadCSFFileByStoryId(storyId: StoryId): Promise<CSFFile<TRenderer>>;
    loadAllCSFFiles({ batchSize }?: {
        batchSize?: number | undefined;
    }): Promise<StoryStore<TRenderer>['cachedCSFFiles']>;
    cacheAllCSFFiles(): Promise<void>;
    preparedMetaFromCSFFile({ csfFile }: {
        csfFile: CSFFile<TRenderer>;
    }): PreparedMeta<TRenderer>;
    loadStory({ storyId }: {
        storyId: StoryId;
    }): Promise<PreparedStory<TRenderer>>;
    storyFromCSFFile({ storyId, csfFile, }: {
        storyId: StoryId;
        csfFile: CSFFile<TRenderer>;
    }): PreparedStory<TRenderer>;
    componentStoriesFromCSFFile({ csfFile, }: {
        csfFile: CSFFile<TRenderer>;
    }): PreparedStory<TRenderer>[];
    loadEntry(id: StoryId): Promise<{
        entryExports: ModuleExports;
        csfFiles: CSFFile<TRenderer>[];
    }>;
    getStoryContext(story: PreparedStory<TRenderer>, { forceInitialArgs }?: {
        forceInitialArgs?: boolean | undefined;
    }): Omit<StoryContextForLoaders, 'viewMode'>;
    cleanupStory(story: PreparedStory<TRenderer>): void;
    extract(options?: {
        includeDocsOnly?: boolean;
    }): Record<StoryId, StoryContextForEnhancers<TRenderer>>;
    getSetStoriesPayload(): {
        v: number;
        globals: _storybook_types.Globals;
        globalParameters: {};
        kindParameters: Parameters;
        stories: Record<string, StoryContextForEnhancers<TRenderer, _storybook_types.Args>>;
    };
    getStoriesJsonData: () => StoryIndexV3;
    getSetIndexPayload(): API_PreparedStoryIndex;
    raw(): BoundStory<TRenderer>[];
    fromId(storyId: StoryId): BoundStory<TRenderer> | null;
}

export { StoryStore as S, prepareMeta as a, processCSFFile as b, prepareContext as c, StorySpecifier as d, prepareStory as p };
