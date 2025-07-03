import * as _storybook_types from '@storybook/types';
import { Renderer, StoryId, StoryRenderOptions, ViewMode, PreparedStory, RenderToCanvas, RenderContextCallbacks, DocsContextProps, CSFFile, ModuleExports, ResolvedModuleExportType, ModuleExport, ResolvedModuleExportFromType, StoryName, StoryContextForLoaders, IndexEntry, StoryIndex, ModuleImportFn, ProjectAnnotations, Globals, Args } from '@storybook/types';
export { DocsContextProps, DocsRenderFunction, ProjectAnnotations as WebProjectAnnotations } from '@storybook/types';
export { c as composeConfigs } from './composeConfigs-62a04721.js';
import { Channel } from '@storybook/channels';
import { S as StoryStore, d as StorySpecifier } from './StoryStore-f7424ddf.js';
import 'synchronous-promise';
import './hooks-655fa363.js';

type RenderType = 'story' | 'docs';
/**
 * A "Render" represents the rendering of a single entry to a single location
 *
 * The implemenations of render are used for two key purposes:
 *  - Tracking the state of the rendering as it moves between preparing, rendering and tearing down.
 *  - Tracking what is rendered to know if a change requires re-rendering or teardown + recreation.
 */
interface Render<TRenderer extends Renderer> {
    type: RenderType;
    id: StoryId;
    isPreparing: () => boolean;
    isEqual: (other: Render<TRenderer>) => boolean;
    disableKeyListeners: boolean;
    teardown?: (options: {
        viewModeChanged: boolean;
    }) => Promise<void>;
    torndown: boolean;
    renderToElement: (canvasElement: TRenderer['canvasElement'], renderStoryToElement?: any, options?: StoryRenderOptions) => Promise<void>;
}

type RenderPhase = 'preparing' | 'loading' | 'rendering' | 'playing' | 'played' | 'completed' | 'aborted' | 'errored';
declare class StoryRender<TRenderer extends Renderer> implements Render<TRenderer> {
    channel: Channel;
    store: StoryStore<TRenderer>;
    private renderToScreen;
    private callbacks;
    id: StoryId;
    viewMode: ViewMode;
    renderOptions: StoryRenderOptions;
    type: RenderType;
    story?: PreparedStory<TRenderer>;
    phase?: RenderPhase;
    private abortController?;
    private canvasElement?;
    private notYetRendered;
    disableKeyListeners: boolean;
    private teardownRender;
    torndown: boolean;
    constructor(channel: Channel, store: StoryStore<TRenderer>, renderToScreen: RenderToCanvas<TRenderer>, callbacks: RenderContextCallbacks<TRenderer>, id: StoryId, viewMode: ViewMode, renderOptions?: StoryRenderOptions, story?: PreparedStory<TRenderer>);
    private runPhase;
    prepare(): Promise<void>;
    isEqual(other: Render<TRenderer>): boolean;
    isPreparing(): boolean;
    isPending(): boolean;
    renderToElement(canvasElement: TRenderer['canvasElement']): Promise<void>;
    private storyContext;
    render({ initial, forceRemount, }?: {
        initial?: boolean;
        forceRemount?: boolean;
    }): Promise<void>;
    rerender(): Promise<void>;
    remount(): Promise<void>;
    cancelRender(): void;
    teardown(): Promise<void>;
}

declare class DocsContext<TRenderer extends Renderer> implements DocsContextProps<TRenderer> {
    channel: Channel;
    protected store: StoryStore<TRenderer>;
    renderStoryToElement: DocsContextProps<TRenderer>['renderStoryToElement'];
    private componentStoriesValue;
    private storyIdToCSFFile;
    private exportToStory;
    private exportsToCSFFile;
    private nameToStoryId;
    private attachedCSFFile?;
    private primaryStory?;
    constructor(channel: Channel, store: StoryStore<TRenderer>, renderStoryToElement: DocsContextProps<TRenderer>['renderStoryToElement'], 
    /** The CSF files known (via the index) to be refererenced by this docs file */
    csfFiles: CSFFile<TRenderer>[]);
    referenceCSFFile(csfFile: CSFFile<TRenderer>): void;
    attachCSFFile(csfFile: CSFFile<TRenderer>): void;
    referenceMeta(metaExports: ModuleExports, attach: boolean): void;
    get projectAnnotations(): _storybook_types.NormalizedProjectAnnotations<TRenderer>;
    private resolveAttachedModuleExportType;
    private resolveModuleExport;
    resolveOf<TType extends ResolvedModuleExportType>(moduleExportOrType: ModuleExport | TType, validTypes?: TType[]): ResolvedModuleExportFromType<TType, TRenderer>;
    storyIdByName: (storyName: StoryName) => string;
    componentStories: () => PreparedStory<TRenderer>[];
    storyById: (storyId?: StoryId) => PreparedStory<TRenderer>;
    getStoryContext: (story: PreparedStory<TRenderer>) => StoryContextForLoaders<TRenderer, _storybook_types.Args>;
    loadStory: (id: StoryId) => Promise<PreparedStory<TRenderer>>;
}

/**
 * A CsfDocsRender is a render of a docs entry that is rendered based on a CSF file.
 *
 * The expectation is the primary CSF file which is the `importPath` for the entry will
 * define a story which may contain the actual rendered JSX code for the template in the
 * `docs.page` parameter.
 *
 * Use cases:
 *  - Autodocs, where there is no story, and we fall back to the globally defined template.
 *  - *.stories.mdx files, where the MDX compiler produces a CSF file with a `.parameter.docs.page`
 *      parameter containing the compiled content of the MDX file.
 */
declare class CsfDocsRender<TRenderer extends Renderer> implements Render<TRenderer> {
    protected channel: Channel;
    protected store: StoryStore<TRenderer>;
    entry: IndexEntry;
    private callbacks;
    readonly type: RenderType;
    readonly subtype = "csf";
    readonly id: StoryId;
    story?: PreparedStory<TRenderer>;
    rerender?: () => Promise<void>;
    teardownRender?: (options: {
        viewModeChanged?: boolean;
    }) => Promise<void>;
    torndown: boolean;
    readonly disableKeyListeners = false;
    preparing: boolean;
    csfFiles?: CSFFile<TRenderer>[];
    constructor(channel: Channel, store: StoryStore<TRenderer>, entry: IndexEntry, callbacks: RenderContextCallbacks<TRenderer>);
    isPreparing(): boolean;
    prepare(): Promise<void>;
    isEqual(other: Render<TRenderer>): boolean;
    docsContext(renderStoryToElement: DocsContextProps<TRenderer>['renderStoryToElement']): DocsContext<TRenderer>;
    renderToElement(canvasElement: TRenderer['canvasElement'], renderStoryToElement: DocsContextProps<TRenderer>['renderStoryToElement']): Promise<void>;
    teardown({ viewModeChanged }?: {
        viewModeChanged?: boolean;
    }): Promise<void>;
}

/**
 * A MdxDocsRender is a render of a docs entry that comes from a true MDX file,
 * that is a `.mdx` file that doesn't get compiled to a CSF file.
 *
 * A MDX render can reference (import) zero or more CSF files that contain stories.
 *
 * Use cases:
 *  - *.mdx file that may or may not reference a specific CSF file with `<Meta of={} />`
 */
declare class MdxDocsRender<TRenderer extends Renderer> implements Render<TRenderer> {
    protected channel: Channel;
    protected store: StoryStore<TRenderer>;
    entry: IndexEntry;
    private callbacks;
    readonly type: RenderType;
    readonly subtype = "mdx";
    readonly id: StoryId;
    private exports?;
    rerender?: () => Promise<void>;
    teardownRender?: (options: {
        viewModeChanged?: boolean;
    }) => Promise<void>;
    torndown: boolean;
    readonly disableKeyListeners = false;
    preparing: boolean;
    csfFiles?: CSFFile<TRenderer>[];
    constructor(channel: Channel, store: StoryStore<TRenderer>, entry: IndexEntry, callbacks: RenderContextCallbacks<TRenderer>);
    isPreparing(): boolean;
    prepare(): Promise<void>;
    isEqual(other: Render<TRenderer>): boolean;
    docsContext(renderStoryToElement: DocsContextProps<TRenderer>['renderStoryToElement']): DocsContext<TRenderer>;
    renderToElement(canvasElement: TRenderer['canvasElement'], renderStoryToElement: DocsContextProps<TRenderer>['renderStoryToElement']): Promise<void>;
    teardown({ viewModeChanged }?: {
        viewModeChanged?: boolean;
    }): Promise<void>;
}

type MaybePromise<T> = Promise<T> | T;
declare class Preview<TRenderer extends Renderer> {
    protected channel: Channel;
    /**
     * @deprecated will be removed in 8.0, please use channel instead
     */
    serverChannel?: Channel;
    storyStore: StoryStore<TRenderer>;
    getStoryIndex?: () => StoryIndex;
    importFn?: ModuleImportFn;
    renderToCanvas?: RenderToCanvas<TRenderer>;
    storyRenders: StoryRender<TRenderer>[];
    previewEntryError?: Error;
    constructor(channel?: Channel);
    initialize({ getStoryIndex, importFn, getProjectAnnotations, }: {
        getStoryIndex?: () => StoryIndex;
        importFn: ModuleImportFn;
        getProjectAnnotations: () => MaybePromise<ProjectAnnotations<TRenderer>>;
    }): Promise<void>;
    setupListeners(): void;
    getProjectAnnotationsOrRenderError(getProjectAnnotations: () => MaybePromise<ProjectAnnotations<TRenderer>>): Promise<ProjectAnnotations<TRenderer>>;
    initializeWithProjectAnnotations(projectAnnotations: ProjectAnnotations<TRenderer>): Promise<void>;
    setInitialGlobals(): Promise<void>;
    emitGlobals(): void;
    getStoryIndexFromServer(): Promise<StoryIndex>;
    initializeWithStoryIndex(storyIndex: StoryIndex): PromiseLike<void>;
    onGetProjectAnnotationsChanged({ getProjectAnnotations, }: {
        getProjectAnnotations: () => MaybePromise<ProjectAnnotations<TRenderer>>;
    }): Promise<void>;
    onStoryIndexChanged(): Promise<void>;
    onStoriesChanged({ importFn, storyIndex, }: {
        importFn?: ModuleImportFn;
        storyIndex?: StoryIndex;
    }): Promise<void>;
    onUpdateGlobals({ globals }: {
        globals: Globals;
    }): Promise<void>;
    onUpdateArgs({ storyId, updatedArgs }: {
        storyId: StoryId;
        updatedArgs: Args;
    }): Promise<void>;
    onResetArgs({ storyId, argNames }: {
        storyId: string;
        argNames?: string[];
    }): Promise<void>;
    onForceReRender(): Promise<void>;
    onForceRemount({ storyId }: {
        storyId: StoryId;
    }): Promise<void>;
    renderStoryToElement(story: PreparedStory<TRenderer>, element: TRenderer['canvasElement'], callbacks: RenderContextCallbacks<TRenderer>, options: StoryRenderOptions): () => Promise<void>;
    teardownRender(render: StoryRender<TRenderer> | CsfDocsRender<TRenderer> | MdxDocsRender<TRenderer>, { viewModeChanged }?: {
        viewModeChanged?: boolean;
    }): Promise<void>;
    extract(options?: {
        includeDocsOnly: boolean;
    }): Promise<Record<string, _storybook_types.StoryContextForEnhancers<TRenderer, Args>>>;
    renderPreviewEntryError(reason: string, err: Error): void;
}

interface SelectionSpecifier {
    storySpecifier: StorySpecifier;
    viewMode: ViewMode;
    args?: Args;
    globals?: Args;
}
interface Selection {
    storyId: StoryId;
    viewMode: ViewMode;
}
interface SelectionStore {
    selectionSpecifier: SelectionSpecifier | null;
    selection?: Selection;
    setSelection(selection: Selection): void;
    setQueryParams(queryParams: qs.ParsedQs): void;
}

interface View<TStorybookRoot> {
    prepareForStory(story: PreparedStory<any>): TStorybookRoot;
    prepareForDocs(): TStorybookRoot;
    showErrorDisplay(err: {
        message?: string;
        stack?: string;
    }): void;
    showNoPreview(): void;
    showPreparingStory(options?: {
        immediate: boolean;
    }): void;
    showPreparingDocs(options?: {
        immediate: boolean;
    }): void;
    showMain(): void;
    showDocs(): void;
    showStory(): void;
    showStoryDuringRender(): void;
}

type PossibleRender<TFramework extends Renderer> = StoryRender<TFramework> | CsfDocsRender<TFramework> | MdxDocsRender<TFramework>;
declare class PreviewWithSelection<TFramework extends Renderer> extends Preview<TFramework> {
    selectionStore: SelectionStore;
    view: View<TFramework['canvasElement']>;
    currentSelection?: Selection;
    currentRender?: PossibleRender<TFramework>;
    constructor(selectionStore: SelectionStore, view: View<TFramework['canvasElement']>);
    setupListeners(): void;
    setInitialGlobals(): Promise<void>;
    initializeWithStoryIndex(storyIndex: StoryIndex): PromiseLike<void>;
    selectSpecifiedStory(): Promise<void>;
    onGetProjectAnnotationsChanged({ getProjectAnnotations, }: {
        getProjectAnnotations: () => MaybePromise<ProjectAnnotations<TFramework>>;
    }): Promise<void>;
    onStoriesChanged({ importFn, storyIndex, }: {
        importFn?: ModuleImportFn;
        storyIndex?: StoryIndex;
    }): Promise<void>;
    onKeydown(event: KeyboardEvent): void;
    onSetCurrentStory(selection: {
        storyId: StoryId;
        viewMode?: ViewMode;
    }): Promise<void>;
    onUpdateQueryParams(queryParams: any): void;
    onUpdateGlobals({ globals }: {
        globals: Globals;
    }): Promise<void>;
    onUpdateArgs({ storyId, updatedArgs }: {
        storyId: StoryId;
        updatedArgs: Args;
    }): Promise<void>;
    onPreloadStories({ ids }: {
        ids: string[];
    }): Promise<void>;
    renderSelection({ persistedArgs }?: {
        persistedArgs?: Args;
    }): Promise<void>;
    teardownRender(render: PossibleRender<TFramework>, { viewModeChanged }?: {
        viewModeChanged?: boolean;
    }): Promise<void>;
    extract(options?: {
        includeDocsOnly: boolean;
    }): Promise<Record<string, _storybook_types.StoryContextForEnhancers<TFramework, Args>>>;
    mainStoryCallbacks(storyId: StoryId): {
        showMain: () => void;
        showError: (err: {
            title: string;
            description: string;
        }) => void;
        showException: (err: Error) => void;
    };
    renderPreviewEntryError(reason: string, err: Error): void;
    renderMissingStory(): void;
    renderStoryLoadingException(storySpecifier: StorySpecifier, err: Error): void;
    renderException(storyId: StoryId, error: Error): void;
    renderError(storyId: StoryId, { title, description }: {
        title: string;
        description: string;
    }): void;
}

declare class PreviewWeb<TFramework extends Renderer> extends PreviewWithSelection<TFramework> {
    constructor();
}

declare function simulateDOMContentLoaded(): void;
declare function simulatePageLoad($container: any): void;

export { DocsContext, Preview, PreviewWeb, PreviewWithSelection, simulateDOMContentLoaded, simulatePageLoad };
