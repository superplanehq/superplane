import * as _storybook_types from '@storybook/types';
import { Renderer, StoryId, LegacyStoryAnnotationsOrFn, NormalizedComponentAnnotations, NormalizedStoryAnnotations, StepRunner, ProjectAnnotations, Args, ComponentAnnotations, ComposedStoryFn, Store_CSFExports, ComposeStoryFn, Parameters, StrictArgTypes, ArgTypesEnhancer, LegacyStoryFn, DecoratorFunction, PartialStoryFn, StoryContextUpdate, ArgTypes, StoryContext, NormalizedStoriesSpecifier, IndexEntry, Addon_StorySortParameterV7, Path, PreparedStory, Addon_StorySortParameter } from '@storybook/types';

declare function normalizeStory<TRenderer extends Renderer>(key: StoryId, storyAnnotations: LegacyStoryAnnotationsOrFn<TRenderer>, meta: NormalizedComponentAnnotations<TRenderer>): NormalizedStoryAnnotations<TRenderer>;

/**
 * Compose step runners to create a single step runner that applies each step runner in order.
 *
 * A step runner is a a function that takes a defined step: `step('label', () => { ... })`
 * and runs it. The prototypical example is from `@storybook/addon-interactions` where the
 * step runner will decorate all instrumented code inside the step with information about the
 * label.
 *
 * In theory it is possible to have more than one addon that wants to run steps; they can be
 * composed together in a similar fashion to decorators. In some ways step runners are like
 * decorators except it is not intended that they change the context or the play function.
 *
 * The basic implementation of a step runner is `async (label, play, context) => play(context)`
 *  -- in fact this is what `composeStepRunners([])` will do.
 *
 * @param stepRunners an array of StepRunner
 * @returns a StepRunner that is the composition of the arguments
 */
declare function composeStepRunners<TRenderer extends Renderer>(stepRunners: StepRunner<TRenderer>[]): StepRunner<TRenderer>;

declare function setProjectAnnotations<TRenderer extends Renderer = Renderer>(projectAnnotations: ProjectAnnotations<TRenderer> | ProjectAnnotations<TRenderer>[]): void;
declare function composeStory<TRenderer extends Renderer = Renderer, TArgs extends Args = Args>(storyAnnotations: LegacyStoryAnnotationsOrFn<TRenderer>, componentAnnotations: ComponentAnnotations<TRenderer, TArgs>, projectAnnotations?: ProjectAnnotations<TRenderer>, defaultConfig?: ProjectAnnotations<TRenderer>, exportsName?: string): ComposedStoryFn<TRenderer, Partial<TArgs>>;
declare function composeStories<TModule extends Store_CSFExports>(storiesImport: TModule, globalConfig: ProjectAnnotations<Renderer>, composeStoryFn: ComposeStoryFn): {};

/**
 * Safely combine parameters recursively. Only copy objects when needed.
 * Algorithm = always overwrite the existing value UNLESS both values
 * are plain objects. In this case flag the key as "special" and handle
 * it with a heuristic.
 */
declare const combineParameters: (...parameterSets: (Parameters | undefined)[]) => Parameters;

type PropDescriptor = string[] | RegExp;
declare const filterArgTypes: (argTypes: StrictArgTypes, include?: PropDescriptor, exclude?: PropDescriptor) => StrictArgTypes<_storybook_types.Args>;

declare const inferControls: ArgTypesEnhancer<Renderer>;

declare function decorateStory<TRenderer extends Renderer>(storyFn: LegacyStoryFn<TRenderer>, decorator: DecoratorFunction<TRenderer>, bindWithContext: (storyFn: LegacyStoryFn<TRenderer>) => PartialStoryFn<TRenderer>): LegacyStoryFn<TRenderer>;
/**
 * Currently StoryContextUpdates are allowed to have any key in the type.
 * However, you cannot overwrite any of the build-it "static" keys.
 *
 * @param inputContextUpdate StoryContextUpdate
 * @returns StoryContextUpdate
 */
declare function sanitizeStoryContextUpdate({ componentId, title, kind, id, name, story, parameters, initialArgs, argTypes, ...update }?: StoryContextUpdate): StoryContextUpdate;
declare function defaultDecorateStory<TRenderer extends Renderer>(storyFn: LegacyStoryFn<TRenderer>, decorators: DecoratorFunction<TRenderer>[]): LegacyStoryFn<TRenderer>;

declare const mapArgsToTypes: (args: Args, argTypes: ArgTypes) => Args;
declare const combineArgs: (value: any, update: any) => Args;
declare const validateOptions: (args: Args, argTypes: ArgTypes) => Args;
declare const DEEPLY_EQUAL: unique symbol;
declare const deepDiff: (value: any, update: any) => any;
declare const UNTARGETED = "UNTARGETED";
declare function groupArgsByTarget<TArgs extends Args = Args>({ args, argTypes, }: Pick<StoryContext<Renderer, TArgs>, 'args' | 'argTypes'>): Record<string, Partial<TArgs>>;
declare function noTargetArgs<TArgs extends Args = Args>(context: Pick<StoryContext<Renderer, TArgs>, 'args' | 'argTypes'>): Partial<TArgs>;

declare const userOrAutoTitleFromSpecifier: (fileName: string | number, entry: NormalizedStoriesSpecifier, userTitle?: string) => string | undefined;
declare const userOrAutoTitle: (fileName: string, storiesEntries: NormalizedStoriesSpecifier[], userTitle?: string) => string | undefined;

declare const sortStoriesV7: (stories: IndexEntry[], storySortParameter: Addon_StorySortParameterV7, fileNameOrder: Path[]) => IndexEntry[];
declare const sortStoriesV6: <TRenderer extends Renderer>(stories: [string, PreparedStory<TRenderer>, Parameters, Parameters][], storySortParameter: Addon_StorySortParameter, fileNameOrder: Path[]) => IndexEntry[];

export { DEEPLY_EQUAL as D, PropDescriptor as P, UNTARGETED as U, combineParameters as a, composeStepRunners as b, combineArgs as c, composeStories as d, composeStory as e, decorateStory as f, defaultDecorateStory as g, filterArgTypes as h, setProjectAnnotations as i, inferControls as j, userOrAutoTitle as k, sortStoriesV7 as l, mapArgsToTypes as m, normalizeStory as n, deepDiff as o, groupArgsByTarget as p, noTargetArgs as q, sortStoriesV6 as r, sanitizeStoryContextUpdate as s, userOrAutoTitleFromSpecifier as u, validateOptions as v };
