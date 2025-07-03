interface Config {
    page: 'manager' | 'preview';
}
type ChannelHandler = (event: ChannelEvent) => void;
interface ChannelTransport {
    send(event: ChannelEvent, options?: any): void;
    setHandler(handler: ChannelHandler): void;
}
interface ChannelEvent {
    type: string;
    from: string;
    args: any[];
}
interface Listener {
    (...args: any[]): void;
}
interface ChannelArgsSingle {
    transport?: ChannelTransport;
    async?: boolean;
}
interface ChannelArgsMulti {
    transports: ChannelTransport[];
    async?: boolean;
}

declare class Channel {
    readonly isAsync: boolean;
    private sender;
    private events;
    private data;
    private readonly transports;
    constructor(input: ChannelArgsMulti);
    constructor(input: ChannelArgsSingle);
    get hasTransport(): boolean;
    addListener(eventName: string, listener: Listener): void;
    emit(eventName: string, ...args: any): void;
    last(eventName: string): any;
    eventNames(): string[];
    listenerCount(eventName: string): number;
    listeners(eventName: string): Listener[] | undefined;
    once(eventName: string, listener: Listener): void;
    removeAllListeners(eventName?: string): void;
    removeListener(eventName: string, listener: Listener): void;
    on(eventName: string, listener: Listener): void;
    off(eventName: string, listener: Listener): void;
    private handleEvent;
    private onceListener;
}

export { Channel as C, Listener as L, Config as a, ChannelTransport as b, ChannelEvent as c, ChannelHandler as d };
