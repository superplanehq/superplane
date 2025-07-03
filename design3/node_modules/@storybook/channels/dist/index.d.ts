import { C as Channel, a as Config, b as ChannelTransport } from './main-c55d8855.js';
export { c as ChannelEvent, d as ChannelHandler, L as Listener } from './main-c55d8855.js';
export { PostMessageTransport, default as createPostMessageChannel } from './postmessage/index.js';
export { WebsocketTransport, default as createWebSocketChannel } from './websocket/index.js';

type Options = Config & {
    extraTransports?: ChannelTransport[];
};
/**
 * Creates a new browser channel instance.
 * @param {Options} options - The options object.
 * @param {Page} options.page - The puppeteer page instance.
 * @param {ChannelTransport[]} [options.extraTransports=[]] - An optional array of extra channel transports.
 * @returns {Channel} - The new channel instance.
 */
declare function createBrowserChannel({ page, extraTransports }: Options): Channel;

export { Channel, ChannelTransport, createBrowserChannel, Channel as default };
