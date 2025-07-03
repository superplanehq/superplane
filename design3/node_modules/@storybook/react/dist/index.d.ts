import * as _storybook_types from '@storybook/types';
import { Addon_ClientStoryApi, Addon_Loadable, Args, ComponentAnnotations, AnnotatedStoryFn, ArgsStoryFn, ArgsFromMeta, StoryAnnotations, DecoratorFunction, StrictArgs, LoaderFunction, StoryContext as StoryContext$1, ProjectAnnotations, StoryAnnotationsOrFn, Store_CSFExports, StoriesWithPartialProps } from '@storybook/types';
export { ArgTypes, Args, Parameters, StrictArgs } from '@storybook/types';
import { R as ReactRenderer } from './types-0fc72a6d.js';
import { ComponentType, ComponentProps, JSXElementConstructor } from 'react';
import { Simplify, SetOptional } from 'type-fest';

interface ClientApi extends Addon_ClientStoryApi<ReactRenderer['storyResult']> {
    configure(loader: Addon_Loadable, module: NodeModule): void;
    forceReRender(): void;
    raw: () => any;
}
declare const storiesOf: ClientApi['storiesOf'];
declare const configure: ClientApi['configure'];
declare const forceReRender: ClientApi['forceReRender'];
declare const raw: ClientApi['raw'];

type JSXElement = keyof JSX.IntrinsicElements | JSXElementConstructor<any>;
/**
 * Metadata to configure the stories for a component.
 *
 * @see [Default export](https://storybook.js.org/docs/formats/component-story-format/#default-export)
 */
type Meta<TCmpOrArgs = Args> = [TCmpOrArgs] extends [ComponentType<any>] ? ComponentAnnotations<ReactRenderer, ComponentProps<TCmpOrArgs>> : ComponentAnnotations<ReactRenderer, TCmpOrArgs>;
/**
 * Story function that represents a CSFv2 component example.
 *
 * @see [Named Story exports](https://storybook.js.org/docs/formats/component-story-format/#named-story-exports)
 */
type StoryFn<TCmpOrArgs = Args> = [TCmpOrArgs] extends [ComponentType<any>] ? AnnotatedStoryFn<ReactRenderer, ComponentProps<TCmpOrArgs>> : AnnotatedStoryFn<ReactRenderer, TCmpOrArgs>;
/**
 * Story function that represents a CSFv3 component example.
 *
 * @see [Named Story exports](https://storybook.js.org/docs/formats/component-story-format/#named-story-exports)
 */
type StoryObj<TMetaOrCmpOrArgs = Args> = [TMetaOrCmpOrArgs] extends [
    {
        render?: ArgsStoryFn<ReactRenderer, any>;
        component?: infer Component;
        args?: infer DefaultArgs;
    }
] ? Simplify<(Component extends ComponentType<any> ? ComponentProps<Component> : unknown) & ArgsFromMeta<ReactRenderer, TMetaOrCmpOrArgs>> extends infer TArgs ? StoryAnnotations<ReactRenderer, AddMocks<TArgs, DefaultArgs>, SetOptional<TArgs, keyof TArgs & keyof (DefaultArgs & ActionArgs<TArgs>)>> : never : TMetaOrCmpOrArgs extends ComponentType<any> ? StoryAnnotations<ReactRenderer, ComponentProps<TMetaOrCmpOrArgs>> : StoryAnnotations<ReactRenderer, TMetaOrCmpOrArgs>;
type AddMocks<TArgs, DefaultArgs> = Simplify<{
    [T in keyof TArgs]: T extends keyof DefaultArgs ? DefaultArgs[T] extends (...args: any) => any & {
        mock: {};
    } ? DefaultArgs[T] : TArgs[T] : TArgs[T];
}>;
type ActionArgs<TArgs> = {
    [P in keyof TArgs as TArgs[P] extends (...args: any[]) => any ? ((...args: any[]) => void) extends TArgs[P] ? P : never : never]: TArgs[P];
};
/**
 * @deprecated Use `Meta` instead, e.g. ComponentMeta<typeof Button> -> Meta<typeof Button>.
 *
 * For the common case where a component's stories are simple components that receives args as props:
 *
 * ```tsx
 * export default { ... } as ComponentMeta<typeof Button>;
 * ```
 */
type ComponentMeta<T extends JSXElement> = Meta<ComponentProps<T>>;
/**
 * @deprecated Use `StoryFn` instead, e.g. ComponentStoryFn<typeof Button> -> StoryFn<typeof Button>.
 * Use `StoryObj` if you want to migrate to CSF3, which uses objects instead of functions to represent stories.
 * You can read more about the CSF3 format here: https://storybook.js.org/blog/component-story-format-3-0/
 *
 * For the common case where a (CSFv2) story is a simple component that receives args as props:
 *
 * ```tsx
 * const Template: ComponentStoryFn<typeof Button> = (args) => <Button {...args} />
 * ```
 */
type ComponentStoryFn<T extends JSXElement> = StoryFn<ComponentProps<T>>;
/**
 * @deprecated Use `StoryObj` instead, e.g. ComponentStoryObj<typeof Button> -> StoryObj<typeof Button>.
 *
 * For the common case where a (CSFv3) story is a simple component that receives args as props:
 *
 * ```tsx
 * const MyStory: ComponentStoryObj<typeof Button> = {
 *   args: { buttonArg1: 'val' },
 * }
 * ```
 */
type ComponentStoryObj<T extends JSXElement> = StoryObj<ComponentProps<T>>;
/**
 * @deprecated Use `StoryFn` instead.
 * Use `StoryObj` if you want to migrate to CSF3, which uses objects instead of functions to represent stories.
 * You can read more about the CSF3 format here: https://storybook.js.org/blog/component-story-format-3-0/
 *
 * Story function that represents a CSFv2 component example.
 *
 * @see [Named Story exports](https://storybook.js.org/docs/formats/component-story-format/#named-story-exports)
 */
type Story<TArgs = Args> = StoryFn<TArgs>;
/**
 * @deprecated Use `StoryFn` instead, e.g. ComponentStory<typeof Button> -> StoryFn<typeof Button>.
 * Use `StoryObj` if you want to migrate to CSF3, which uses objects instead of functions to represent stories
 * You can read more about the CSF3 format here: https://storybook.js.org/blog/component-story-format-3-0/.
 *
 * For the common case where a (CSFv3) story is a simple component that receives args as props:
 *
 * ```tsx
 * const MyStory: ComponentStory<typeof Button> = {
 *   args: { buttonArg1: 'val' },
 * }
 * ```
 */
type ComponentStory<T extends JSXElement> = ComponentStoryFn<T>;
/**
 * @deprecated Use Decorator instead.
 */
type DecoratorFn = DecoratorFunction<ReactRenderer>;
type Decorator<TArgs = StrictArgs> = DecoratorFunction<ReactRenderer, TArgs>;
type Loader<TArgs = StrictArgs> = LoaderFunction<ReactRenderer, TArgs>;
type StoryContext<TArgs = StrictArgs> = StoryContext$1<ReactRenderer, TArgs>;
type Preview = ProjectAnnotations<ReactRenderer>;

/** Function that sets the globalConfig of your storybook. The global config is the preview module of your .storybook folder.
 *
 * It should be run a single time, so that your global config (e.g. decorators) is applied to your stories when using `composeStories` or `composeStory`.
 *
 * Example:
 *```jsx
 * // setup.js (for jest)
 * import { setProjectAnnotations } from '@storybook/react';
 * import projectAnnotations from './.storybook/preview';
 *
 * setProjectAnnotations(projectAnnotations);
 *```
 *
 * @param projectAnnotations - e.g. (import projectAnnotations from '../.storybook/preview')
 */
declare function setProjectAnnotations(projectAnnotations: ProjectAnnotations<ReactRenderer> | ProjectAnnotations<ReactRenderer>[]): void;
/** Preserved for users migrating from `@storybook/testing-react`.
 *
 * @deprecated Use setProjectAnnotations instead
 */
declare function setGlobalConfig(projectAnnotations: ProjectAnnotations<ReactRenderer> | ProjectAnnotations<ReactRenderer>[]): void;
/**
 * Function that will receive a story along with meta (e.g. a default export from a .stories file)
 * and optionally projectAnnotations e.g. (import * from '../.storybook/preview)
 * and will return a composed component that has all args/parameters/decorators/etc combined and applied to it.
 *
 *
 * It's very useful for reusing a story in scenarios outside of Storybook like unit testing.
 *
 * Example:
 *```jsx
 * import { render } from '@testing-library/react';
 * import { composeStory } from '@storybook/react';
 * import Meta, { Primary as PrimaryStory } from './Button.stories';
 *
 * const Primary = composeStory(PrimaryStory, Meta);
 *
 * test('renders primary button with Hello World', () => {
 *   const { getByText } = render(<Primary>Hello world</Primary>);
 *   expect(getByText(/Hello world/i)).not.toBeNull();
 * });
 *```
 *
 * @param story
 * @param componentAnnotations - e.g. (import Meta from './Button.stories')
 * @param [projectAnnotations] - e.g. (import * as projectAnnotations from '../.storybook/preview') this can be applied automatically if you use `setProjectAnnotations` in your setup files.
 * @param [exportsName] - in case your story does not contain a name and you want it to have a name.
 */
declare function composeStory<TArgs extends Args = Args>(story: StoryAnnotationsOrFn<ReactRenderer, TArgs>, componentAnnotations: Meta<TArgs | any>, projectAnnotations?: ProjectAnnotations<ReactRenderer>, exportsName?: string): _storybook_types.ComposedStoryFn<ReactRenderer, Partial<TArgs>>;
/**
 * Function that will receive a stories import (e.g. `import * as stories from './Button.stories'`)
 * and optionally projectAnnotations (e.g. `import * from '../.storybook/preview`)
 * and will return an object containing all the stories passed, but now as a composed component that has all args/parameters/decorators/etc combined and applied to it.
 *
 *
 * It's very useful for reusing stories in scenarios outside of Storybook like unit testing.
 *
 * Example:
 *```jsx
 * import { render } from '@testing-library/react';
 * import { composeStories } from '@storybook/react';
 * import * as stories from './Button.stories';
 *
 * const { Primary, Secondary } = composeStories(stories);
 *
 * test('renders primary button with Hello World', () => {
 *   const { getByText } = render(<Primary>Hello world</Primary>);
 *   expect(getByText(/Hello world/i)).not.toBeNull();
 * });
 *```
 *
 * @param csfExports - e.g. (import * as stories from './Button.stories')
 * @param [projectAnnotations] - e.g. (import * as projectAnnotations from '../.storybook/preview') this can be applied automatically if you use `setProjectAnnotations` in your setup files.
 */
declare function composeStories<TModule extends Store_CSFExports<ReactRenderer, any>>(csfExports: TModule, projectAnnotations?: ProjectAnnotations<ReactRenderer>): Omit<StoriesWithPartialProps<ReactRenderer, TModule>, keyof Store_CSFExports>;

export { ComponentMeta, ComponentStory, ComponentStoryFn, ComponentStoryObj, Decorator, DecoratorFn, Loader, Meta, Preview, ReactRenderer, Story, StoryContext, StoryFn, StoryObj, composeStories, composeStory, configure, forceReRender, raw, setGlobalConfig, setProjectAnnotations, storiesOf };
