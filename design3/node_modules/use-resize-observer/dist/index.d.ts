import { RefObject, RefCallback } from "react";
export declare type ObservedSize = {
    width: number | undefined;
    height: number | undefined;
};
export declare type ResizeHandler = (size: ObservedSize) => void;
declare type HookResponse<T extends Element> = {
    ref: RefCallback<T>;
} & ObservedSize;
export declare type ResizeObserverBoxOptions = "border-box" | "content-box" | "device-pixel-content-box";
declare global {
    interface ResizeObserverEntry {
        readonly devicePixelContentBoxSize: ReadonlyArray<ResizeObserverSize>;
    }
}
export declare type RoundingFunction = (n: number) => number;
declare function useResizeObserver<T extends Element>(opts?: {
    ref?: RefObject<T> | T | null | undefined;
    onResize?: ResizeHandler;
    box?: ResizeObserverBoxOptions;
    round?: RoundingFunction;
}): HookResponse<T>;
export default useResizeObserver;
