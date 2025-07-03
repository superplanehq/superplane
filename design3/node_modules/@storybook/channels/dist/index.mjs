import { PostMessageTransport } from './chunk-BNMUBNN5.mjs';
export { PostMessageTransport, createChannel as createPostMessageChannel } from './chunk-BNMUBNN5.mjs';
import { WebsocketTransport } from './chunk-V4SVTEPD.mjs';
export { WebsocketTransport, createChannel as createWebSocketChannel } from './chunk-V4SVTEPD.mjs';
import { Channel } from './chunk-NH5GSF3H.mjs';
export { Channel } from './chunk-NH5GSF3H.mjs';
import { global } from '@storybook/global';

var {CONFIG_TYPE}=global,src_default=Channel;function createBrowserChannel({page,extraTransports=[]}){let transports=[new PostMessageTransport({page}),...extraTransports];if(CONFIG_TYPE==="DEVELOPMENT"){let protocol=window.location.protocol==="http:"?"ws":"wss",{hostname,port}=window.location,channelUrl=`${protocol}://${hostname}:${port}/storybook-server-channel`;transports.push(new WebsocketTransport({url:channelUrl,onError:()=>{}}));}return new Channel({transports})}

export { createBrowserChannel, src_default as default };
