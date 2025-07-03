import { Conditional, DocsContextProps, PreparedStory, ModuleExports, ModuleExport, Parameters as Parameters$1, StrictArgTypes, Renderer as Renderer$1, Args as Args$1, StoryId, StoryContextForLoaders, StoryAnnotations, ResolvedModuleExportType, ResolvedModuleExportFromType, ProjectAnnotations, BaseAnnotations, ComponentTitle } from '@storybook/types';
export { DocsContextProps } from '@storybook/types';
import React, { ComponentProps, FunctionComponent, ReactNode, FC, PropsWithChildren, Context } from 'react';
import { Renderer } from '@storybook/csf';
import { PropDescriptor } from '@storybook/preview-api';
import { SyntaxHighlighter, ActionItem, SyntaxHighlighterFormatTypes } from '@storybook/components';
import { SourceType } from '@storybook/docs-tools';
import { Channel } from '@storybook/channels';
import { ThemeVars } from '@storybook/theming';
import PureMarkdown from 'markdown-to-jsx';

declare enum SourceError {
    NO_STORY = "There\u2019s no story here.",
    SOURCE_UNAVAILABLE = "Oh no! The source is not available."
}
interface SourceCodeProps {
    /**
     * The language the syntax highlighter uses for your story’s code
     */
    language?: string;
    /**
     * Use this to override the content of the source block.
     */
    code?: string;
    /**
     * The (prettier) formatter the syntax highlighter uses for your story’s code.
     */
    format?: ComponentProps<typeof SyntaxHighlighter>['format'];
    /**
     * Display the source snippet in a dark mode.
     */
    dark?: boolean;
}
interface SourceProps$1 extends SourceCodeProps {
    isLoading?: boolean;
    error?: SourceError;
}
/**
 * Syntax-highlighted source code for a component (or anything!)
 */
declare const Source$1: FunctionComponent<SourceProps$1>;

interface PreviewProps {
    isLoading?: true;
    layout?: Layout;
    isColumn?: boolean;
    columns?: number;
    withSource?: SourceProps$1;
    isExpanded?: boolean;
    withToolbar?: boolean;
    className?: string;
    additionalActions?: ActionItem[];
    children?: ReactNode;
}
type Layout = 'padded' | 'fullscreen' | 'centered';

interface ArgType {
    name?: string;
    description?: string;
    defaultValue?: any;
    if?: Conditional;
    [key: string]: any;
}
interface ArgTypes$1 {
    [key: string]: ArgType;
}
interface Args {
    [key: string]: any;
}
type Globals = {
    [name: string]: any;
};

declare enum ArgsTableError {
    NO_COMPONENT = "No component found.",
    ARGS_UNSUPPORTED = "Args unsupported. See Args documentation for your framework."
}
type SortType = 'alpha' | 'requiredFirst' | 'none';
interface ArgsTableOptionProps {
    children?: React.ReactNode;
    updateArgs?: (args: Args) => void;
    resetArgs?: (argNames?: string[]) => void;
    compact?: boolean;
    inAddonPanel?: boolean;
    initialExpandedArgs?: boolean;
    isLoading?: boolean;
    sort?: SortType;
}
interface ArgsTableDataProps {
    rows: ArgTypes$1;
    args?: Args;
    globals?: Globals;
}
interface ArgsTableErrorProps {
    error: ArgsTableError;
}
interface ArgsTableLoadingProps {
    isLoading: boolean;
}
type ArgsTableProps$1 = ArgsTableOptionProps & (ArgsTableDataProps | ArgsTableErrorProps | ArgsTableLoadingProps);
/**
 * Display the props for a component as a props table. Each row is a collection of
 * ArgDefs, usually derived from docgen info for the component.
 */
declare const ArgsTable$1: FC<ArgsTableProps$1>;

interface CommonProps {
    story: PreparedStory;
    inline: boolean;
    primary: boolean;
}
interface InlineStoryProps extends CommonProps {
    inline: true;
    height?: string;
    autoplay: boolean;
    forceInitialArgs: boolean;
    renderStoryToElement: DocsContextProps['renderStoryToElement'];
}
interface IFrameStoryProps extends CommonProps {
    inline: false;
    height: string;
}
type StoryProps$2 = InlineStoryProps | IFrameStoryProps;
/**
 * A story element, either rendered inline or in an iframe,
 * with configurable height.
 */
declare const Story$1: FunctionComponent<StoryProps$2>;

interface TypesetProps {
    fontFamily?: string;
    fontSizes: string[];
    fontWeight?: number;
    sampleText?: string;
}
/**
 * Convenient styleguide documentation showing examples of type
 * with different sizes and weights and configurable sample text.
 */
declare const Typeset: FC<TypesetProps>;

type Colors = string[] | {
    [key: string]: string;
};
interface ColorItemProps {
    title: string;
    subtitle: string;
    colors: Colors;
}
/**
 * A single color row your styleguide showing title, subtitle and one or more colors, used
 * as a child of `ColorPalette`.
 */
declare const ColorItem: FunctionComponent<ColorItemProps>;
interface ColorPaletteProps {
    children?: React.ReactNode;
}
/**
 * Styleguide documentation for colors, including names, captions, and color swatches,
 * all specified as `ColorItem` children of this wrapper component.
 */
declare const ColorPalette: FunctionComponent<ColorPaletteProps>;

interface IconItemProps {
    name: string;
    children?: React.ReactNode;
}
/**
 * An individual icon with a caption and an example (passed as `children`).
 */
declare const IconItem: FunctionComponent<IconItemProps>;
interface IconGalleryProps {
    children?: React.ReactNode;
}
/**
 * Show a grid of icons, as specified by `IconItem`.
 */
declare const IconGallery: FunctionComponent<IconGalleryProps>;

declare const anchorBlockIdFromId: (storyId: string) => string;
interface AnchorProps {
    storyId: string;
}
declare const Anchor: FC<PropsWithChildren<AnchorProps>>;

type ArgTypesParameters = {
    include?: PropDescriptor;
    exclude?: PropDescriptor;
    sort?: SortType;
};
type ArgTypesProps = ArgTypesParameters & {
    of?: Renderer['component'] | ModuleExports;
};
declare const ArgTypes: FC<ArgTypesProps>;

declare const PRIMARY_STORY = "^";
type Component = any;
type DocsStoryProps = {
    of: ModuleExport;
    expanded?: boolean;
    withToolbar?: boolean;
    __forceInitialArgs?: boolean;
    __primary?: boolean;
};

interface BaseProps {
    include?: PropDescriptor;
    exclude?: PropDescriptor;
    sort?: SortType;
}
type OfProps = BaseProps & {
    of: '^' | Component;
};
type ComponentsProps = BaseProps & {
    parameters: Parameters$1;
    components: {
        [label: string]: Component;
    };
};
type StoryProps$1 = BaseProps & {
    story: '.' | '^' | string;
    showComponent?: boolean;
};
type ArgsTableProps = BaseProps | OfProps | ComponentsProps | StoryProps$1;
declare const extractComponentArgTypes: (component: Component, parameters: Parameters$1, include?: PropDescriptor, exclude?: PropDescriptor) => StrictArgTypes;
declare const getComponent: (props: ArgsTableProps, component: Component) => Component;
declare const StoryTable: FC<StoryProps$1 & {
    component: Component;
    subcomponents: Record<string, Component>;
}>;
declare const ComponentsTable: FC<ComponentsProps>;
declare const ArgsTable: FC<ArgsTableProps>;

declare const DocsContext: Context<DocsContextProps<Renderer$1>>;

type ArgsHash = string;
declare function argsHash(args: Args$1): ArgsHash;
interface SourceItem {
    code: string;
    format?: SyntaxHighlighterFormatTypes;
}
type StorySources = Record<StoryId, Record<ArgsHash, SourceItem>>;
interface SourceContextProps {
    sources: StorySources;
    setSource?: (id: StoryId, item: SourceItem) => void;
}
declare const SourceContext: Context<SourceContextProps>;
declare const UNKNOWN_ARGS_HASH = "--unknown--";
declare const SourceContainer: FC<PropsWithChildren<{
    channel: Channel;
}>>;

declare enum SourceState {
    OPEN = "open",
    CLOSED = "closed",
    NONE = "none"
}
type SourceParameters = SourceCodeProps & {
    /**
     * Where to read the source code from, see `SourceType`
     */
    type?: SourceType;
    /**
     * Transform the detected source for display
     *
     * @deprecated use `transform` prop instead
     */
    transformSource?: (code: string, storyContext: StoryContextForLoaders) => string;
    /**
     * Transform the detected source for display
     */
    transform?: (code: string, storyContext: StoryContextForLoaders) => string;
    /**
     * Internal: set by our CSF loader (`enrichCsf` in `@storybook/csf-tools`).
     */
    originalSource?: string;
};
type SourceProps = SourceParameters & {
    /**
     * Pass the export defining a story to render its source
     *
     * ```jsx
     * import { Source } from '@storybook/blocks';
     * import * as ButtonStories from './Button.stories';
     *
     * <Source of={ButtonStories.Primary} />
     * ```
     */
    of?: ModuleExport;
    /** @deprecated use of={storyExport} instead */
    id?: string;
    /** @deprecated use of={storyExport} instead */
    ids?: string[];
    /**
     * Internal prop to control if a story re-renders on args updates
     */
    __forceInitialArgs?: boolean;
};
type SourceStateProps = {
    state: SourceState;
};
type PureSourceProps = ComponentProps<typeof Source$1>;
declare const useSourceProps: (props: SourceProps, docsContext: DocsContextProps<any>, sourceContext: SourceContextProps) => PureSourceProps & SourceStateProps;
/**
 * Story source doc block renders source code if provided,
 * or the source for a story if `storyId` is provided, or
 * the source for the current story if nothing is provided.
 */
declare const Source: FC<SourceProps>;

type PureStoryProps = ComponentProps<typeof Story$1>;
/**
 * Props to define a story
 *
 * @deprecated Define stories in CSF files
 */
type StoryDefProps = StoryAnnotations;
/**
 * Props to reference another story
 */
type StoryRefProps = {
    /**
     * @deprecated Use of={storyExport} instead
     */
    id?: string;
    /**
     * @deprecated Use of={storyExport} and define the story in the CSF file
     */
    story?: StoryAnnotations;
    /**
     * Pass the export defining a story to render that story
     *
     * ```jsx
     * import { Meta, Story } from '@storybook/blocks';
     * import * as ButtonStories from './Button.stories';
     *
     * <Meta of={ButtonStories} />
     * <Story of={ButtonStories.Primary} />
     * ```
     */
    of?: ModuleExport;
    /**
     * Pass all exports of the CSF file if this MDX file is unattached
     *
     * ```jsx
     * import { Story } from '@storybook/blocks';
     * import * as ButtonStories from './Button.stories';
     *
     * <Story of={ButtonStories.Primary} meta={ButtonStories} />
     * ```
     */
    meta?: ModuleExports;
};
type StoryParameters = {
    /**
     * Render the story inline or in an iframe
     */
    inline?: boolean;
    /**
     * When rendering in an iframe (`inline={false}`), set the story height
     */
    height?: string;
    /**
     * Whether to run the story's play function
     */
    autoplay?: boolean;
    /**
     * Internal prop to control if a story re-renders on args updates
     */
    __forceInitialArgs?: boolean;
    /**
     * Internal prop if this story is the primary story
     */
    __primary?: boolean;
};
type StoryProps = (StoryDefProps | StoryRefProps) & StoryParameters;
declare const getStoryId: (props: StoryProps, context: DocsContextProps) => StoryId;
declare const getStoryProps: <TFramework extends Renderer$1>(props: StoryParameters, story: PreparedStory<TFramework>, context: DocsContextProps<TFramework>) => PureStoryProps;
declare const Story: FC<StoryProps>;

type DeprecatedCanvasProps = {
    /**
     * @deprecated multiple stories are not supported
     */
    isColumn?: boolean;
    /**
     * @deprecated multiple stories are not supported
     */
    columns?: number;
    /**
     * @deprecated use `sourceState` instead
     */
    withSource?: SourceState;
    /**
     * @deprecated use `source.code` instead
     */
    mdxSource?: string;
    /**
     * @deprecated reference stories with the `of` prop instead
     */
    children?: ReactNode;
};
type CanvasProps = Pick<PreviewProps, 'withToolbar' | 'additionalActions' | 'className'> & {
    /**
     * Pass the export defining a story to render that story
     *
     * ```jsx
     * import { Meta, Canvas } from '@storybook/blocks';
     * import * as ButtonStories from './Button.stories';
     *
     * <Meta of={ButtonStories} />
     * <Canvas of={ButtonStories.Primary} />
     * ```
     */
    of?: ModuleExport;
    /**
     * Pass all exports of the CSF file if this MDX file is unattached
     *
     * ```jsx
     * import { Canvas } from '@storybook/blocks';
     * import * as ButtonStories from './Button.stories';
     *
     * <Canvas of={ButtonStories.Primary} meta={ButtonStories} />
     * ```
     */
    meta?: ModuleExports;
    /**
     * Specify the initial state of the source panel
     * hidden: the source panel is hidden by default
     * shown: the source panel is shown by default
     * none: the source panel is not available and the button to show it is hidden
     * @default 'hidden'
     */
    sourceState?: 'hidden' | 'shown' | 'none';
    /**
     * how to layout the story within the canvas
     * padded: the story has padding within the canvas
     * fullscreen: the story is rendered edge to edge within the canvas
     * centered: the story is centered within the canvas
     * @default 'padded'
     */
    layout?: Layout;
    /**
     * @see {SourceProps}
     */
    source?: Omit<SourceProps, 'dark'>;
    /**
     * @see {StoryProps}
     */
    story?: Pick<StoryProps, 'inline' | 'height' | 'autoplay' | '__forceInitialArgs' | '__primary'>;
};
declare const Canvas: FC<CanvasProps & DeprecatedCanvasProps>;

type ControlsParameters = {
    include?: PropDescriptor;
    exclude?: PropDescriptor;
    sort?: SortType;
};
type ControlsProps = ControlsParameters & {
    of?: Renderer['component'] | ModuleExports;
};
declare const Controls: FC<ControlsProps>;

type Of = Parameters<DocsContextProps['resolveOf']>[0];
/**
 * A hook to resolve the `of` prop passed to a block.
 * will return the resolved module
 * if the resolved module is a meta it will include a preparedMeta property similar to a preparedStory
 * if the resolved module is a component it will include the project annotations
 */
declare const useOf: <TType extends ResolvedModuleExportType>(moduleExportOrType: Of, validTypes?: TType[]) => ResolvedModuleExportFromType<TType>;

declare enum DescriptionType {
    INFO = "info",
    NOTES = "notes",
    DOCGEN = "docgen",
    AUTO = "auto"
}
interface DescriptionProps {
    /**
     * Specify where to get the description from. Can be a component, a CSF file or a story.
     * If not specified, the description will be extracted from the meta of the attached CSF file.
     */
    of?: Of;
    /**
     * @deprecated Manually specifying description type is deprecated. See https://github.com/storybookjs/storybook/blob/next/MIGRATION.md#description-block-parametersnotes-and-parametersinfo
     */
    type?: DescriptionType;
    /**
     * @deprecated The 'markdown' prop on the Description block is deprecated. See https://github.com/storybookjs/storybook/blob/next/MIGRATION.md#description-block-parametersnotes-and-parametersinfo
     */
    markdown?: string;
    /**
     * @deprecated The 'children' prop on the Description block is deprecated. See https://github.com/storybookjs/storybook/blob/next/MIGRATION.md#description-block-parametersnotes-and-parametersinfo
     */
    children?: string;
}
declare const DescriptionContainer: FC<DescriptionProps>;

type DocsProps<TRenderer extends Renderer$1 = Renderer$1> = {
    docsParameter: Parameters$1;
    context: DocsContextProps<TRenderer>;
};
declare function Docs<TRenderer extends Renderer$1 = Renderer$1>({ context, docsParameter, }: DocsProps<TRenderer>): React.JSX.Element;

declare const DocsPage: FC;

interface DocsContainerProps<TFramework extends Renderer$1 = Renderer$1> {
    context: DocsContextProps<TFramework>;
    theme?: ThemeVars;
}
declare const DocsContainer: FC<PropsWithChildren<DocsContainerProps>>;

declare const DocsStory: FC<DocsStoryProps>;

type ExternalDocsProps<TRenderer extends Renderer$1 = Renderer$1> = {
    projectAnnotationsList: ProjectAnnotations<TRenderer>[];
};
declare function ExternalDocs<TRenderer extends Renderer$1 = Renderer$1>({ projectAnnotationsList, children, }: PropsWithChildren<ExternalDocsProps<TRenderer>>): React.JSX.Element;

declare const ExternalDocsContainer: React.FC<{
    projectAnnotations: any;
}>;

interface HeadingProps {
    disableAnchor?: boolean;
}
declare const Heading: FC<PropsWithChildren<HeadingProps>>;

type MarkdownProps = typeof PureMarkdown extends React.ComponentType<infer Props> ? Props : never;
declare const Markdown: (props: MarkdownProps) => React.JSX.Element;

type MetaProps = BaseAnnotations & {
    of?: ModuleExports;
    title?: string;
};
/**
 * This component is used to declare component metadata in docs
 * and gets transformed into a default export underneath the hood.
 */
declare const Meta: FC<MetaProps>;

interface PrimaryProps {
    /**
     * @deprecated Primary block should only be used to render the primary story, which is automatically found.
     */
    name?: string;
    /**
     * Specify where to get the primary story from.
     */
    of?: Of;
}
declare const Primary: FC<PrimaryProps>;

interface StoriesProps {
    title?: JSX.Element | string;
    includePrimary?: boolean;
}
declare const Stories: FC<StoriesProps>;

declare const Subheading: FC<PropsWithChildren<HeadingProps>>;

interface SubtitleProps {
    children?: ReactNode;
}
declare const Subtitle: FunctionComponent<SubtitleProps>;

interface TitleProps {
    children?: ReactNode;
}
declare const extractTitle: (title: ComponentTitle) => string;
declare const Title: FunctionComponent<TitleProps>;

declare const Unstyled: React.FC<React.DetailedHTMLProps<React.HTMLAttributes<HTMLDivElement>, HTMLDivElement>>;

declare const Wrapper: FC<React.DetailedHTMLProps<React.HTMLAttributes<HTMLDivElement>, HTMLDivElement>>;

declare const assertIsFn: (val: any) => any;
declare const AddContext: FC<DocsContextProps>;
interface CodeOrSourceMdxProps {
    className?: string;
}
declare const CodeOrSourceMdx: FC<CodeOrSourceMdxProps>;
interface AnchorMdxProps {
    href: string;
    target: string;
}
declare const AnchorMdx: FC<PropsWithChildren<AnchorMdxProps>>;
interface HeaderMdxProps {
    as: string;
    id: string;
}
declare const HeaderMdx: FC<PropsWithChildren<HeaderMdxProps>>;
declare const HeadersMdx: {};

interface ControlProps<T> {
    name: string;
    value?: T;
    defaultValue?: T;
    argType?: ArgType;
    onChange: (value: T) => T | void;
    onFocus?: (evt: any) => void;
    onBlur?: (evt: any) => void;
}
type BooleanValue = boolean;
interface BooleanConfig {
}
type ColorValue = string;
type PresetColor = ColorValue | {
    color: ColorValue;
    title?: string;
};
interface ColorConfig {
    presetColors?: PresetColor[];
    /**
     * Whether the color picker should be open by default when rendered.
     * @default false
     */
    startOpen?: boolean;
}
type DateValue = Date | number;
interface DateConfig {
}
type NumberValue = number;
interface NumberConfig {
    min?: number;
    max?: number;
    step?: number;
}
type RangeConfig = NumberConfig;
type ObjectValue = any;
interface ObjectConfig {
}
type OptionsSingleSelection = any;
type OptionsMultiSelection = any[];
type OptionsSelection = OptionsSingleSelection | OptionsMultiSelection;
type OptionsArray = any[];
type OptionsObject = Record<string, any>;
type Options = OptionsArray | OptionsObject;
type OptionsControlType = 'radio' | 'inline-radio' | 'check' | 'inline-check' | 'select' | 'multi-select';
interface OptionsConfig {
    labels: Record<any, string>;
    type: OptionsControlType;
}
interface NormalizedOptionsConfig {
    options: OptionsObject;
}
type TextValue = string;
interface TextConfig {
    maxLength?: number;
}
type ControlType = 'array' | 'boolean' | 'color' | 'date' | 'number' | 'range' | 'object' | OptionsControlType | 'text';
type Control = BooleanConfig | ColorConfig | DateConfig | NumberConfig | ObjectConfig | OptionsConfig | RangeConfig | TextConfig;

type ColorControlProps = ControlProps<ColorValue> & ColorConfig;

type BooleanProps = ControlProps<BooleanValue> & BooleanConfig;
/**
 * # Boolean Control
 * Renders a switch toggle with "True" or "False".
 * or if the value is `undefined`, renders a button to set the boolean.
 *
 * ## Example usage
 *
 * ```
 *
 * <BooleanControl name="isTrue" value={value} onChange={handleValueChange}/>
 * ```
 */
declare const BooleanControl: FC<BooleanProps>;

declare const parseDate: (value: string) => Date;
declare const parseTime: (value: string) => Date;
declare const formatDate: (value: Date | number) => string;
declare const formatTime: (value: Date | number) => string;
type DateProps = ControlProps<DateValue> & DateConfig;
declare const DateControl: FC<DateProps>;

type NumberProps = ControlProps<NumberValue | null> & NumberConfig;
declare const parse: (value: string) => number;
declare const format: (value: NumberValue) => string;
declare const NumberControl: FC<NumberProps>;

type OptionsProps = ControlProps<OptionsSelection> & OptionsConfig;
declare const OptionsControl: FC<OptionsProps>;

type ObjectProps = ControlProps<ObjectValue> & ObjectConfig & {
    theme: any;
};
declare const ObjectControl: FC<ObjectProps>;

type RangeProps = ControlProps<NumberValue | null> & RangeConfig;
declare const RangeControl: FC<RangeProps>;

type TextProps = ControlProps<TextValue | undefined> & TextConfig;
declare const TextControl: FC<TextProps>;

interface FilesControlProps extends ControlProps<string[]> {
    /**
     * The accept attribute value is a string that defines the file types the file input should accept. This string is a comma-separated list of unique file type specifiers.
     * @example
     * *\/*
     * @example
     * .webm,video/webm
     * @example
     * .doc,.docx,application/msword
     * @defaultValue `image/*`
     * @see https://developer.mozilla.org/en-US/docs/Web/HTML/Element/input/file#accept
     */
    accept?: string;
}
declare const FilesControl: FC<FilesControlProps>;

type ColorProps = ColorControlProps;
declare const LazyColorControl: React.LazyExoticComponent<React.FC<ColorControlProps>>;
declare const ColorControl: (props: ComponentProps<typeof LazyColorControl>) => React.JSX.Element;

export { AddContext, Anchor, AnchorMdx, AnchorProps, ArgTypes, ArgsTable, BooleanConfig, BooleanControl, BooleanProps, BooleanValue, Canvas, CodeOrSourceMdx, ColorConfig, ColorControl, ColorItem, ColorPalette, ColorProps, ColorValue, Component, ComponentsTable, Control, ControlProps, ControlType, Controls, DateConfig, DateControl, DateProps, DateValue, DescriptionContainer as Description, DescriptionType, Docs, DocsContainer, DocsContainerProps, DocsContext, DocsPage, DocsProps, DocsStory, DocsStoryProps, ExternalDocs, ExternalDocsContainer, ExternalDocsProps, FilesControl, FilesControlProps, HeaderMdx, HeadersMdx, Heading, HeadingProps, IconGallery, IconItem, Markdown, Meta, NormalizedOptionsConfig, NumberConfig, NumberControl, NumberValue, ObjectConfig, ObjectControl, ObjectProps, ObjectValue, Of, Options, OptionsArray, OptionsConfig, OptionsControl, OptionsControlType, OptionsMultiSelection, OptionsObject, OptionsProps, OptionsSelection, OptionsSingleSelection, PRIMARY_STORY, PresetColor, Primary, ArgsTable$1 as PureArgsTable, RangeConfig, RangeControl, SortType, Source, SourceContainer, SourceContext, SourceContextProps, SourceItem, SourceProps, SourceState, Stories, Story, StoryProps, StorySources, StoryTable, Subheading, Subtitle, TextConfig, TextControl, TextProps, TextValue, Title, Typeset, UNKNOWN_ARGS_HASH, Unstyled, Wrapper, anchorBlockIdFromId, argsHash, assertIsFn, extractComponentArgTypes, extractTitle, format, formatDate, formatTime, getComponent, getStoryId, getStoryProps, parse, parseDate, parseTime, useOf, useSourceProps };
