import { Simplify, RemoveIndexSignature, UnionToIntersection } from 'type-fest';

interface SBBaseType {
    required?: boolean;
    raw?: string;
}
type SBScalarType = SBBaseType & {
    name: 'boolean' | 'string' | 'number' | 'function' | 'symbol';
};
type SBArrayType = SBBaseType & {
    name: 'array';
    value: SBType;
};
type SBObjectType = SBBaseType & {
    name: 'object';
    value: Record<string, SBType>;
};
type SBEnumType = SBBaseType & {
    name: 'enum';
    value: (string | number)[];
};
type SBIntersectionType = SBBaseType & {
    name: 'intersection';
    value: SBType[];
};
type SBUnionType = SBBaseType & {
    name: 'union';
    value: SBType[];
};
type SBOtherType = SBBaseType & {
    name: 'other';
    value: string;
};
type SBType = SBScalarType | SBEnumType | SBArrayType | SBObjectType | SBIntersectionType | SBUnionType | SBOtherType;

type StoryId = string;
type ComponentId = string;
type ComponentTitle = string;
type StoryName = string;
/** @deprecated */
type StoryKind = ComponentTitle;
type Tag = string;
interface StoryIdentifier {
    componentId: ComponentId;
    title: ComponentTitle;
    /** @deprecated */
    kind: ComponentTitle;
    id: StoryId;
    name: StoryName;
    /** @deprecated */
    story: StoryName;
    tags: Tag[];
}
interface Parameters {
    [name: string]: any;
}
interface StrictParameters {
    [name: string]: unknown;
}
type ControlType = 'object' | 'boolean' | 'check' | 'inline-check' | 'radio' | 'inline-radio' | 'select' | 'multi-select' | 'number' | 'range' | 'file' | 'color' | 'date' | 'text';
type ConditionalTest = {
    truthy?: boolean;
} | {
    exists: boolean;
} | {
    eq: any;
} | {
    neq: any;
};
type ConditionalValue = {
    arg: string;
} | {
    global: string;
};
type Conditional = ConditionalValue & ConditionalTest;
interface ControlBase {
    [key: string]: any;
    /**
     * @see https://storybook.js.org/docs/api/arg-types#controltype
     */
    type?: ControlType;
    disable?: boolean;
}
interface Report {
    type: string;
    version?: number;
    result: unknown;
    status: 'failed' | 'passed' | 'warning';
}
interface ReportingAPI {
    reports: Report[];
    addReport: (report: Report) => void;
}
type Control = ControlType | false | (ControlBase & (ControlBase | {
    type: 'color';
    /**
     * @see https://storybook.js.org/docs/api/arg-types#controlpresetcolors
     */
    presetColors?: string[];
} | {
    type: 'file';
    /**
     * @see https://storybook.js.org/docs/api/arg-types#controlaccept
     */
    accept?: string;
} | {
    type: 'inline-check' | 'radio' | 'inline-radio' | 'select' | 'multi-select';
    /**
     * @see https://storybook.js.org/docs/api/arg-types#controllabels
     */
    labels?: {
        [options: string]: string;
    };
} | {
    type: 'number' | 'range';
    /**
     * @see https://storybook.js.org/docs/api/arg-types#controlmax
     */
    max?: number;
    /**
     * @see https://storybook.js.org/docs/api/arg-types#controlmin
     */
    min?: number;
    /**
     * @see https://storybook.js.org/docs/api/arg-types#controlstep
     */
    step?: number;
}));
interface InputType {
    /**
     * @see https://storybook.js.org/docs/api/arg-types#control
     */
    control?: Control;
    /**
     * @see https://storybook.js.org/docs/api/arg-types#description
     */
    description?: string;
    /**
     * @see https://storybook.js.org/docs/api/arg-types#if
     */
    if?: Conditional;
    /**
     * @see https://storybook.js.org/docs/api/arg-types#mapping
     */
    mapping?: {
        [key: string]: any;
    };
    /**
     * @see https://storybook.js.org/docs/api/arg-types#name
     */
    name?: string;
    /**
     * @see https://storybook.js.org/docs/api/arg-types#options
     */
    options?: readonly any[];
    /**
     * @see https://storybook.js.org/docs/api/arg-types#table
     */
    table?: {
        [key: string]: unknown;
        /**
         * @see https://storybook.js.org/docs/api/arg-types#tablecategory
         */
        category?: string;
        /**
         * @see https://storybook.js.org/docs/api/arg-types#tabledefaultvalue
         */
        defaultValue?: {
            summary?: string;
            detail?: string;
        };
        /**
         * @see https://storybook.js.org/docs/api/arg-types#tabledisable
         */
        disable?: boolean;
        /**
         * @see https://storybook.js.org/docs/api/arg-types#tablesubcategory
         */
        subcategory?: string;
        /**
         * @see https://storybook.js.org/docs/api/arg-types#tabletype
         */
        type?: {
            summary?: string;
            detail?: string;
        };
    };
    /**
     * @see https://storybook.js.org/docs/api/arg-types#type
     */
    type?: SBType | SBScalarType['name'];
    /**
     * @see https://storybook.js.org/docs/api/arg-types#defaultvalue
     *
     * @deprecated Use `table.defaultValue.summary` instead.
     */
    defaultValue?: any;
    [key: string]: any;
}
interface StrictInputType extends InputType {
    name: string;
    type?: SBType;
}
interface Args {
    [name: string]: any;
}
interface StrictArgs {
    [name: string]: unknown;
}
/**
 * @see https://storybook.js.org/docs/api/arg-types#argtypes
 */
type ArgTypes<TArgs = Args> = {
    [name in keyof TArgs]: InputType;
};
type StrictArgTypes<TArgs = Args> = {
    [name in keyof TArgs]: StrictInputType;
};
interface Globals {
    [name: string]: any;
}
interface GlobalTypes {
    [name: string]: InputType;
}
interface StrictGlobalTypes {
    [name: string]: StrictInputType;
}
interface Renderer {
    /** What is the type of the `component` annotation in this renderer? */
    component: unknown;
    /** What does the story function return in this renderer? */
    storyResult: unknown;
    /** What type of element does this renderer render to? */
    canvasElement: unknown;
    mount(): Promise<Canvas>;
    T?: unknown;
}
/** @deprecated - use `Renderer` */
type AnyFramework = Renderer;
interface StoryContextForEnhancers<TRenderer extends Renderer = Renderer, TArgs = Args> extends StoryIdentifier {
    component?: (TRenderer & {
        T: any;
    })['component'];
    subcomponents?: Record<string, (TRenderer & {
        T: any;
    })['component']>;
    parameters: Parameters;
    initialArgs: TArgs;
    argTypes: StrictArgTypes<TArgs>;
}
type ArgsEnhancer<TRenderer extends Renderer = Renderer, TArgs = Args> = (context: StoryContextForEnhancers<TRenderer, TArgs>) => TArgs;
type ArgTypesEnhancer<TRenderer extends Renderer = Renderer, TArgs = Args> = ((context: StoryContextForEnhancers<TRenderer, TArgs>) => StrictArgTypes<TArgs>) & {
    secondPass?: boolean;
};
interface StoryContextUpdate<TArgs = Args> {
    args?: TArgs;
    globals?: Globals;
    [key: string]: any;
}
type ViewMode = 'story' | 'docs';
type LoaderFunction<TRenderer extends Renderer = Renderer, TArgs = Args> = (context: StoryContextForLoaders<TRenderer, TArgs>) => Promise<Record<string, any> | void> | Record<string, any> | void;
type Awaitable<T> = T | PromiseLike<T>;
type CleanupCallback = () => Awaitable<unknown>;
type BeforeAll = () => Awaitable<CleanupCallback | void>;
type BeforeEach<TRenderer extends Renderer = Renderer, TArgs = Args> = (context: StoryContext<TRenderer, TArgs>) => Awaitable<CleanupCallback | void>;
type AfterEach<TRenderer extends Renderer = Renderer, TArgs = Args> = (context: StoryContext<TRenderer, TArgs>) => Awaitable<void>;
interface Canvas {
}
interface StoryContext<TRenderer extends Renderer = Renderer, TArgs = Args> extends StoryContextForEnhancers<TRenderer, TArgs>, Required<StoryContextUpdate<TArgs>> {
    loaded: Record<string, any>;
    abortSignal: AbortSignal;
    canvasElement: TRenderer['canvasElement'];
    hooks: unknown;
    originalStoryFn: StoryFn<TRenderer>;
    viewMode: ViewMode;
    step: StepFunction<TRenderer, TArgs>;
    context: this;
    canvas: Canvas;
    mount: TRenderer['mount'];
    reporting: ReportingAPI;
}
/** @deprecated Use {@link StoryContext} instead. */
interface StoryContextForLoaders<TRenderer extends Renderer = Renderer, TArgs = Args> extends StoryContext<TRenderer, TArgs> {
}
/** @deprecated Use {@link StoryContext} instead. */
interface PlayFunctionContext<TRenderer extends Renderer = Renderer, TArgs = Args> extends StoryContext<TRenderer, TArgs> {
}
type StepLabel = string;
type StepFunction<TRenderer extends Renderer = Renderer, TArgs = Args> = (label: StepLabel, play: PlayFunction<TRenderer, TArgs>) => Promise<void> | void;
type PlayFunction<TRenderer extends Renderer = Renderer, TArgs = Args> = (context: PlayFunctionContext<TRenderer, TArgs>) => Promise<void> | void;
type PartialStoryFn<TRenderer extends Renderer = Renderer, TArgs = Args> = (update?: StoryContextUpdate<Partial<TArgs>>) => TRenderer['storyResult'];
type LegacyStoryFn<TRenderer extends Renderer = Renderer, TArgs = Args> = (context: StoryContext<TRenderer, TArgs>) => TRenderer['storyResult'];
type ArgsStoryFn<TRenderer extends Renderer = Renderer, TArgs = Args> = (args: TArgs, context: StoryContext<TRenderer, TArgs>) => (TRenderer & {
    T: TArgs;
})['storyResult'];
type StoryFn<TRenderer extends Renderer = Renderer, TArgs = Args> = LegacyStoryFn<TRenderer, TArgs> | ArgsStoryFn<TRenderer, TArgs>;
type DecoratorFunction<TRenderer extends Renderer = Renderer, TArgs = Args> = (fn: PartialStoryFn<TRenderer, TArgs>, c: StoryContext<TRenderer, TArgs>) => TRenderer['storyResult'];
type DecoratorApplicator<TRenderer extends Renderer = Renderer, TArgs = Args> = (storyFn: LegacyStoryFn<TRenderer, TArgs>, decorators: DecoratorFunction<TRenderer, TArgs>[]) => LegacyStoryFn<TRenderer, TArgs>;
type StepRunner<TRenderer extends Renderer = Renderer, TArgs = Args> = (label: StepLabel, play: PlayFunction<TRenderer, TArgs>, context: StoryContext<TRenderer, TArgs>) => Promise<void>;
interface BaseAnnotations<TRenderer extends Renderer = Renderer, TArgs = Args> {
    /**
     * Wrapper components or Storybook decorators that wrap a story.
     *
     * Decorators defined in Meta will be applied to every story variation.
     * @see [Decorators](https://storybook.js.org/docs/writing-stories/decorators)
     */
    decorators?: DecoratorFunction<TRenderer, Simplify<TArgs>>[] | DecoratorFunction<TRenderer, Simplify<TArgs>>;
    /**
     * Custom metadata for a story.
     * @see [Parameters](https://storybook.js.org/docs/writing-stories/parameters)
     */
    parameters?: Parameters;
    /**
     * Dynamic data that are provided (and possibly updated by) Storybook and its addons.
     * @see [Args](https://storybook.js.org/docs/writing-stories/args)
     */
    args?: Partial<TArgs>;
    /**
     * ArgTypes encode basic metadata for args, such as `name`, `description`, `defaultValue` for an arg. These get automatically filled in by Storybook Docs.
     * @see [ArgTypes](https://storybook.js.org/docs/api/arg-types)
     */
    argTypes?: Partial<ArgTypes<TArgs>>;
    /**
     * Asynchronous functions which provide data for a story.
     * @see [Loaders](https://storybook.js.org/docs/writing-stories/loaders)
     */
    loaders?: LoaderFunction<TRenderer, TArgs>[] | LoaderFunction<TRenderer, TArgs>;
    /**
     * Function to be called before each story. When the function is async, it will be awaited.
     *
     * `beforeEach` can be added to preview, the default export and to a specific story.
     * They are run (and awaited) in the order: preview, default export, story
     *
     * A cleanup function can be returned.
     */
    beforeEach?: BeforeEach<TRenderer, TArgs>[] | BeforeEach<TRenderer, TArgs>;
    /**
     * Function to be called after each play function for post-test assertions.
     * Don't use this function for cleaning up state.
     * You can use the return callback of `beforeEach` for that, which is run when switching stories.
     * When the function is async, it will be awaited.
     *
     * `afterEach` can be added to preview, the default export and to a specific story.
     * They are run (and awaited) reverse order: preview, default export, story
     */
    experimental_afterEach?: AfterEach<TRenderer, TArgs>[] | AfterEach<TRenderer, TArgs>;
    /**
     * Define a custom render function for the story(ies). If not passed, a default render function by the renderer will be used.
     */
    render?: ArgsStoryFn<TRenderer, TArgs>;
    /**
     * Named tags for a story, used to filter stories in different contexts.
     */
    tags?: Tag[];
    mount?: (context: StoryContext<TRenderer, TArgs>) => TRenderer['mount'];
}
interface ProjectAnnotations<TRenderer extends Renderer = Renderer, TArgs = Args> extends BaseAnnotations<TRenderer, TArgs> {
    argsEnhancers?: ArgsEnhancer<TRenderer, Args>[];
    argTypesEnhancers?: ArgTypesEnhancer<TRenderer, Args>[];
    /**
     * Lifecycle hook which runs once, before any loaders, decorators or stories, and may rerun when configuration changes or when reinitializing (e.g. between test runs).
     * The function may be synchronous or asynchronous, and may return a cleanup function which may also be synchronous or asynchronous.
     * The cleanup function is not guaranteed to run (e.g. when the browser closes), but runs when configuration changes or when reinitializing.
     * This hook may only be defined globally (i.e. not on component or story level).
     * When multiple hooks are specified, they are to be executed sequentially (and awaited) in the following order:
     * - Addon hooks (in order of addons array in e.g. .storybook/main.js)
     * - Annotation hooks (in order of previewAnnotations array in e.g. .storybook/main.js)
     * - Preview hook (via e.g. .storybook/preview.js)
     * Cleanup functions are executed sequentially in reverse order of initialization.
     */
    beforeAll?: BeforeAll;
    /**
     * @deprecated Project `globals` renamed to `initiaGlobals`
     */
    globals?: Globals;
    initialGlobals?: Globals;
    globalTypes?: GlobalTypes;
    applyDecorators?: DecoratorApplicator<TRenderer, Args>;
    runStep?: StepRunner<TRenderer, TArgs>;
}
type StoryDescriptor$1 = string[] | RegExp;
interface ComponentAnnotations<TRenderer extends Renderer = Renderer, TArgs = Args> extends BaseAnnotations<TRenderer, TArgs> {
    /**
     * Title of the component which will be presented in the navigation. **Should be unique.**
     *
     * Components can be organized in a nested structure using "/" as a separator.
     *
     * Since CSF 3.0 this property is optional -- it can be inferred from the filesystem path
     *
     * @example
     * export default {
     *   ...
     *   title: 'Design System/Atoms/Button'
     * }
     *
     * @see [Story Hierarchy](https://storybook.js.org/docs/writing-stories/naming-components-and-hierarchy#structure-and-hierarchy)
     */
    title?: ComponentTitle;
    /**
     * Id of the component (prefix of the story id) which is used for URLs.
     *
     * By default is inferred from sanitizing the title
     *
     * @see [Permalink to stories](https://storybook.js.org/docs/configure/sidebar-and-urls#permalink-to-stories)
     */
    id?: ComponentId;
    /**
     * Used to only include certain named exports as stories. Useful when you want to have non-story exports such as mock data or ignore a few stories.
     * @example
     * includeStories: ['SimpleStory', 'ComplexStory']
     * includeStories: /.*Story$/
     *
     * @see [Non-story exports](https://storybook.js.org/docs/api/csf#non-story-exports)
     */
    includeStories?: StoryDescriptor$1;
    /**
     * Used to exclude certain named exports. Useful when you want to have non-story exports such as mock data or ignore a few stories.
     * @example
     * excludeStories: ['simpleData', 'complexData']
     * excludeStories: /.*Data$/
     *
     * @see [Non-story exports](https://storybook.js.org/docs/api/csf#non-story-exports)
     */
    excludeStories?: StoryDescriptor$1;
    /**
     * The primary component for your story.
     *
     * Used by addons for automatic prop table generation and display of other component metadata.
     */
    component?: (TRenderer & {
        T: Record<string, unknown> extends Required<TArgs> ? any : TArgs;
    })['component'];
    /**
     * Auxiliary subcomponents that are part of the stories.
     *
     * Used by addons for automatic prop table generation and display of other component metadata.
     *
     * @example
     * import { Button, ButtonGroup } from './components';
     *
     * export default {
     *   ...
     *   subcomponents: { Button, ButtonGroup }
     * }
     *
     * By defining them each component will have its tab in the args table.
     */
    subcomponents?: Record<string, (TRenderer & {
        T: any;
    })['component']>;
    /**
     * Function that is executed after the story is rendered.
     */
    play?: PlayFunction<TRenderer, TArgs>;
    /**
     * Override the globals values for all stories in this component
     */
    globals?: Globals;
}
type StoryAnnotations<TRenderer extends Renderer = Renderer, TArgs = Args, TRequiredArgs = Partial<TArgs>> = BaseAnnotations<TRenderer, TArgs> & {
    /**
     * Override the display name in the UI (CSF v3)
     */
    name?: StoryName;
    /**
     * Override the display name in the UI (CSF v2)
     */
    storyName?: StoryName;
    /**
     * Function that is executed after the story is rendered.
     */
    play?: PlayFunction<TRenderer, TArgs>;
    /**
     * Override the globals values for this story
     */
    globals?: Globals;
    /** @deprecated */
    story?: Omit<StoryAnnotations<TRenderer, TArgs>, 'story'>;
} & ({} extends TRequiredArgs ? {
    args?: TRequiredArgs;
} : {
    args: TRequiredArgs;
});
type LegacyAnnotatedStoryFn<TRenderer extends Renderer = Renderer, TArgs = Args> = StoryFn<TRenderer, TArgs> & StoryAnnotations<TRenderer, TArgs>;
type LegacyStoryAnnotationsOrFn<TRenderer extends Renderer = Renderer, TArgs = Args> = LegacyAnnotatedStoryFn<TRenderer, TArgs> | StoryAnnotations<TRenderer, TArgs>;
type AnnotatedStoryFn<TRenderer extends Renderer = Renderer, TArgs = Args> = ArgsStoryFn<TRenderer, TArgs> & StoryAnnotations<TRenderer, TArgs>;
type StoryAnnotationsOrFn<TRenderer extends Renderer = Renderer, TArgs = Args> = AnnotatedStoryFn<TRenderer, TArgs> | StoryAnnotations<TRenderer, TArgs>;
type ArgsFromMeta<TRenderer extends Renderer, Meta> = Meta extends {
    render?: ArgsStoryFn<TRenderer, infer RArgs>;
    loaders?: (infer Loaders)[] | infer Loaders;
    decorators?: (infer Decorators)[] | infer Decorators;
} ? Simplify<RemoveIndexSignature<RArgs & DecoratorsArgs<TRenderer, Decorators> & LoaderArgs<TRenderer, Loaders>>> : unknown;
type DecoratorsArgs<TRenderer extends Renderer, Decorators> = UnionToIntersection<Decorators extends DecoratorFunction<TRenderer, infer TArgs> ? TArgs : unknown>;
type LoaderArgs<TRenderer extends Renderer, Loaders> = UnionToIntersection<Loaders extends LoaderFunction<TRenderer, infer TArgs> ? TArgs : unknown>;

/**
 * Helper function to include/exclude an arg based on the value of other other args
 * aka "conditional args"
 */
declare const includeConditionalArg: (argType: InputType, args: Args, globals: Globals) => any;

/**
 * Remove punctuation and illegal characters from a story ID.
 *
 * See https://gist.github.com/davidjrice/9d2af51100e41c6c4b4a
 */
declare const sanitize: (string: string) => string;
/**
 * Generate a storybook ID from a component/kind and story name.
 */
declare const toId: (kind: string, name?: string) => string;
/**
 * Transform a CSF named export into a readable story name
 */
declare const storyNameFromExport: (key: string) => string;
type StoryDescriptor = string[] | RegExp;
interface IncludeExcludeOptions {
    includeStories?: StoryDescriptor;
    excludeStories?: StoryDescriptor;
}
/**
 * Does a named export match CSF inclusion/exclusion options?
 */
declare function isExportStory(key: string, { includeStories, excludeStories }: IncludeExcludeOptions): boolean | null;
interface SeparatorOptions {
    rootSeparator: string | RegExp;
    groupSeparator: string | RegExp;
}
/**
 * Parse out the component/kind name from a path, using the given separator config.
 */
declare const parseKind: (kind: string, { rootSeparator, groupSeparator }: SeparatorOptions) => {
    root: string | null;
    groups: string[];
};
/**
 * Combine a set of project / meta / story tags, removing duplicates and handling negations.
 */
declare const combineTags: (...tags: string[]) => string[];

export { AfterEach, AnnotatedStoryFn, AnyFramework, ArgTypes, ArgTypesEnhancer, Args, ArgsEnhancer, ArgsFromMeta, ArgsStoryFn, BaseAnnotations, BeforeAll, BeforeEach, Canvas, CleanupCallback, ComponentAnnotations, ComponentId, ComponentTitle, Conditional, DecoratorApplicator, DecoratorFunction, GlobalTypes, Globals, IncludeExcludeOptions, InputType, LegacyAnnotatedStoryFn, LegacyStoryAnnotationsOrFn, LegacyStoryFn, LoaderFunction, Parameters, PartialStoryFn, PlayFunction, PlayFunctionContext, ProjectAnnotations, Renderer, SBArrayType, SBEnumType, SBIntersectionType, SBObjectType, SBOtherType, SBScalarType, SBType, SBUnionType, SeparatorOptions, StepFunction, StepLabel, StepRunner, StoryAnnotations, StoryAnnotationsOrFn, StoryContext, StoryContextForEnhancers, StoryContextForLoaders, StoryContextUpdate, StoryFn, StoryId, StoryIdentifier, StoryKind, StoryName, StrictArgTypes, StrictArgs, StrictGlobalTypes, StrictInputType, StrictParameters, Tag, ViewMode, combineTags, includeConditionalArg, isExportStory, parseKind, sanitize, storyNameFromExport, toId };
