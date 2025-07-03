import { b as ChannelTransport, d as ChannelHandler, C as Channel } from '../main-c55d8855.js';

type OnError = (message: Event) => void;
interface WebsocketTransportArgs {
    url: string;
    onError: OnError;
}
interface CreateChannelArgs {
    url?: string;
    async?: boolean;
    onError?: OnError;
}
declare class WebsocketTransport implements ChannelTransport {
    private buffer;
    private handler?;
    private socket;
    private isReady;
    constructor({ url, onError }: WebsocketTransportArgs);
    setHandler(handler: ChannelHandler): void;
    send(event: any): void;
    private sendLater;
    private sendNow;
    private flush;
}
/**
 * @deprecated This function is deprecated. Use the `createBrowserChannel` factory function from `@storybook/channels` instead. This API will be removed in 8.0.
 * @param {CreateChannelArgs} options - The options for creating the channel.
 * @param {string} [options.url] - The URL of the WebSocket server to connect to.
 * @param {boolean} [options.async=false] - Whether the channel should be asynchronous.
 * @param {OnError} [options.onError] - A function to handle errors that occur during the channel's lifetime.
 * @returns {Channel} - The newly created channel.
 */
declare function createChannel({ url, async, onError, }: CreateChannelArgs): Channel;

export { WebsocketTransport, createChannel, createChannel as default };
