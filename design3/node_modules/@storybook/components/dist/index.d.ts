import * as React$1 from 'react';
import React__default, { ComponentProps, FC, AnchorHTMLAttributes, MouseEvent, ReactNode, FunctionComponent, ReactElement, Component, RefObject, SyntheticEvent, DetailedHTMLProps, ButtonHTMLAttributes, ElementType } from 'react';
import * as _storybook_theming from '@storybook/theming';
import { Theme, CSSObject } from '@storybook/theming';
import { BuiltInParserName } from 'prettier';
import * as react_textarea_autosize from 'react-textarea-autosize';
import { Addon_RenderOptions } from '@storybook/types';

declare const A: _storybook_theming.StyledComponent<React$1.AnchorHTMLAttributes<HTMLAnchorElement> & {
    children?: React$1.ReactNode;
} & {
    theme?: _storybook_theming.Theme;
}, {}, {}>;

declare const Blockquote: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.BlockquoteHTMLAttributes<HTMLElement>, HTMLElement>, {}>;

declare const DefaultCodeBlock: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React__default.ElementType<any>;
}, React__default.DetailedHTMLProps<React__default.HTMLAttributes<HTMLElement>, HTMLElement>, {}>;
declare const Code: ({ className, children, ...props }: ComponentProps<typeof DefaultCodeBlock>) => React__default.JSX.Element;

declare const Div: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLDivElement>, HTMLDivElement>, {}>;

declare const DL: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLDListElement>, HTMLDListElement>, {}>;

declare const H1: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLHeadingElement>, HTMLHeadingElement>, {}>;

declare const H2: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLHeadingElement>, HTMLHeadingElement>, {}>;

declare const H3: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLHeadingElement>, HTMLHeadingElement>, {}>;

declare const H4: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLHeadingElement>, HTMLHeadingElement>, {}>;

declare const H5: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLHeadingElement>, HTMLHeadingElement>, {}>;

declare const H6: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLHeadingElement>, HTMLHeadingElement>, {}>;

declare const HR: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLHRElement>, HTMLHRElement>, {}>;

declare const Img: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.ImgHTMLAttributes<HTMLImageElement>, HTMLImageElement>, {}>;

declare const LI: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.LiHTMLAttributes<HTMLLIElement>, HTMLLIElement>, {}>;

declare const OL: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.OlHTMLAttributes<HTMLOListElement>, HTMLOListElement>, {}>;

declare const P: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLParagraphElement>, HTMLParagraphElement>, {}>;

declare const Pre: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLPreElement>, HTMLPreElement>, {}>;

declare const Span: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLSpanElement>, HTMLSpanElement>, {}>;

declare const Table: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.TableHTMLAttributes<HTMLTableElement>, HTMLTableElement>, {}>;

declare const TT: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLTitleElement>, HTMLTitleElement>, {}>;

declare const UL: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLUListElement>, HTMLUListElement>, {}>;

interface BadgeProps {
    status: 'positive' | 'negative' | 'neutral' | 'warning' | 'critical';
}
declare const Badge: FC<BadgeProps>;

interface LinkStylesProps {
    secondary?: boolean;
    tertiary?: boolean;
    nochrome?: boolean;
    inverse?: boolean;
    isButton?: boolean;
}
interface LinkInnerProps {
    withArrow?: boolean;
    containsIcon?: boolean;
}
type AProps = AnchorHTMLAttributes<HTMLAnchorElement>;
interface LinkProps extends LinkInnerProps, LinkStylesProps {
    cancel?: boolean;
    className?: string;
    style?: object;
    onClick?: (e: MouseEvent) => void;
    href?: string;
}
declare const Link$1: FC<LinkProps & AProps>;

declare const DocumentWrapper: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLDivElement>, HTMLDivElement>, {}>;

interface SyntaxHighlighterRendererProps {
    rows: any[];
    stylesheet: string;
    useInlineStyles: boolean;
}
type SyntaxHighlighterRenderer = (props: SyntaxHighlighterRendererProps) => ReactNode;
interface SyntaxHighlighterCustomProps {
    language: string;
    copyable?: boolean;
    bordered?: boolean;
    padded?: boolean;
    format?: SyntaxHighlighterFormatTypes;
    formatter?: (type: SyntaxHighlighterFormatTypes, source: string) => string;
    className?: string;
    renderer?: SyntaxHighlighterRenderer;
}
type SyntaxHighlighterFormatTypes = boolean | 'dedent' | BuiltInParserName;
type LineTagPropsFunction = (lineNumber: number) => React.HTMLProps<HTMLElement>;
interface SyntaxHighlighterBaseProps {
    children?: React.ReactNode;
    codeTagProps?: React.HTMLProps<HTMLElement>;
    customStyle?: any;
    language?: string;
    lineNumberStyle?: any;
    lineProps?: LineTagPropsFunction | React.HTMLProps<HTMLElement>;
    showLineNumbers?: boolean;
    startingLineNumber?: number;
    wrapLongLines?: boolean;
    style?: any;
    useInlineStyles?: boolean;
}
type SyntaxHighlighterProps = SyntaxHighlighterBaseProps & SyntaxHighlighterCustomProps;

declare const LazySyntaxHighlighter: React__default.LazyExoticComponent<(props: SyntaxHighlighterBaseProps & SyntaxHighlighterCustomProps) => React__default.JSX.Element>;
declare const LazySyntaxHighlighterWithFormatter: React__default.LazyExoticComponent<(props: SyntaxHighlighterBaseProps & SyntaxHighlighterCustomProps) => React__default.JSX.Element>;
declare const SyntaxHighlighter: {
    (props: ComponentProps<typeof LazySyntaxHighlighter> | ComponentProps<typeof LazySyntaxHighlighterWithFormatter>): React__default.JSX.Element;
    registerLanguage(name: string, func: any): void;
};

declare function createCopyToClipboardFunction(): (text: string) => Promise<void>;

interface ActionItem {
    title: string | JSX.Element;
    className?: string;
    onClick: (e: MouseEvent<HTMLButtonElement>) => void;
    disabled?: boolean;
}
interface ActionBarProps {
    actionItems: ActionItem[];
}
declare const ActionBar: FC<ActionBarProps>;

interface SpacedProps {
    children?: React__default.ReactNode;
    col?: number;
    row?: number;
    outer?: number | boolean;
}
declare const Spaced: FC<SpacedProps>;

interface PlaceholderProps {
    children?: React__default.ReactNode;
}
declare const Placeholder: FunctionComponent<PlaceholderProps>;

interface ScrollAreaProps {
    children?: React__default.ReactNode;
    horizontal?: boolean;
    vertical?: boolean;
    className?: string;
    offset?: number;
    scrollbarSize?: number;
}
declare const ScrollArea: FC<ScrollAreaProps>;

type ZoomProps = {
    scale: number;
    children: ReactElement | ReactElement[];
};
declare function ZoomElement({ scale, children }: ZoomProps): React__default.JSX.Element;

type IZoomIFrameProps = {
    scale: number;
    children: ReactElement<HTMLIFrameElement>;
    iFrameRef: RefObject<HTMLIFrameElement>;
    active?: boolean;
};
declare class ZoomIFrame extends Component<IZoomIFrameProps> {
    iframe: HTMLIFrameElement;
    componentDidMount(): void;
    shouldComponentUpdate(nextProps: IZoomIFrameProps): boolean;
    setIframeInnerZoom(scale: number): void;
    setIframeZoom(scale: number): void;
    render(): React__default.JSX.Element;
}

declare const Zoom: {
    Element: typeof ZoomElement;
    IFrame: typeof ZoomIFrame;
};

declare const ErrorFormatter: FC<{
    error: Error;
}>;

declare const ButtonWrapper: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React__default.ElementType<any>;
} & {
    isLink?: boolean;
    primary?: boolean;
    secondary?: boolean;
    tertiary?: boolean;
    gray?: boolean;
    inForm?: boolean;
    disabled?: boolean;
    small?: boolean;
    outline?: boolean;
    containsIcon?: boolean;
    children?: ReactNode;
    href?: string;
}, React__default.DetailedHTMLProps<React__default.ButtonHTMLAttributes<HTMLButtonElement>, HTMLButtonElement>, {}>;
declare const Button: FC<ComponentProps<typeof ButtonWrapper>>;

type Sizes = '100%' | 'flex' | 'auto';
type Alignments = 'end' | 'center' | 'start';
type ValidationStates = 'valid' | 'error' | 'warn';

interface FieldProps {
    children?: ReactNode;
    label?: ReactNode;
}

declare const Form: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.FormHTMLAttributes<HTMLFormElement>, HTMLFormElement>, {}> & {
    Field: React$1.FC<FieldProps>;
    Input: _storybook_theming.StyledComponent<Pick<Omit<React$1.HTMLProps<HTMLInputElement>, "height" | "align" | "size" | "valid"> & {
        size?: Sizes;
        align?: Alignments;
        valid?: ValidationStates;
        height?: number;
    }, "download" | "href" | "hrefLang" | "media" | "target" | "type" | "form" | "list" | "cite" | "cellPadding" | "cellSpacing" | "summary" | "width" | "alt" | "crossOrigin" | "height" | "sizes" | "src" | "srcSet" | "useMap" | "value" | "reversed" | "start" | "data" | "label" | "slot" | "span" | "style" | "title" | "pattern" | "className" | "color" | "content" | "translate" | "default" | "wrap" | "hidden" | "children" | "defaultChecked" | "defaultValue" | "suppressContentEditableWarning" | "suppressHydrationWarning" | "accessKey" | "autoFocus" | "contentEditable" | "contextMenu" | "dir" | "draggable" | "id" | "lang" | "nonce" | "placeholder" | "spellCheck" | "tabIndex" | "radioGroup" | "role" | "about" | "datatype" | "inlist" | "prefix" | "property" | "rel" | "resource" | "rev" | "typeof" | "vocab" | "autoCapitalize" | "autoCorrect" | "autoSave" | "itemProp" | "itemScope" | "itemType" | "itemID" | "itemRef" | "results" | "security" | "unselectable" | "inputMode" | "is" | "aria-activedescendant" | "aria-atomic" | "aria-autocomplete" | "aria-busy" | "aria-checked" | "aria-colcount" | "aria-colindex" | "aria-colspan" | "aria-controls" | "aria-current" | "aria-describedby" | "aria-details" | "aria-disabled" | "aria-dropeffect" | "aria-errormessage" | "aria-expanded" | "aria-flowto" | "aria-grabbed" | "aria-haspopup" | "aria-hidden" | "aria-invalid" | "aria-keyshortcuts" | "aria-label" | "aria-labelledby" | "aria-level" | "aria-live" | "aria-modal" | "aria-multiline" | "aria-multiselectable" | "aria-orientation" | "aria-owns" | "aria-placeholder" | "aria-posinset" | "aria-pressed" | "aria-readonly" | "aria-relevant" | "aria-required" | "aria-roledescription" | "aria-rowcount" | "aria-rowindex" | "aria-rowspan" | "aria-selected" | "aria-setsize" | "aria-sort" | "aria-valuemax" | "aria-valuemin" | "aria-valuenow" | "aria-valuetext" | "dangerouslySetInnerHTML" | "onCopy" | "onCopyCapture" | "onCut" | "onCutCapture" | "onPaste" | "onPasteCapture" | "onCompositionEnd" | "onCompositionEndCapture" | "onCompositionStart" | "onCompositionStartCapture" | "onCompositionUpdate" | "onCompositionUpdateCapture" | "onFocus" | "onFocusCapture" | "onBlur" | "onBlurCapture" | "onChange" | "onChangeCapture" | "onBeforeInput" | "onBeforeInputCapture" | "onInput" | "onInputCapture" | "onReset" | "onResetCapture" | "onSubmit" | "onSubmitCapture" | "onInvalid" | "onInvalidCapture" | "onLoad" | "onLoadCapture" | "onError" | "onErrorCapture" | "onKeyDown" | "onKeyDownCapture" | "onKeyPress" | "onKeyPressCapture" | "onKeyUp" | "onKeyUpCapture" | "onAbort" | "onAbortCapture" | "onCanPlay" | "onCanPlayCapture" | "onCanPlayThrough" | "onCanPlayThroughCapture" | "onDurationChange" | "onDurationChangeCapture" | "onEmptied" | "onEmptiedCapture" | "onEncrypted" | "onEncryptedCapture" | "onEnded" | "onEndedCapture" | "onLoadedData" | "onLoadedDataCapture" | "onLoadedMetadata" | "onLoadedMetadataCapture" | "onLoadStart" | "onLoadStartCapture" | "onPause" | "onPauseCapture" | "onPlay" | "onPlayCapture" | "onPlaying" | "onPlayingCapture" | "onProgress" | "onProgressCapture" | "onRateChange" | "onRateChangeCapture" | "onSeeked" | "onSeekedCapture" | "onSeeking" | "onSeekingCapture" | "onStalled" | "onStalledCapture" | "onSuspend" | "onSuspendCapture" | "onTimeUpdate" | "onTimeUpdateCapture" | "onVolumeChange" | "onVolumeChangeCapture" | "onWaiting" | "onWaitingCapture" | "onAuxClick" | "onAuxClickCapture" | "onClick" | "onClickCapture" | "onContextMenu" | "onContextMenuCapture" | "onDoubleClick" | "onDoubleClickCapture" | "onDrag" | "onDragCapture" | "onDragEnd" | "onDragEndCapture" | "onDragEnter" | "onDragEnterCapture" | "onDragExit" | "onDragExitCapture" | "onDragLeave" | "onDragLeaveCapture" | "onDragOver" | "onDragOverCapture" | "onDragStart" | "onDragStartCapture" | "onDrop" | "onDropCapture" | "onMouseDown" | "onMouseDownCapture" | "onMouseEnter" | "onMouseLeave" | "onMouseMove" | "onMouseMoveCapture" | "onMouseOut" | "onMouseOutCapture" | "onMouseOver" | "onMouseOverCapture" | "onMouseUp" | "onMouseUpCapture" | "onSelect" | "onSelectCapture" | "onTouchCancel" | "onTouchCancelCapture" | "onTouchEnd" | "onTouchEndCapture" | "onTouchMove" | "onTouchMoveCapture" | "onTouchStart" | "onTouchStartCapture" | "onPointerDown" | "onPointerDownCapture" | "onPointerMove" | "onPointerMoveCapture" | "onPointerUp" | "onPointerUpCapture" | "onPointerCancel" | "onPointerCancelCapture" | "onPointerEnter" | "onPointerEnterCapture" | "onPointerLeave" | "onPointerLeaveCapture" | "onPointerOver" | "onPointerOverCapture" | "onPointerOut" | "onPointerOutCapture" | "onGotPointerCapture" | "onGotPointerCaptureCapture" | "onLostPointerCapture" | "onLostPointerCaptureCapture" | "onScroll" | "onScrollCapture" | "onWheel" | "onWheelCapture" | "onAnimationStart" | "onAnimationStartCapture" | "onAnimationEnd" | "onAnimationEndCapture" | "onAnimationIteration" | "onAnimationIterationCapture" | "onTransitionEnd" | "onTransitionEndCapture" | "disabled" | "key" | "rows" | "align" | "as" | "max" | "method" | "min" | "name" | "controls" | "size" | "open" | "multiple" | "step" | "formAction" | "formEncType" | "formMethod" | "formNoValidate" | "formTarget" | "valid" | "accept" | "acceptCharset" | "action" | "allowFullScreen" | "allowTransparency" | "async" | "autoComplete" | "autoPlay" | "capture" | "charSet" | "challenge" | "checked" | "classID" | "cols" | "colSpan" | "coords" | "dateTime" | "defer" | "encType" | "frameBorder" | "headers" | "high" | "htmlFor" | "httpEquiv" | "integrity" | "keyParams" | "keyType" | "kind" | "loop" | "low" | "manifest" | "marginHeight" | "marginWidth" | "maxLength" | "mediaGroup" | "minLength" | "muted" | "noValidate" | "optimum" | "playsInline" | "poster" | "preload" | "readOnly" | "required" | "rowSpan" | "sandbox" | "scope" | "scoped" | "scrolling" | "seamless" | "selected" | "shape" | "srcDoc" | "srcLang" | "wmode"> & React$1.RefAttributes<any> & {
        theme?: _storybook_theming.Theme;
    } & {
        size?: Sizes;
        align?: Alignments;
        valid?: ValidationStates;
        height?: number;
    }, {}, {}> & {
        displayName: string;
    };
    Select: _storybook_theming.StyledComponent<Omit<React$1.SelectHTMLAttributes<HTMLSelectElement>, "height" | "align" | "size" | "valid"> & {
        size?: Sizes;
        align?: Alignments;
        valid?: ValidationStates;
        height?: number;
    } & React$1.RefAttributes<any> & {
        theme?: _storybook_theming.Theme;
    }, {}, {}> & {
        displayName: string;
    };
    Textarea: _storybook_theming.StyledComponent<Omit<react_textarea_autosize.TextareaAutosizeProps, "height" | "align" | "size" | "valid"> & {
        size?: Sizes;
        align?: Alignments;
        valid?: ValidationStates;
        height?: number;
    } & React$1.RefAttributes<any> & {
        theme?: _storybook_theming.Theme;
    }, {}, {}> & {
        displayName: string;
    };
    Button: React$1.FC<any>;
};

declare const top: "top";
declare const bottom: "bottom";
declare const right: "right";
declare const left: "left";
declare type BasePlacement = typeof top | typeof bottom | typeof right | typeof left;
declare type VariationPlacement = "top-start" | "top-end" | "bottom-start" | "bottom-end" | "right-start" | "right-end" | "left-start" | "left-end";
declare type AutoPlacement = "auto" | "auto-start" | "auto-end";
declare type Placement = AutoPlacement | BasePlacement | VariationPlacement;
declare const beforeRead: "beforeRead";
declare const read: "read";
declare const afterRead: "afterRead";
declare const beforeMain: "beforeMain";
declare const main: "main";
declare const afterMain: "afterMain";
declare const beforeWrite: "beforeWrite";
declare const write: "write";
declare const afterWrite: "afterWrite";
declare type ModifierPhases = typeof beforeRead | typeof read | typeof afterRead | typeof beforeMain | typeof main | typeof afterMain | typeof beforeWrite | typeof write | typeof afterWrite;

declare type Obj = {
    [key: string]: any;
};
declare type VisualViewport = EventTarget & {
    width: number;
    height: number;
    offsetLeft: number;
    offsetTop: number;
    scale: number;
};
declare type Window = {
    innerHeight: number;
    offsetHeight: number;
    innerWidth: number;
    offsetWidth: number;
    pageXOffset: number;
    pageYOffset: number;
    getComputedStyle: typeof getComputedStyle;
    addEventListener(type: any, listener: any, optionsOrUseCapture?: any): void;
    removeEventListener(type: any, listener: any, optionsOrUseCapture?: any): void;
    Element: Element;
    HTMLElement: HTMLElement;
    Node: Node;
    toString(): "[object Window]";
    devicePixelRatio: number;
    visualViewport?: VisualViewport;
    ShadowRoot: ShadowRoot;
};
declare type Rect = {
    width: number;
    height: number;
    x: number;
    y: number;
};
declare type Offsets = {
    y: number;
    x: number;
};
declare type PositioningStrategy = "absolute" | "fixed";
declare type StateRects = {
    reference: Rect;
    popper: Rect;
};
declare type OffsetData = {
    [key in Placement]?: Offsets;
};
declare type State = {
    elements: {
        reference: Element | VirtualElement;
        popper: HTMLElement;
        arrow?: HTMLElement;
    };
    options: OptionsGeneric<any>;
    placement: Placement;
    strategy: PositioningStrategy;
    orderedModifiers: Array<Modifier<any, any>>;
    rects: StateRects;
    scrollParents: {
        reference: Array<Element | Window | VisualViewport>;
        popper: Array<Element | Window | VisualViewport>;
    };
    styles: {
        [key: string]: Partial<CSSStyleDeclaration>;
    };
    attributes: {
        [key: string]: {
            [key: string]: string | boolean;
        };
    };
    modifiersData: {
        arrow?: {
            x?: number;
            y?: number;
            centerOffset: number;
        };
        hide?: {
            isReferenceHidden: boolean;
            hasPopperEscaped: boolean;
            referenceClippingOffsets: SideObject;
            popperEscapeOffsets: SideObject;
        };
        offset?: OffsetData;
        preventOverflow?: Offsets;
        popperOffsets?: Offsets;
        [key: string]: any;
    };
    reset: boolean;
};
declare type SetAction<S> = S | ((prev: S) => S);
declare type Instance = {
    state: State;
    destroy: () => void;
    forceUpdate: () => void;
    update: () => Promise<Partial<State>>;
    setOptions: (setOptionsAction: SetAction<Partial<OptionsGeneric<any>>>) => Promise<Partial<State>>;
};
declare type ModifierArguments<Options extends Obj> = {
    state: State;
    instance: Instance;
    options: Partial<Options>;
    name: string;
};
declare type Modifier<Name, Options extends Obj> = {
    name: Name;
    enabled: boolean;
    phase: ModifierPhases;
    requires?: Array<string>;
    requiresIfExists?: Array<string>;
    fn: (arg0: ModifierArguments<Options>) => State | void;
    effect?: (arg0: ModifierArguments<Options>) => (() => void) | void;
    options?: Partial<Options>;
    data?: Obj;
};
declare type Options = {
    placement: Placement;
    modifiers: Array<Partial<Modifier<any, any>>>;
    strategy: PositioningStrategy;
    onFirstUpdate?: (arg0: Partial<State>) => void;
};
declare type OptionsGeneric<TModifier> = {
    placement: Placement;
    modifiers: Array<TModifier>;
    strategy: PositioningStrategy;
    onFirstUpdate?: (arg0: Partial<State>) => void;
};
declare type SideObject = {
    top: number;
    left: number;
    right: number;
    bottom: number;
};
declare type VirtualElement = {
    getBoundingClientRect: () => ClientRect | DOMRect;
    contextElement?: Element;
};

declare const createPopper: <TModifier extends Partial<Modifier<any, any>>>(reference: Element | VirtualElement, popper: HTMLElement, options?: Partial<OptionsGeneric<TModifier>>) => Instance;

declare type TriggerType = 'click' | 'double-click' | 'right-click' | 'hover' | 'focus';
declare type Config = {
    /**
     * Whether to close the tooltip when its trigger is out of boundary
     * @default false
     */
    closeOnTriggerHidden?: boolean;
    /**
     * Event or events that trigger the tooltip
     * @default hover
     */
    trigger?: TriggerType | TriggerType[] | null;
    /**
     * Delay in hiding the tooltip (ms)
     * @default 0
     */
    delayHide?: number;
    /**
     * Delay in showing the tooltip (ms)
     * @default 0
     */
    delayShow?: number;
    /**
     * Whether to make the tooltip spawn at cursor position
     * @default false
     */
    followCursor?: boolean;
    /**
     * Options to MutationObserver, used internally for updating
     * tooltip position based on its DOM changes
     * @default  { attributes: true, childList: true, subtree: true }
     */
    mutationObserverOptions?: MutationObserverInit | null;
    /**
     * Whether tooltip is shown by default
     * @default false
     */
    defaultVisible?: boolean;
    /**
     * Used to create controlled tooltip
     */
    visible?: boolean;
    /**
     * Called when the visibility of the tooltip changes
     */
    onVisibleChange?: (state: boolean) => void;
    /**
     * If `true`, a click outside the trigger element closes the tooltip
     * @default true
     */
    closeOnOutsideClick?: boolean;
    /**
     * If `true`, hovering the tooltip will keep it open. Normally tooltip closes when the mouse cursor moves out of
     * the trigger element. If it moves to the tooltip element, the tooltip stays open.
     * @default false
     */
    interactive?: boolean;
    /**
     * Alias for popper.js placement, see https://popper.js.org/docs/v2/constructors/#placement
     */
    placement?: Placement;
    /**
     * Shorthand for popper.js offset modifier, see https://popper.js.org/docs/v2/modifiers/offset/
     * @default [0, 6]
     */
    offset?: [number, number];
};
declare type PopperOptions = Partial<Options> & {
    createPopper?: typeof createPopper;
};

declare const TargetContainer: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React__default.ElementType<any>;
} & {
    trigger: Config['trigger'];
}, React__default.DetailedHTMLProps<React__default.HTMLAttributes<HTMLDivElement>, HTMLDivElement>, {}>;
interface WithHideFn {
    onHide: () => void;
}
interface WithTooltipPureProps extends Omit<Config, 'closeOnOutsideClick'>, Omit<ComponentProps<typeof TargetContainer>, 'trigger'>, PopperOptions {
    svg?: boolean;
    withArrows?: boolean;
    hasChrome?: boolean;
    tooltip: ReactNode | ((p: WithHideFn) => ReactNode);
    children: ReactNode;
    onDoubleClick?: () => void;
    /**
     * @deprecated use `defaultVisible` property instead. This property will be removed in SB 8.0
     */
    tooltipShown?: boolean;
    /**
     * @deprecated use `closeOnOutsideClick` property instead. This property will be removed in SB 8.0
     */
    closeOnClick?: boolean;
    /**
     * @deprecated use `onVisibleChange` property instead. This property will be removed in SB 8.0
     */
    onVisibilityChange?: (visibility: boolean) => void | boolean;
    /**
     * If `true`, a click outside the trigger element closes the tooltip
     * @default false
     */
    closeOnOutsideClick?: boolean;
}

declare const LazyWithTooltip: React__default.LazyExoticComponent<React__default.FC<Omit<WithTooltipPureProps, "onVisibleChange"> & {
    startOpen?: boolean;
    onVisibleChange?: (visible: boolean) => boolean | void;
}>>;
declare const WithTooltip: (props: ComponentProps<typeof LazyWithTooltip>) => React__default.JSX.Element;
declare const LazyWithTooltipPure: React__default.LazyExoticComponent<React__default.FC<WithTooltipPureProps>>;
declare const WithTooltipPure: (props: ComponentProps<typeof LazyWithTooltipPure>) => React__default.JSX.Element;

interface TooltipMessageProps {
    title?: ReactNode;
    desc?: ReactNode;
    links?: {
        title: string;
        href?: string;
        onClick?: () => void;
    }[];
}
declare const TooltipMessage: FC<TooltipMessageProps>;

interface TooltipNoteProps {
    note: string;
}
declare const TooltipNote: FC<TooltipNoteProps>;

declare const icons: {
    readonly user: React__default.JSX.Element;
    readonly useralt: React__default.JSX.Element;
    readonly useradd: React__default.JSX.Element;
    readonly users: React__default.JSX.Element;
    readonly profile: React__default.JSX.Element;
    readonly facehappy: React__default.JSX.Element;
    readonly faceneutral: React__default.JSX.Element;
    readonly facesad: React__default.JSX.Element;
    readonly accessibility: React__default.JSX.Element;
    readonly accessibilityalt: React__default.JSX.Element;
    readonly arrowup: React__default.JSX.Element;
    readonly arrowdown: React__default.JSX.Element;
    readonly arrowleft: React__default.JSX.Element;
    readonly arrowright: React__default.JSX.Element;
    readonly arrowupalt: React__default.JSX.Element;
    readonly arrowdownalt: React__default.JSX.Element;
    readonly arrowleftalt: React__default.JSX.Element;
    readonly arrowrightalt: React__default.JSX.Element;
    readonly expandalt: React__default.JSX.Element;
    readonly collapse: React__default.JSX.Element;
    readonly expand: React__default.JSX.Element;
    readonly unfold: React__default.JSX.Element;
    readonly transfer: React__default.JSX.Element;
    readonly redirect: React__default.JSX.Element;
    readonly undo: React__default.JSX.Element;
    readonly reply: React__default.JSX.Element;
    readonly sync: React__default.JSX.Element;
    readonly upload: React__default.JSX.Element;
    readonly download: React__default.JSX.Element;
    readonly back: React__default.JSX.Element;
    readonly proceed: React__default.JSX.Element;
    readonly refresh: React__default.JSX.Element;
    readonly globe: React__default.JSX.Element;
    readonly compass: React__default.JSX.Element;
    readonly location: React__default.JSX.Element;
    readonly pin: React__default.JSX.Element;
    readonly time: React__default.JSX.Element;
    readonly dashboard: React__default.JSX.Element;
    readonly timer: React__default.JSX.Element;
    readonly home: React__default.JSX.Element;
    readonly admin: React__default.JSX.Element;
    readonly info: React__default.JSX.Element;
    readonly question: React__default.JSX.Element;
    readonly support: React__default.JSX.Element;
    readonly alert: React__default.JSX.Element;
    readonly email: React__default.JSX.Element;
    readonly phone: React__default.JSX.Element;
    readonly link: React__default.JSX.Element;
    readonly unlink: React__default.JSX.Element;
    readonly bell: React__default.JSX.Element;
    readonly rss: React__default.JSX.Element;
    readonly sharealt: React__default.JSX.Element;
    readonly share: React__default.JSX.Element;
    readonly circlehollow: React__default.JSX.Element;
    readonly circle: React__default.JSX.Element;
    readonly bookmarkhollow: React__default.JSX.Element;
    readonly bookmark: React__default.JSX.Element;
    readonly hearthollow: React__default.JSX.Element;
    readonly heart: React__default.JSX.Element;
    readonly starhollow: React__default.JSX.Element;
    readonly star: React__default.JSX.Element;
    readonly certificate: React__default.JSX.Element;
    readonly verified: React__default.JSX.Element;
    readonly thumbsup: React__default.JSX.Element;
    readonly shield: React__default.JSX.Element;
    readonly basket: React__default.JSX.Element;
    readonly beaker: React__default.JSX.Element;
    readonly hourglass: React__default.JSX.Element;
    readonly flag: React__default.JSX.Element;
    readonly cloudhollow: React__default.JSX.Element;
    readonly cloud: React__default.JSX.Element;
    readonly edit: React__default.JSX.Element;
    readonly cog: React__default.JSX.Element;
    readonly nut: React__default.JSX.Element;
    readonly wrench: React__default.JSX.Element;
    readonly ellipsis: React__default.JSX.Element;
    readonly check: React__default.JSX.Element;
    readonly form: React__default.JSX.Element;
    readonly batchdeny: React__default.JSX.Element;
    readonly batchaccept: React__default.JSX.Element;
    readonly controls: React__default.JSX.Element;
    readonly plus: React__default.JSX.Element;
    readonly closeAlt: React__default.JSX.Element;
    readonly cross: React__default.JSX.Element;
    readonly trash: React__default.JSX.Element;
    readonly pinalt: React__default.JSX.Element;
    readonly unpin: React__default.JSX.Element;
    readonly add: React__default.JSX.Element;
    readonly subtract: React__default.JSX.Element;
    readonly close: React__default.JSX.Element;
    readonly delete: React__default.JSX.Element;
    readonly passed: React__default.JSX.Element;
    readonly changed: React__default.JSX.Element;
    readonly failed: React__default.JSX.Element;
    readonly clear: React__default.JSX.Element;
    readonly comment: React__default.JSX.Element;
    readonly commentadd: React__default.JSX.Element;
    readonly requestchange: React__default.JSX.Element;
    readonly comments: React__default.JSX.Element;
    readonly lock: React__default.JSX.Element;
    readonly unlock: React__default.JSX.Element;
    readonly key: React__default.JSX.Element;
    readonly outbox: React__default.JSX.Element;
    readonly credit: React__default.JSX.Element;
    readonly button: React__default.JSX.Element;
    readonly type: React__default.JSX.Element;
    readonly pointerdefault: React__default.JSX.Element;
    readonly pointerhand: React__default.JSX.Element;
    readonly browser: React__default.JSX.Element;
    readonly tablet: React__default.JSX.Element;
    readonly mobile: React__default.JSX.Element;
    readonly watch: React__default.JSX.Element;
    readonly sidebar: React__default.JSX.Element;
    readonly sidebaralt: React__default.JSX.Element;
    readonly sidebaralttoggle: React__default.JSX.Element;
    readonly sidebartoggle: React__default.JSX.Element;
    readonly bottombar: React__default.JSX.Element;
    readonly bottombartoggle: React__default.JSX.Element;
    readonly cpu: React__default.JSX.Element;
    readonly database: React__default.JSX.Element;
    readonly memory: React__default.JSX.Element;
    readonly structure: React__default.JSX.Element;
    readonly box: React__default.JSX.Element;
    readonly power: React__default.JSX.Element;
    readonly photo: React__default.JSX.Element;
    readonly component: React__default.JSX.Element;
    readonly grid: React__default.JSX.Element;
    readonly outline: React__default.JSX.Element;
    readonly photodrag: React__default.JSX.Element;
    readonly search: React__default.JSX.Element;
    readonly zoom: React__default.JSX.Element;
    readonly zoomout: React__default.JSX.Element;
    readonly zoomreset: React__default.JSX.Element;
    readonly eye: React__default.JSX.Element;
    readonly eyeclose: React__default.JSX.Element;
    readonly lightning: React__default.JSX.Element;
    readonly lightningoff: React__default.JSX.Element;
    readonly contrast: React__default.JSX.Element;
    readonly switchalt: React__default.JSX.Element;
    readonly mirror: React__default.JSX.Element;
    readonly grow: React__default.JSX.Element;
    readonly paintbrush: React__default.JSX.Element;
    readonly ruler: React__default.JSX.Element;
    readonly stop: React__default.JSX.Element;
    readonly camera: React__default.JSX.Element;
    readonly video: React__default.JSX.Element;
    readonly speaker: React__default.JSX.Element;
    readonly play: React__default.JSX.Element;
    readonly playback: React__default.JSX.Element;
    readonly playnext: React__default.JSX.Element;
    readonly rewind: React__default.JSX.Element;
    readonly fastforward: React__default.JSX.Element;
    readonly stopalt: React__default.JSX.Element;
    readonly sidebyside: React__default.JSX.Element;
    readonly stacked: React__default.JSX.Element;
    readonly sun: React__default.JSX.Element;
    readonly moon: React__default.JSX.Element;
    readonly book: React__default.JSX.Element;
    readonly document: React__default.JSX.Element;
    readonly copy: React__default.JSX.Element;
    readonly category: React__default.JSX.Element;
    readonly folder: React__default.JSX.Element;
    readonly print: React__default.JSX.Element;
    readonly graphline: React__default.JSX.Element;
    readonly calendar: React__default.JSX.Element;
    readonly graphbar: React__default.JSX.Element;
    readonly menu: React__default.JSX.Element;
    readonly menualt: React__default.JSX.Element;
    readonly filter: React__default.JSX.Element;
    readonly docchart: React__default.JSX.Element;
    readonly doclist: React__default.JSX.Element;
    readonly markup: React__default.JSX.Element;
    readonly bold: React__default.JSX.Element;
    readonly italic: React__default.JSX.Element;
    readonly paperclip: React__default.JSX.Element;
    readonly listordered: React__default.JSX.Element;
    readonly listunordered: React__default.JSX.Element;
    readonly paragraph: React__default.JSX.Element;
    readonly markdown: React__default.JSX.Element;
    readonly repository: React__default.JSX.Element;
    readonly commit: React__default.JSX.Element;
    readonly branch: React__default.JSX.Element;
    readonly pullrequest: React__default.JSX.Element;
    readonly merge: React__default.JSX.Element;
    readonly apple: React__default.JSX.Element;
    readonly linux: React__default.JSX.Element;
    readonly ubuntu: React__default.JSX.Element;
    readonly windows: React__default.JSX.Element;
    readonly storybook: React__default.JSX.Element;
    readonly azuredevops: React__default.JSX.Element;
    readonly bitbucket: React__default.JSX.Element;
    readonly chrome: React__default.JSX.Element;
    readonly chromatic: React__default.JSX.Element;
    readonly componentdriven: React__default.JSX.Element;
    readonly discord: React__default.JSX.Element;
    readonly facebook: React__default.JSX.Element;
    readonly figma: React__default.JSX.Element;
    readonly gdrive: React__default.JSX.Element;
    readonly github: React__default.JSX.Element;
    readonly gitlab: React__default.JSX.Element;
    readonly google: React__default.JSX.Element;
    readonly graphql: React__default.JSX.Element;
    readonly medium: React__default.JSX.Element;
    readonly redux: React__default.JSX.Element;
    readonly twitter: React__default.JSX.Element;
    readonly youtube: React__default.JSX.Element;
    readonly vscode: React__default.JSX.Element;
};
type IconKey = keyof typeof icons;

interface ItemProps {
    disabled?: boolean;
}
declare const Item: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React__default.ElementType<any>;
} & ItemProps, React__default.DetailedHTMLProps<React__default.AnchorHTMLAttributes<HTMLAnchorElement>, HTMLAnchorElement>, {}>;
type LinkWrapperType = FC<any>;
interface ListItemProps extends Omit<ComponentProps<typeof Item>, 'href' | 'title'> {
    loading?: boolean;
    /**
     * @deprecated This property will be removed in SB 8.0
     * Use `icon` property instead.
     */
    left?: ReactNode;
    title?: ReactNode;
    center?: ReactNode;
    right?: ReactNode;
    icon?: keyof typeof icons | ReactElement;
    active?: boolean;
    disabled?: boolean;
    href?: string;
    LinkWrapper?: LinkWrapperType;
    isIndented?: boolean;
}
declare const ListItem: FC<ListItemProps>;

interface Link extends Omit<ListItemProps, 'onClick'> {
    id: string;
    isGatsby?: boolean;
    onClick?: (event: SyntheticEvent, item: ListItemProps) => void;
}
interface TooltipLinkListProps {
    links: Link[];
    LinkWrapper?: LinkWrapperType;
}
declare const TooltipLinkList: FC<TooltipLinkListProps>;

declare const TabBar: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React__default.ElementType<any>;
}, React__default.DetailedHTMLProps<React__default.HTMLAttributes<HTMLDivElement>, HTMLDivElement>, {}>;
interface TabWrapperProps {
    active: boolean;
    render?: () => JSX.Element;
    children?: ReactNode;
}
declare const TabWrapper: FC<TabWrapperProps>;
interface TabsProps {
    children?: ReactElement<{
        children: FC<Addon_RenderOptions>;
        title: ReactNode | FC;
    }>[];
    id?: string;
    tools?: ReactNode;
    selected?: string;
    actions?: {
        onSelect: (id: string) => void;
    } & Record<string, any>;
    backgroundColor?: string;
    absolute?: boolean;
    bordered?: boolean;
    menuName?: string;
}
declare const Tabs: FC<TabsProps>;
interface TabsStateProps {
    children: TabsProps['children'];
    initial: string;
    absolute: boolean;
    bordered: boolean;
    backgroundColor: string;
    menuName: string;
}
interface TabsStateState {
    selected: string;
}
declare class TabsState extends Component<TabsStateProps, TabsStateState> {
    static defaultProps: TabsStateProps;
    constructor(props: TabsStateProps);
    handlers: {
        onSelect: (id: string) => void;
    };
    render(): React__default.JSX.Element;
}

interface BarButtonProps extends DetailedHTMLProps<ButtonHTMLAttributes<HTMLButtonElement>, HTMLButtonElement> {
    href?: void;
    target?: void;
}
interface BarLinkProps extends DetailedHTMLProps<AnchorHTMLAttributes<HTMLAnchorElement>, HTMLAnchorElement> {
    disabled?: void;
    href: string;
}
interface TabButtonProps {
    active?: boolean;
    textColor?: string;
}
declare const TabButton: _storybook_theming.StyledComponent<Pick<BarButtonProps | BarLinkProps, "href" | "target" | "type" | "slot" | "style" | "title" | "className" | "color" | "content" | "translate" | "hidden" | "children" | "defaultChecked" | "defaultValue" | "suppressContentEditableWarning" | "suppressHydrationWarning" | "accessKey" | "autoFocus" | "contentEditable" | "contextMenu" | "dir" | "draggable" | "id" | "lang" | "nonce" | "placeholder" | "spellCheck" | "tabIndex" | "radioGroup" | "role" | "about" | "datatype" | "inlist" | "prefix" | "property" | "rel" | "resource" | "rev" | "typeof" | "vocab" | "autoCapitalize" | "autoCorrect" | "autoSave" | "itemProp" | "itemScope" | "itemType" | "itemID" | "itemRef" | "results" | "security" | "unselectable" | "inputMode" | "is" | "aria-activedescendant" | "aria-atomic" | "aria-autocomplete" | "aria-busy" | "aria-checked" | "aria-colcount" | "aria-colindex" | "aria-colspan" | "aria-controls" | "aria-current" | "aria-describedby" | "aria-details" | "aria-disabled" | "aria-dropeffect" | "aria-errormessage" | "aria-expanded" | "aria-flowto" | "aria-grabbed" | "aria-haspopup" | "aria-hidden" | "aria-invalid" | "aria-keyshortcuts" | "aria-label" | "aria-labelledby" | "aria-level" | "aria-live" | "aria-modal" | "aria-multiline" | "aria-multiselectable" | "aria-orientation" | "aria-owns" | "aria-placeholder" | "aria-posinset" | "aria-pressed" | "aria-readonly" | "aria-relevant" | "aria-required" | "aria-roledescription" | "aria-rowcount" | "aria-rowindex" | "aria-rowspan" | "aria-selected" | "aria-setsize" | "aria-sort" | "aria-valuemax" | "aria-valuemin" | "aria-valuenow" | "aria-valuetext" | "dangerouslySetInnerHTML" | "onCopy" | "onCopyCapture" | "onCut" | "onCutCapture" | "onPaste" | "onPasteCapture" | "onCompositionEnd" | "onCompositionEndCapture" | "onCompositionStart" | "onCompositionStartCapture" | "onCompositionUpdate" | "onCompositionUpdateCapture" | "onFocus" | "onFocusCapture" | "onBlur" | "onBlurCapture" | "onChange" | "onChangeCapture" | "onBeforeInput" | "onBeforeInputCapture" | "onInput" | "onInputCapture" | "onReset" | "onResetCapture" | "onSubmit" | "onSubmitCapture" | "onInvalid" | "onInvalidCapture" | "onLoad" | "onLoadCapture" | "onError" | "onErrorCapture" | "onKeyDown" | "onKeyDownCapture" | "onKeyPress" | "onKeyPressCapture" | "onKeyUp" | "onKeyUpCapture" | "onAbort" | "onAbortCapture" | "onCanPlay" | "onCanPlayCapture" | "onCanPlayThrough" | "onCanPlayThroughCapture" | "onDurationChange" | "onDurationChangeCapture" | "onEmptied" | "onEmptiedCapture" | "onEncrypted" | "onEncryptedCapture" | "onEnded" | "onEndedCapture" | "onLoadedData" | "onLoadedDataCapture" | "onLoadedMetadata" | "onLoadedMetadataCapture" | "onLoadStart" | "onLoadStartCapture" | "onPause" | "onPauseCapture" | "onPlay" | "onPlayCapture" | "onPlaying" | "onPlayingCapture" | "onProgress" | "onProgressCapture" | "onRateChange" | "onRateChangeCapture" | "onSeeked" | "onSeekedCapture" | "onSeeking" | "onSeekingCapture" | "onStalled" | "onStalledCapture" | "onSuspend" | "onSuspendCapture" | "onTimeUpdate" | "onTimeUpdateCapture" | "onVolumeChange" | "onVolumeChangeCapture" | "onWaiting" | "onWaitingCapture" | "onAuxClick" | "onAuxClickCapture" | "onClick" | "onClickCapture" | "onContextMenu" | "onContextMenuCapture" | "onDoubleClick" | "onDoubleClickCapture" | "onDrag" | "onDragCapture" | "onDragEnd" | "onDragEndCapture" | "onDragEnter" | "onDragEnterCapture" | "onDragExit" | "onDragExitCapture" | "onDragLeave" | "onDragLeaveCapture" | "onDragOver" | "onDragOverCapture" | "onDragStart" | "onDragStartCapture" | "onDrop" | "onDropCapture" | "onMouseDown" | "onMouseDownCapture" | "onMouseEnter" | "onMouseLeave" | "onMouseMove" | "onMouseMoveCapture" | "onMouseOut" | "onMouseOutCapture" | "onMouseOver" | "onMouseOverCapture" | "onMouseUp" | "onMouseUpCapture" | "onSelect" | "onSelectCapture" | "onTouchCancel" | "onTouchCancelCapture" | "onTouchEnd" | "onTouchEndCapture" | "onTouchMove" | "onTouchMoveCapture" | "onTouchStart" | "onTouchStartCapture" | "onPointerDown" | "onPointerDownCapture" | "onPointerMove" | "onPointerMoveCapture" | "onPointerUp" | "onPointerUpCapture" | "onPointerCancel" | "onPointerCancelCapture" | "onPointerEnter" | "onPointerEnterCapture" | "onPointerLeave" | "onPointerLeaveCapture" | "onPointerOver" | "onPointerOverCapture" | "onPointerOut" | "onPointerOutCapture" | "onGotPointerCapture" | "onGotPointerCaptureCapture" | "onLostPointerCapture" | "onLostPointerCaptureCapture" | "onScroll" | "onScrollCapture" | "onWheel" | "onWheelCapture" | "onAnimationStart" | "onAnimationStartCapture" | "onAnimationEnd" | "onAnimationEndCapture" | "onAnimationIteration" | "onAnimationIterationCapture" | "onTransitionEnd" | "onTransitionEndCapture" | "disabled" | "key"> & React__default.RefAttributes<HTMLAnchorElement | HTMLButtonElement> & {
    theme?: _storybook_theming.Theme;
} & TabButtonProps, {}, {}>;
interface IconButtonProps {
    active?: boolean;
    disabled?: boolean;
}
declare const IconButton: _storybook_theming.StyledComponent<Pick<BarButtonProps | BarLinkProps, "href" | "target" | "type" | "slot" | "style" | "title" | "className" | "color" | "content" | "translate" | "hidden" | "children" | "defaultChecked" | "defaultValue" | "suppressContentEditableWarning" | "suppressHydrationWarning" | "accessKey" | "autoFocus" | "contentEditable" | "contextMenu" | "dir" | "draggable" | "id" | "lang" | "nonce" | "placeholder" | "spellCheck" | "tabIndex" | "radioGroup" | "role" | "about" | "datatype" | "inlist" | "prefix" | "property" | "rel" | "resource" | "rev" | "typeof" | "vocab" | "autoCapitalize" | "autoCorrect" | "autoSave" | "itemProp" | "itemScope" | "itemType" | "itemID" | "itemRef" | "results" | "security" | "unselectable" | "inputMode" | "is" | "aria-activedescendant" | "aria-atomic" | "aria-autocomplete" | "aria-busy" | "aria-checked" | "aria-colcount" | "aria-colindex" | "aria-colspan" | "aria-controls" | "aria-current" | "aria-describedby" | "aria-details" | "aria-disabled" | "aria-dropeffect" | "aria-errormessage" | "aria-expanded" | "aria-flowto" | "aria-grabbed" | "aria-haspopup" | "aria-hidden" | "aria-invalid" | "aria-keyshortcuts" | "aria-label" | "aria-labelledby" | "aria-level" | "aria-live" | "aria-modal" | "aria-multiline" | "aria-multiselectable" | "aria-orientation" | "aria-owns" | "aria-placeholder" | "aria-posinset" | "aria-pressed" | "aria-readonly" | "aria-relevant" | "aria-required" | "aria-roledescription" | "aria-rowcount" | "aria-rowindex" | "aria-rowspan" | "aria-selected" | "aria-setsize" | "aria-sort" | "aria-valuemax" | "aria-valuemin" | "aria-valuenow" | "aria-valuetext" | "dangerouslySetInnerHTML" | "onCopy" | "onCopyCapture" | "onCut" | "onCutCapture" | "onPaste" | "onPasteCapture" | "onCompositionEnd" | "onCompositionEndCapture" | "onCompositionStart" | "onCompositionStartCapture" | "onCompositionUpdate" | "onCompositionUpdateCapture" | "onFocus" | "onFocusCapture" | "onBlur" | "onBlurCapture" | "onChange" | "onChangeCapture" | "onBeforeInput" | "onBeforeInputCapture" | "onInput" | "onInputCapture" | "onReset" | "onResetCapture" | "onSubmit" | "onSubmitCapture" | "onInvalid" | "onInvalidCapture" | "onLoad" | "onLoadCapture" | "onError" | "onErrorCapture" | "onKeyDown" | "onKeyDownCapture" | "onKeyPress" | "onKeyPressCapture" | "onKeyUp" | "onKeyUpCapture" | "onAbort" | "onAbortCapture" | "onCanPlay" | "onCanPlayCapture" | "onCanPlayThrough" | "onCanPlayThroughCapture" | "onDurationChange" | "onDurationChangeCapture" | "onEmptied" | "onEmptiedCapture" | "onEncrypted" | "onEncryptedCapture" | "onEnded" | "onEndedCapture" | "onLoadedData" | "onLoadedDataCapture" | "onLoadedMetadata" | "onLoadedMetadataCapture" | "onLoadStart" | "onLoadStartCapture" | "onPause" | "onPauseCapture" | "onPlay" | "onPlayCapture" | "onPlaying" | "onPlayingCapture" | "onProgress" | "onProgressCapture" | "onRateChange" | "onRateChangeCapture" | "onSeeked" | "onSeekedCapture" | "onSeeking" | "onSeekingCapture" | "onStalled" | "onStalledCapture" | "onSuspend" | "onSuspendCapture" | "onTimeUpdate" | "onTimeUpdateCapture" | "onVolumeChange" | "onVolumeChangeCapture" | "onWaiting" | "onWaitingCapture" | "onAuxClick" | "onAuxClickCapture" | "onClick" | "onClickCapture" | "onContextMenu" | "onContextMenuCapture" | "onDoubleClick" | "onDoubleClickCapture" | "onDrag" | "onDragCapture" | "onDragEnd" | "onDragEndCapture" | "onDragEnter" | "onDragEnterCapture" | "onDragExit" | "onDragExitCapture" | "onDragLeave" | "onDragLeaveCapture" | "onDragOver" | "onDragOverCapture" | "onDragStart" | "onDragStartCapture" | "onDrop" | "onDropCapture" | "onMouseDown" | "onMouseDownCapture" | "onMouseEnter" | "onMouseLeave" | "onMouseMove" | "onMouseMoveCapture" | "onMouseOut" | "onMouseOutCapture" | "onMouseOver" | "onMouseOverCapture" | "onMouseUp" | "onMouseUpCapture" | "onSelect" | "onSelectCapture" | "onTouchCancel" | "onTouchCancelCapture" | "onTouchEnd" | "onTouchEndCapture" | "onTouchMove" | "onTouchMoveCapture" | "onTouchStart" | "onTouchStartCapture" | "onPointerDown" | "onPointerDownCapture" | "onPointerMove" | "onPointerMoveCapture" | "onPointerUp" | "onPointerUpCapture" | "onPointerCancel" | "onPointerCancelCapture" | "onPointerEnter" | "onPointerEnterCapture" | "onPointerLeave" | "onPointerLeaveCapture" | "onPointerOver" | "onPointerOverCapture" | "onPointerOut" | "onPointerOutCapture" | "onGotPointerCapture" | "onGotPointerCaptureCapture" | "onLostPointerCapture" | "onLostPointerCaptureCapture" | "onScroll" | "onScrollCapture" | "onWheel" | "onWheelCapture" | "onAnimationStart" | "onAnimationStartCapture" | "onAnimationEnd" | "onAnimationEndCapture" | "onAnimationIteration" | "onAnimationIterationCapture" | "onTransitionEnd" | "onTransitionEndCapture" | "disabled" | "key"> & React__default.RefAttributes<HTMLAnchorElement | HTMLButtonElement> & {
    theme?: _storybook_theming.Theme;
} & IconButtonProps, {}, {}>;
declare const IconButtonSkeleton: () => React__default.JSX.Element;

declare const Separator: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React__default.ElementType<any>;
} & SeparatorProps, React__default.DetailedHTMLProps<React__default.HTMLAttributes<HTMLSpanElement>, HTMLSpanElement>, {}>;
declare const interleaveSeparators: (list: any[]) => any;
interface SeparatorProps {
    force?: boolean;
}

declare const Bar: _storybook_theming.StyledComponent<ScrollAreaProps & {
    children?: React__default.ReactNode;
} & {
    scrollable?: boolean;
} & {
    theme?: _storybook_theming.Theme;
} & {
    border?: boolean;
    scrollable?: boolean;
}, {}, {}>;
interface FlexBarProps extends ComponentProps<typeof Bar> {
    border?: boolean;
    backgroundColor?: string;
}
declare const FlexBar: FC<FlexBarProps>;

interface AddonPanelProps {
    active: boolean;
    children: ReactNode;
}
declare const AddonPanel: ({ active, children }: AddonPanelProps) => React__default.JSX.Element;

declare const Svg: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React__default.ElementType<any>;
}, React__default.SVGProps<SVGSVGElement>, {}>;
interface IconsProps extends ComponentProps<typeof Svg> {
    icon: IconType;
    useSymbol?: boolean;
}
declare const Icons: FunctionComponent<IconsProps>;
type IconType = keyof typeof icons;
interface SymbolsProps extends ComponentProps<typeof Svg> {
    icons?: IconKey[];
}
declare const Symbols: React__default.NamedExoticComponent<SymbolsProps>;

declare const StorybookLogo: FC<{
    alt: string;
}>;

declare const StorybookIcon: FC;

declare const ProgressWrapper: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React__default.ElementType<any>;
}, React__default.DetailedHTMLProps<React__default.HTMLAttributes<HTMLDivElement>, HTMLDivElement>, {}>;
interface Progress {
    value: number;
    message: string;
    modules?: {
        complete: number;
        total: number;
    };
}
interface LoaderProps {
    progress?: Progress;
    error?: Error;
    size?: number;
}
declare const Loader: FC<LoaderProps & ComponentProps<typeof ProgressWrapper>>;

declare const getStoryHref: (baseUrl: string, storyId: string, additionalParams?: Record<string, string>) => string;

declare const nameSpaceClassNames: ({ ...props }: {
    [x: string]: any;
}, key: string) => {
    [x: string]: any;
};

/**
 * This is a "local" reset to style subtrees with Storybook styles
 *
 * We can't style individual elements (e.g. h1, h2, etc.) in here
 * because the CSS specificity is too high, so those styles can too
 * easily override child elements that are not expecting it.
 */
declare const ResetWrapper: _storybook_theming.StyledComponent<{
    theme?: _storybook_theming.Theme;
    as?: React$1.ElementType<any>;
}, React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLDivElement>, HTMLDivElement>, {}>;

declare const codeCommon: ({ theme }: {
    theme: Theme;
}) => CSSObject;
declare const withReset: ({ theme }: {
    theme: Theme;
}) => CSSObject;

interface ClipboardCodeProps {
    code: string;
}
declare const ClipboardCode: ({ code, ...props }: ClipboardCodeProps) => React__default.JSX.Element;

declare const components: {
    h1: (props: React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLHeadingElement>, HTMLHeadingElement>) => React$1.JSX.Element;
    h2: (props: React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLHeadingElement>, HTMLHeadingElement>) => React$1.JSX.Element;
    h3: (props: React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLHeadingElement>, HTMLHeadingElement>) => React$1.JSX.Element;
    h4: (props: React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLHeadingElement>, HTMLHeadingElement>) => React$1.JSX.Element;
    h5: (props: React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLHeadingElement>, HTMLHeadingElement>) => React$1.JSX.Element;
    h6: (props: React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLHeadingElement>, HTMLHeadingElement>) => React$1.JSX.Element;
    pre: (props: React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLPreElement>, HTMLPreElement>) => React$1.JSX.Element;
    a: (props: React$1.DetailedHTMLProps<React$1.AnchorHTMLAttributes<HTMLAnchorElement>, HTMLAnchorElement>) => React$1.JSX.Element;
    hr: (props: React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLHRElement>, HTMLHRElement>) => React$1.JSX.Element;
    dl: (props: React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLDListElement>, HTMLDListElement>) => React$1.JSX.Element;
    blockquote: (props: React$1.DetailedHTMLProps<React$1.BlockquoteHTMLAttributes<HTMLElement>, HTMLElement>) => React$1.JSX.Element;
    table: (props: React$1.DetailedHTMLProps<React$1.TableHTMLAttributes<HTMLTableElement>, HTMLTableElement>) => React$1.JSX.Element;
    img: (props: React$1.DetailedHTMLProps<React$1.ImgHTMLAttributes<HTMLImageElement>, HTMLImageElement>) => React$1.JSX.Element;
    div: (props: React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLDivElement>, HTMLDivElement>) => React$1.JSX.Element;
    span: (props: React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLSpanElement>, HTMLSpanElement>) => React$1.JSX.Element;
    li: (props: React$1.DetailedHTMLProps<React$1.LiHTMLAttributes<HTMLLIElement>, HTMLLIElement>) => React$1.JSX.Element;
    ul: (props: React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLUListElement>, HTMLUListElement>) => React$1.JSX.Element;
    ol: (props: React$1.DetailedHTMLProps<React$1.OlHTMLAttributes<HTMLOListElement>, HTMLOListElement>) => React$1.JSX.Element;
    p: (props: React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLParagraphElement>, HTMLParagraphElement>) => React$1.JSX.Element;
    code: (props: React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLElement>, HTMLElement>) => React$1.JSX.Element;
    tt: (props: React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLTitleElement>, HTMLTitleElement>) => React$1.JSX.Element;
    resetwrapper: (props: React$1.DetailedHTMLProps<React$1.HTMLAttributes<HTMLDivElement>, HTMLDivElement>) => React$1.JSX.Element;
};
declare const resetComponents: Record<string, ElementType>;

export { A, ActionBar, ActionItem, AddonPanel, Badge, Bar, Blockquote, Button, ClipboardCode, Code, DL, Div, DocumentWrapper, ErrorFormatter, FlexBar, Form, H1, H2, H3, H4, H5, H6, HR, IconButton, IconButtonSkeleton, Icons, IconsProps, Img, LI, Link$1 as Link, ListItem, Loader, OL, P, Placeholder, Pre, ResetWrapper, ScrollArea, Separator, Spaced, Span, StorybookIcon, StorybookLogo, Symbols, SyntaxHighlighter, SyntaxHighlighterFormatTypes, SyntaxHighlighterProps, SyntaxHighlighterRendererProps, TT, TabBar, TabButton, TabWrapper, Table, Tabs, TabsState, TooltipLinkList, Link as TooltipLinkListLink, TooltipMessage, TooltipNote, UL, WithTooltip, WithTooltipPure, Zoom, codeCommon, components, createCopyToClipboardFunction, getStoryHref, icons, interleaveSeparators, nameSpaceClassNames, resetComponents, withReset };
