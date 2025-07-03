import type * as React from 'react';

export declare function activeElement(doc: Document): Element | null;

export declare function contains(parent?: Element | null, child?: Element | null): boolean;

export declare function getDocument(node: Element | null): Document;

export declare function getPlatform(): string;

export declare function getTarget(event: Event): EventTarget | null;

export declare function getUserAgent(): string;

export declare function isAndroid(): boolean;

export declare function isEventTargetWithin(event: Event, node: Node | null | undefined): boolean;

export declare function isJSDOM(): boolean;

export declare function isMac(): boolean;

export declare function isMouseLikePointerType(pointerType: string | undefined, strict?: boolean): boolean;

export declare function isReactEvent(event: any): event is React.SyntheticEvent;

export declare function isRootElement(element: Element): boolean;

export declare function isSafari(): boolean;

export declare function isTypeableCombobox(element: Element | null): boolean;

export declare function isTypeableElement(element: unknown): boolean;

export declare function isVirtualClick(event: MouseEvent | PointerEvent): boolean;

export declare function isVirtualPointerEvent(event: PointerEvent): boolean;

export declare function stopEvent(event: Event | React.SyntheticEvent): void;

export declare const TYPEABLE_SELECTOR: string;

export { }
