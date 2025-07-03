import { b as ChannelTransport, a as Config, d as ChannelHandler, c as ChannelEvent, C as Channel } from '../main-c55d8855.js';

declare const KEY = "storybook-channel";
declare class PostMessageTransport implements ChannelTransport {
    private readonly config;
    private buffer;
    private handler?;
    private connected;
    constructor(config: Config);
    setHandler(handler: ChannelHandler): void;
    /**
     * Sends `event` to the associated window. If the window does not yet exist
     * the event will be stored in a buffer and sent when the window exists.
     * @param event
     */
    send(event: ChannelEvent, options?: any): Promise<any>;
    private flush;
    private getFrames;
    private getCurrentFrames;
    private getLocalFrame;
    private handleEvent;
}
/**
 * @deprecated This export is deprecated. Use `PostMessageTransport` instead. This API will be removed in 8.0.
 */
declare const PostmsgTransport: typeof PostMessageTransport;
/**
 * @deprecated This function is deprecated. Use the `createBrowserChannel` factory function from `@storybook/channels` instead. This API will be removed in 8.0.
 * @param {Config} config - The configuration object.
 * @returns {Channel} The channel instance.
 */
declare function createChannel({ page }: Config): Channel;

export { KEY, PostMessageTransport, PostmsgTransport, createChannel, createChannel as default };
