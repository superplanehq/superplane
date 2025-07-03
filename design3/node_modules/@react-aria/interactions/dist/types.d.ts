import { DOMAttributes, PressEvents, RefObject, FocusableElement, FocusEvents, KeyboardEvents, FocusableDOMProps, FocusableProps, HoverEvents, MoveEvents, ScrollEvents, LongPressEvent } from "@react-types/shared";
import React, { MutableRefObject, ReactElement, ReactNode, JSX, FocusEvent } from "react";
export interface PressProps extends PressEvents {
    /** Whether the target is in a controlled press state (e.g. an overlay it triggers is open). */
    isPressed?: boolean;
    /** Whether the press events should be disabled. */
    isDisabled?: boolean;
    /** Whether the target should not receive focus on press. */
    preventFocusOnPress?: boolean;
    /**
     * Whether press events should be canceled when the pointer leaves the target while pressed.
     * By default, this is `false`, which means if the pointer returns back over the target while
     * still pressed, onPressStart will be fired again. If set to `true`, the press is canceled
     * when the pointer leaves the target and onPressStart will not be fired if the pointer returns.
     */
    shouldCancelOnPointerExit?: boolean;
    /** Whether text selection should be enabled on the pressable element. */
    allowTextSelectionOnPress?: boolean;
}
export interface PressHookProps extends PressProps {
    /** A ref to the target element. */
    ref?: RefObject<Element | null>;
}
export interface PressResult {
    /** Whether the target is currently pressed. */
    isPressed: boolean;
    /** Props to spread on the target element. */
    pressProps: DOMAttributes;
}
/**
 * Handles press interactions across mouse, touch, keyboard, and screen readers.
 * It normalizes behavior across browsers and platforms, and handles many nuances
 * of dealing with pointer and keyboard events.
 */
export function usePress(props: PressHookProps): PressResult;
export interface FocusProps<Target = FocusableElement> extends FocusEvents<Target> {
    /** Whether the focus events should be disabled. */
    isDisabled?: boolean;
}
export interface FocusResult<Target = FocusableElement> {
    /** Props to spread onto the target element. */
    focusProps: DOMAttributes<Target>;
}
/**
 * Handles focus events for the immediate target.
 * Focus events on child elements will be ignored.
 */
export function useFocus<Target extends FocusableElement = FocusableElement>(props: FocusProps<Target>): FocusResult<Target>;
export interface KeyboardProps extends KeyboardEvents {
    /** Whether the keyboard events should be disabled. */
    isDisabled?: boolean;
}
export interface KeyboardResult {
    /** Props to spread onto the target element. */
    keyboardProps: DOMAttributes;
}
/**
 * Handles keyboard interactions for a focusable element.
 */
export function useKeyboard(props: KeyboardProps): KeyboardResult;
export interface FocusableOptions<T = FocusableElement> extends FocusableProps<T>, FocusableDOMProps {
    /** Whether focus should be disabled. */
    isDisabled?: boolean;
}
export interface FocusableProviderProps extends DOMAttributes {
    /** The child element to provide DOM props to. */
    children?: ReactNode;
}
interface FocusableContextValue extends FocusableProviderProps {
    ref?: MutableRefObject<FocusableElement | null>;
}
/** @private */
export let FocusableContext: React.Context<FocusableContextValue | null>;
/**
 * Provides DOM props to the nearest focusable child.
 */
export const FocusableProvider: React.ForwardRefExoticComponent<FocusableProviderProps & React.RefAttributes<FocusableElement>>;
export interface FocusableAria {
    /** Props for the focusable element. */
    focusableProps: DOMAttributes;
}
/**
 * Used to make an element focusable and capable of auto focus.
 */
export function useFocusable<T extends FocusableElement = FocusableElement>(props: FocusableOptions<T>, domRef: RefObject<FocusableElement | null>): FocusableAria;
interface FocusableComponentProps extends FocusableOptions {
    children: ReactElement<DOMAttributes, string>;
}
export const Focusable: React.ForwardRefExoticComponent<FocusableComponentProps & React.RefAttributes<FocusableElement>>;
interface PressableProps extends PressProps {
    children: ReactElement<DOMAttributes, string>;
}
export const Pressable: React.ForwardRefExoticComponent<PressableProps & React.RefAttributes<FocusableElement>>;
interface PressResponderProps extends PressProps {
    children: ReactNode;
}
export const PressResponder: React.ForwardRefExoticComponent<PressResponderProps & React.RefAttributes<FocusableElement>>;
export function ClearPressResponder({ children }: {
    children: ReactNode;
}): JSX.Element;
export type Modality = 'keyboard' | 'pointer' | 'virtual';
export type FocusVisibleHandler = (isFocusVisible: boolean) => void;
export interface FocusVisibleProps {
    /** Whether the element is a text input. */
    isTextInput?: boolean;
    /** Whether the element will be auto focused. */
    autoFocus?: boolean;
}
export interface FocusVisibleResult {
    /** Whether keyboard focus is visible globally. */
    isFocusVisible: boolean;
}
/**
 * EXPERIMENTAL
 * Adds a window (i.e. iframe) to the list of windows that are being tracked for focus visible.
 *
 * Sometimes apps render portions of their tree into an iframe. In this case, we cannot accurately track if the focus
 * is visible because we cannot see interactions inside the iframe. If you have this in your application's architecture,
 * then this function will attach event listeners inside the iframe. You should call `addWindowFocusTracking` with an
 * element from inside the window you wish to add. We'll retrieve the relevant elements based on that.
 * Note, you do not need to call this for the default window, as we call it for you.
 *
 * When you are ready to stop listening, but you do not wish to unmount the iframe, you may call the cleanup function
 * returned by `addWindowFocusTracking`. Otherwise, when you unmount the iframe, all listeners and state will be cleaned
 * up automatically for you.
 *
 * @param element @default document.body - The element provided will be used to get the window to add.
 * @returns A function to remove the event listeners and cleanup the state.
 */
export function addWindowFocusTracking(element?: HTMLElement | null): () => void;
/**
 * If true, keyboard focus is visible.
 */
export function isFocusVisible(): boolean;
export function getInteractionModality(): Modality | null;
export function setInteractionModality(modality: Modality): void;
/**
 * Keeps state of the current modality.
 */
export function useInteractionModality(): Modality | null;
/**
 * Manages focus visible state for the page, and subscribes individual components for updates.
 */
export function useFocusVisible(props?: FocusVisibleProps): FocusVisibleResult;
/**
 * Listens for trigger change and reports if focus is visible (i.e., modality is not pointer).
 */
export function useFocusVisibleListener(fn: FocusVisibleHandler, deps: ReadonlyArray<any>, opts?: {
    isTextInput?: boolean;
}): void;
export interface FocusWithinProps {
    /** Whether the focus within events should be disabled. */
    isDisabled?: boolean;
    /** Handler that is called when the target element or a descendant receives focus. */
    onFocusWithin?: (e: FocusEvent) => void;
    /** Handler that is called when the target element and all descendants lose focus. */
    onBlurWithin?: (e: FocusEvent) => void;
    /** Handler that is called when the the focus within state changes. */
    onFocusWithinChange?: (isFocusWithin: boolean) => void;
}
export interface FocusWithinResult {
    /** Props to spread onto the target element. */
    focusWithinProps: DOMAttributes;
}
/**
 * Handles focus events for the target and its descendants.
 */
export function useFocusWithin(props: FocusWithinProps): FocusWithinResult;
export interface HoverProps extends HoverEvents {
    /** Whether the hover events should be disabled. */
    isDisabled?: boolean;
}
export interface HoverResult {
    /** Props to spread on the target element. */
    hoverProps: DOMAttributes;
    isHovered: boolean;
}
/**
 * Handles pointer hover interactions for an element. Normalizes behavior
 * across browsers and platforms, and ignores emulated mouse events on touch devices.
 */
export function useHover(props: HoverProps): HoverResult;
export interface InteractOutsideProps {
    ref: RefObject<Element | null>;
    onInteractOutside?: (e: PointerEvent) => void;
    onInteractOutsideStart?: (e: PointerEvent) => void;
    /** Whether the interact outside events should be disabled. */
    isDisabled?: boolean;
}
/**
 * Example, used in components like Dialogs and Popovers so they can close
 * when a user clicks outside them.
 */
export function useInteractOutside(props: InteractOutsideProps): void;
export interface MoveResult {
    /** Props to spread on the target element. */
    moveProps: DOMAttributes;
}
/**
 * Handles move interactions across mouse, touch, and keyboard, including dragging with
 * the mouse or touch, and using the arrow keys. Normalizes behavior across browsers and
 * platforms, and ignores emulated mouse events on touch devices.
 */
export function useMove(props: MoveEvents): MoveResult;
export interface ScrollWheelProps extends ScrollEvents {
    /** Whether the scroll listener should be disabled. */
    isDisabled?: boolean;
}
export function useScrollWheel(props: ScrollWheelProps, ref: RefObject<HTMLElement | null>): void;
export interface LongPressProps {
    /** Whether long press events should be disabled. */
    isDisabled?: boolean;
    /** Handler that is called when a long press interaction starts. */
    onLongPressStart?: (e: LongPressEvent) => void;
    /**
     * Handler that is called when a long press interaction ends, either
     * over the target or when the pointer leaves the target.
     */
    onLongPressEnd?: (e: LongPressEvent) => void;
    /**
     * Handler that is called when the threshold time is met while
     * the press is over the target.
     */
    onLongPress?: (e: LongPressEvent) => void;
    /**
     * The amount of time in milliseconds to wait before triggering a long press.
     * @default 500ms
     */
    threshold?: number;
    /**
     * A description for assistive techology users indicating that a long press
     * action is available, e.g. "Long press to open menu".
     */
    accessibilityDescription?: string;
}
export interface LongPressResult {
    /** Props to spread on the target element. */
    longPressProps: DOMAttributes;
}
/**
 * Handles long press interactions across mouse and touch devices. Supports a customizable time threshold,
 * accessibility description, and normalizes behavior across browsers and devices.
 */
export function useLongPress(props: LongPressProps): LongPressResult;
/**
 * A utility function that focuses an element while avoiding undesired side effects such
 * as page scrolling and screen reader issues with CSS transitions.
 */
export function focusSafely(element: FocusableElement): void;
export type { PressEvent, PressEvents, MoveStartEvent, MoveMoveEvent, MoveEndEvent, MoveEvents, HoverEvent, HoverEvents, FocusEvents, KeyboardEvents } from '@react-types/shared';

//# sourceMappingURL=types.d.ts.map
