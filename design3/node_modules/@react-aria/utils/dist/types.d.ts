import React, { MutableRefObject, Ref, DOMAttributes, JSX, ReactNode, HTMLAttributes, EffectCallback, Dispatch, RefObject as _RefObject1 } from "react";
import { AriaLabelingProps, DOMProps, LinkDOMProps, FocusableElement, Orientation, Href, RouterOptions, RefObject, AsyncLoadable, Collection } from "@react-types/shared";
export const useLayoutEffect: typeof React.useLayoutEffect;
/**
 * If a default is not provided, generate an id.
 * @param defaultId - Default component id.
 */
export function useId(defaultId?: string): string;
/**
 * Merges two ids.
 * Different ids will trigger a side-effect and re-render components hooked up with `useId`.
 */
export function mergeIds(idA: string, idB: string): string;
/**
 * Used to generate an id, and after render, check if that id is rendered so we know
 * if we can use it in places such as labelledby.
 * @param depArray - When to recalculate if the id is in the DOM.
 */
export function useSlotId(depArray?: ReadonlyArray<any>): string;
/**
 * Calls all functions in the order they were chained with the same arguments.
 */
export function chain(...callbacks: any[]): (...args: any[]) => void;
export const getOwnerDocument: (el: Element | null | undefined) => Document;
export const getOwnerWindow: (el: (Window & typeof global) | Element | null | undefined) => Window & typeof global;
/**
 * Type guard that checks if a node is a ShadowRoot. Uses nodeType and host property checks to
 * distinguish ShadowRoot from other DocumentFragments.
 */
export function isShadowRoot(node: Node | null): node is ShadowRoot;
/**
 * ShadowDOM safe version of Node.contains.
 */
export function nodeContains(node: Node | null | undefined, otherNode: Node | null | undefined): boolean;
/**
 * ShadowDOM safe version of document.activeElement.
 */
export const getActiveElement: (doc?: Document) => Element | null;
/**
 * ShadowDOM safe version of event.target.
 */
export function getEventTarget<T extends Event>(event: T): Element;
export class ShadowTreeWalker implements TreeWalker {
    readonly filter: NodeFilter | null;
    readonly root: Node;
    readonly whatToShow: number;
    constructor(doc: Document, root: Node, whatToShow?: number, filter?: NodeFilter | null);
    get currentNode(): Node;
    set currentNode(node: Node);
    get doc(): Document;
    firstChild(): Node | null;
    lastChild(): Node | null;
    nextNode(): Node | null;
    previousNode(): Node | null;
    /**
     * @deprecated
     */
    nextSibling(): Node | null;
    /**
     * @deprecated
     */
    previousSibling(): Node | null;
    /**
     * @deprecated
     */
    parentNode(): Node | null;
}
/**
 * ShadowDOM safe version of document.createTreeWalker.
 */
export function createShadowTreeWalker(doc: Document, root: Node, whatToShow?: number, filter?: NodeFilter | null): TreeWalker;
interface Props {
    [key: string]: any;
}
type PropsArg = Props | null | undefined;
type TupleTypes<T> = {
    [P in keyof T]: T[P];
} extends {
    [key: number]: infer V;
} ? NullToObject<V> : never;
type NullToObject<T> = T extends (null | undefined) ? {} : T;
type UnionToIntersection<U> = (U extends any ? (k: U) => void : never) extends ((k: infer I) => void) ? I : never;
/**
 * Merges multiple props objects together. Event handlers are chained,
 * classNames are combined, and ids are deduplicated - different ids
 * will trigger a side-effect and re-render components hooked up with `useId`.
 * For all other props, the last prop object overrides all previous ones.
 * @param args - Multiple sets of props to merge together.
 */
export function mergeProps<T extends PropsArg[]>(...args: T): UnionToIntersection<TupleTypes<T>>;
/**
 * Merges multiple refs into one. Works with either callback or object refs.
 */
export function mergeRefs<T>(...refs: Array<Ref<T> | MutableRefObject<T> | null | undefined>): Ref<T>;
interface Options {
    /**
     * If labelling associated aria properties should be included in the filter.
     */
    labelable?: boolean;
    /** Whether the element is a link and should include DOM props for <a> elements. */
    isLink?: boolean;
    /**
     * A Set of other property names that should be included in the filter.
     */
    propNames?: Set<string>;
}
/**
 * Filters out all props that aren't valid DOM props or defined via override prop obj.
 * @param props - The component props to be filtered.
 * @param opts - Props to override.
 */
export function filterDOMProps(props: DOMProps & AriaLabelingProps & LinkDOMProps, opts?: Options): DOMProps & AriaLabelingProps;
export function focusWithoutScrolling(element: FocusableElement): void;
export function getOffset(element: HTMLElement, reverse?: boolean, orientation?: Orientation): number;
export const isMac: () => boolean;
export const isIPhone: () => boolean;
export const isIPad: () => boolean;
export const isIOS: () => boolean;
export const isAppleDevice: () => boolean;
export const isWebKit: () => boolean;
export const isChrome: () => boolean;
export const isAndroid: () => boolean;
export const isFirefox: () => boolean;
interface Router {
    isNative: boolean;
    open: (target: Element, modifiers: Modifiers, href: Href, routerOptions: RouterOptions | undefined) => void;
    useHref: (href: Href) => string;
}
interface RouterProviderProps {
    navigate: (path: Href, routerOptions: RouterOptions | undefined) => void;
    useHref?: (href: Href) => string;
    children: ReactNode;
}
/**
 * A RouterProvider accepts a `navigate` function from a framework or client side router,
 * and provides it to all nested React Aria links to enable client side navigation.
 */
export function RouterProvider(props: RouterProviderProps): JSX.Element;
export function useRouter(): Router;
interface Modifiers {
    metaKey?: boolean;
    ctrlKey?: boolean;
    altKey?: boolean;
    shiftKey?: boolean;
}
export function shouldClientNavigate(link: HTMLAnchorElement, modifiers: Modifiers): boolean;
export function openLink(target: HTMLAnchorElement, modifiers: Modifiers, setOpening?: boolean): void;
export function useSyntheticLinkProps(props: LinkDOMProps): DOMAttributes<HTMLElement>;
/** @deprecated - For backward compatibility. */
export function getSyntheticLinkProps(props: LinkDOMProps): DOMAttributes<HTMLElement>;
export function useLinkProps(props?: LinkDOMProps): LinkDOMProps;
export function runAfterTransition(fn: () => void): void;
interface UseDrag1DProps {
    containerRef: MutableRefObject<HTMLElement>;
    reverse?: boolean;
    orientation?: Orientation;
    onHover?: (hovered: boolean) => void;
    onDrag?: (dragging: boolean) => void;
    onPositionChange?: (position: number) => void;
    onIncrement?: () => void;
    onDecrement?: () => void;
    onIncrementToMax?: () => void;
    onDecrementToMin?: () => void;
    onCollapseToggle?: () => void;
}
export function useDrag1D(props: UseDrag1DProps): HTMLAttributes<HTMLElement>;
interface GlobalListeners {
    addGlobalListener<K extends keyof WindowEventMap>(el: Window, type: K, listener: (this: Document, ev: WindowEventMap[K]) => any, options?: boolean | AddEventListenerOptions): void;
    addGlobalListener<K extends keyof DocumentEventMap>(el: EventTarget, type: K, listener: (this: Document, ev: DocumentEventMap[K]) => any, options?: boolean | AddEventListenerOptions): void;
    addGlobalListener(el: EventTarget, type: string, listener: EventListenerOrEventListenerObject, options?: boolean | AddEventListenerOptions): void;
    removeGlobalListener<K extends keyof DocumentEventMap>(el: EventTarget, type: K, listener: (this: Document, ev: DocumentEventMap[K]) => any, options?: boolean | EventListenerOptions): void;
    removeGlobalListener(el: EventTarget, type: string, listener: EventListenerOrEventListenerObject, options?: boolean | EventListenerOptions): void;
    removeAllGlobalListeners(): void;
}
export function useGlobalListeners(): GlobalListeners;
/**
 * Merges aria-label and aria-labelledby into aria-labelledby when both exist.
 * @param props - Aria label props.
 * @param defaultLabel - Default value for aria-label when not present.
 */
export function useLabels(props: DOMProps & AriaLabelingProps, defaultLabel?: string): DOMProps & AriaLabelingProps;
/**
 * Offers an object ref for a given callback ref or an object ref. Especially
 * helfpul when passing forwarded refs (created using `React.forwardRef`) to
 * React Aria hooks.
 *
 * @param ref The original ref intended to be used.
 * @returns An object ref that updates the given ref.
 * @see https://react.dev/reference/react/forwardRef
 */
export function useObjectRef<T>(ref?: ((instance: T | null) => (() => void) | void) | MutableRefObject<T | null> | null): MutableRefObject<T | null>;
export function useUpdateEffect(effect: EffectCallback, dependencies: any[]): void;
export function useUpdateLayoutEffect(effect: EffectCallback, dependencies: any[]): void;
type useResizeObserverOptionsType<T> = {
    ref: RefObject<T | undefined | null> | undefined;
    box?: ResizeObserverBoxOptions;
    onResize: () => void;
};
export function useResizeObserver<T extends Element>(options: useResizeObserverOptionsType<T>): void;
interface ContextValue<T> {
    ref?: MutableRefObject<T | null>;
}
export function useSyncRef<T>(context?: ContextValue<T> | null, ref?: RefObject<T | null>): void;
export function isScrollable(node: Element | null, checkForOverflow?: boolean): boolean;
export function getScrollParent(node: Element, checkForOverflow?: boolean): Element;
export function getScrollParents(node: Element, checkForOverflow?: boolean): Element[];
interface ViewportSize {
    width: number;
    height: number;
}
export function useViewportSize(): ViewportSize;
export function useDescription(description?: string): AriaLabelingProps;
export function useEffectEvent<T extends Function>(fn?: T): T;
export function useEvent<K extends keyof GlobalEventHandlersEventMap>(ref: RefObject<EventTarget | null>, event: K | (string & {}), handler?: (this: Document, ev: GlobalEventHandlersEventMap[K]) => any, options?: boolean | AddEventListenerOptions): void;
type SetValueAction<S> = (prev: S) => Generator<any, void, unknown>;
export function useValueEffect<S>(defaultValue: S | (() => S)): [S, Dispatch<SetValueAction<S>>];
interface ScrollIntoViewportOpts {
    /** The optional containing element of the target to be centered in the viewport. */
    containingElement?: Element | null;
}
/**
 * Scrolls `scrollView` so that `element` is visible.
 * Similar to `element.scrollIntoView({block: 'nearest'})` (not supported in Edge),
 * but doesn't affect parents above `scrollView`.
 */
export function scrollIntoView(scrollView: HTMLElement, element: HTMLElement): void;
/**
 * Scrolls the `targetElement` so it is visible in the viewport. Accepts an optional `opts.containingElement`
 * that will be centered in the viewport prior to scrolling the targetElement into view. If scrolling is prevented on
 * the body (e.g. targetElement is in a popover), this will only scroll the scroll parents of the targetElement up to but not including the body itself.
 */
export function scrollIntoViewport(targetElement: Element | null, opts?: ScrollIntoViewportOpts): void;
export function isVirtualClick(event: MouseEvent | PointerEvent): boolean;
export function isVirtualPointerEvent(event: PointerEvent): boolean;
export function useDeepMemo<T>(value: T, isEqual: (a: T, b: T) => boolean): T;
export function useFormReset<T>(ref: RefObject<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement | null> | undefined, initialValue: T, onReset: (value: T) => void): void;
interface LoadMoreProps {
    /** Whether data is currently being loaded. */
    isLoading?: boolean;
    /** Handler that is called when more items should be loaded, e.g. while scrolling near the bottom.  */
    onLoadMore?: () => void;
    /**
     * The amount of offset from the bottom of your scrollable region that should trigger load more.
     * Uses a percentage value relative to the scroll body's client height. Load more is then triggered
     * when your current scroll position's distance from the bottom of the currently loaded list of items is less than
     * or equal to the provided value. (e.g. 1 = 100% of the scroll region's height).
     * @default 1
     */
    scrollOffset?: number;
    /** The data currently loaded. */
    items?: any;
}
export function useLoadMore(props: LoadMoreProps, ref: _RefObject1<HTMLElement | null>): void;
export interface LoadMoreSentinelProps extends Omit<AsyncLoadable, 'isLoading'> {
    collection: Collection<any>;
    /**
     * The amount of offset from the bottom of your scrollable region that should trigger load more.
     * Uses a percentage value relative to the scroll body's client height. Load more is then triggered
     * when your current scroll position's distance from the bottom of the currently loaded list of items is less than
     * or equal to the provided value. (e.g. 1 = 100% of the scroll region's height).
     * @default 1
     */
    scrollOffset?: number;
}
export function UNSTABLE_useLoadMoreSentinel(props: LoadMoreSentinelProps, ref: _RefObject1<HTMLElement | null>): void;
export function inertValue(value?: boolean): string | boolean | undefined;
export const CLEAR_FOCUS_EVENT = "react-aria-clear-focus";
export const FOCUS_EVENT = "react-aria-focus";
interface Event {
    altKey: boolean;
    ctrlKey: boolean;
    metaKey: boolean;
}
export function isCtrlKeyPressed(e: Event): boolean;
export function useEnterAnimation(ref: _RefObject1<HTMLElement | null>, isReady?: boolean): boolean;
export function useExitAnimation(ref: _RefObject1<HTMLElement | null>, isOpen: boolean): boolean;
export function isFocusable(element: Element): boolean;
export function isTabbable(element: Element): boolean;
export { clamp, snapValueToStep } from '@react-stately/utils';

//# sourceMappingURL=types.d.ts.map
