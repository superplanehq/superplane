import * as _storybook_types from '@storybook/types';
import { Renderer, NormalizedProjectAnnotations, StoryId, IndexEntry, ComponentId, Path, ModuleExports, DecoratorFunction, Parameters, LoaderFunction, Args, ArgTypes, ArgsEnhancer, ArgTypesEnhancer, StepRunner, ModuleImportFn, Globals, GlobalTypes, Addon_StoryApi } from '@storybook/types';
import { S as StoryStore } from './StoryStore-f7424ddf.js';

declare class StoryStoreFacade<TRenderer extends Renderer> {
    projectAnnotations: NormalizedProjectAnnotations<TRenderer>;
    entries: Record<StoryId, IndexEntry & {
        componentId?: ComponentId;
    }>;
    csfExports: Record<Path, ModuleExports>;
    constructor();
    importFn(path: Path): Promise<ModuleExports>;
    getStoryIndex(store: StoryStore<TRenderer>): {
        v: number;
        entries: Record<string, IndexEntry>;
    };
    clearFilenameExports(fileName: Path): void;
    addStoriesFromExports(fileName: Path, fileExports: ModuleExports): void;
}

declare const addDecorator: (decorator: DecoratorFunction<Renderer>) => void;
declare const addParameters: (parameters: Parameters) => void;
declare const addLoader: (loader: LoaderFunction<Renderer>) => void;
declare const addArgs: (args: Args) => void;
declare const addArgTypes: (argTypes: ArgTypes) => void;
declare const addArgsEnhancer: (enhancer: ArgsEnhancer<Renderer>) => void;
declare const addArgTypesEnhancer: (enhancer: ArgTypesEnhancer<Renderer>) => void;
declare const addStepRunner: (stepRunner: StepRunner) => void;
declare const setGlobalRender: (render: StoryStoreFacade<any>['projectAnnotations']['render']) => void;
declare class ClientApi<TRenderer extends Renderer> {
    facade: StoryStoreFacade<TRenderer>;
    storyStore?: StoryStore<TRenderer>;
    private addons;
    onImportFnChanged?: ({ importFn }: {
        importFn: ModuleImportFn;
    }) => void;
    private lastFileName;
    constructor({ storyStore }?: {
        storyStore?: StoryStore<TRenderer>;
    });
    importFn(path: Path): Promise<ModuleExports>;
    getStoryIndex(): {
        v: number;
        entries: Record<string, _storybook_types.IndexEntry>;
    };
    addDecorator: (decorator: DecoratorFunction<TRenderer>) => void;
    addParameters: ({ globals, globalTypes, ...parameters }: Parameters & {
        globals?: Globals | undefined;
        globalTypes?: GlobalTypes | undefined;
    }) => void;
    addStepRunner: (stepRunner: StepRunner<TRenderer>) => void;
    addLoader: (loader: LoaderFunction<TRenderer>) => void;
    addArgs: (args: Args) => void;
    addArgTypes: (argTypes: ArgTypes) => void;
    addArgsEnhancer: (enhancer: ArgsEnhancer<TRenderer>) => void;
    addArgTypesEnhancer: (enhancer: ArgTypesEnhancer<TRenderer>) => void;
    _addedExports: Record<string, ModuleExports>;
    _loadAddedExports(): void;
    storiesOf: (kind: string, m?: NodeModule) => Addon_StoryApi<TRenderer['storyResult']>;
    raw: () => _storybook_types.BoundStory<TRenderer>[] | undefined;
    get _storyStore(): StoryStore<TRenderer> | undefined;
}

export { ClientApi as C, addArgTypes as a, addArgTypesEnhancer as b, addArgs as c, addArgsEnhancer as d, addDecorator as e, addLoader as f, addParameters as g, addStepRunner as h, setGlobalRender as s };
