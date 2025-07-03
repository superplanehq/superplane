export declare abstract class Machine<State, Event extends {
    type: number | string;
}> {
    #private;
    disposables: {
        addEventListener<TEventName extends keyof WindowEventMap>(element: Window | Document | HTMLElement, name: TEventName, listener: (event: WindowEventMap[TEventName]) => any, options?: boolean | AddEventListenerOptions | undefined): () => void;
        requestAnimationFrame(callback: FrameRequestCallback): () => void;
        nextFrame(callback: FrameRequestCallback): () => void;
        setTimeout(callback: (...args: any[]) => void, ms?: number | undefined, ...args: any[]): () => void;
        microTask(cb: () => void): () => void;
        style(node: ElementCSSInlineStyle, property: string, value: string): () => void;
        group(cb: (d: {
            addEventListener<TEventName extends keyof WindowEventMap>(element: Window | Document | HTMLElement, name: TEventName, listener: (event: WindowEventMap[TEventName]) => any, options?: boolean | AddEventListenerOptions | undefined): () => void;
            requestAnimationFrame(callback: FrameRequestCallback): () => void;
            nextFrame(callback: FrameRequestCallback): () => void;
            setTimeout(callback: (...args: any[]) => void, ms?: number | undefined, ...args: any[]): () => void;
            microTask(cb: () => void): () => void;
            style(node: ElementCSSInlineStyle, property: string, value: string): () => void;
            group(cb: any): () => void;
            add(cb: () => void): () => void;
            dispose(): void;
        }) => void): () => void;
        add(cb: () => void): () => void;
        dispose(): void;
    };
    constructor(initialState: State);
    dispose(): void;
    get state(): Readonly<State>;
    abstract reduce(state: Readonly<State>, event: Event): Readonly<State>;
    subscribe<Slice>(selector: (state: Readonly<State>) => Slice, callback: (state: Slice) => void): () => void;
    on(type: Event['type'], callback: (state: State, event: Event) => void): () => void;
    send(event: Event): void;
}
export declare function shallowEqual(a: any, b: any): boolean;
export declare function batch<F extends (...args: any[]) => void, P extends any[] = Parameters<F>>(setup: () => [callback: F, handle: () => void]): (...args: P) => void;
