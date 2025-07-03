import { Channel } from './chunk-NH5GSF3H.mjs';
import { global } from '@storybook/global';
import { logger } from '@storybook/client-logger';
import { isJSON, parse, stringify } from 'telejson';
import invariant from 'tiny-invariant';

var {WebSocket}=global,WebsocketTransport=class{constructor({url,onError}){this.buffer=[];this.isReady=!1;this.socket=new WebSocket(url),this.socket.onopen=()=>{this.isReady=!0,this.flush();},this.socket.onmessage=({data})=>{let event=typeof data=="string"&&isJSON(data)?parse(data):data;invariant(this.handler,"WebsocketTransport handler should be set"),this.handler(event);},this.socket.onerror=e=>{onError&&onError(e);};}setHandler(handler){this.handler=handler;}send(event){this.isReady?this.sendNow(event):this.sendLater(event);}sendLater(event){this.buffer.push(event);}sendNow(event){let data=stringify(event,{maxDepth:15,allowFunction:!0});this.socket.send(data);}flush(){let{buffer}=this;this.buffer=[],buffer.forEach(event=>this.send(event));}};function createChannel({url,async=!1,onError=err=>logger.warn(err)}){let channelUrl=url;if(!channelUrl){let protocol=window.location.protocol==="http:"?"ws":"wss",{hostname,port}=window.location;channelUrl=`${protocol}://${hostname}:${port}/storybook-server-channel`;}let transport=new WebsocketTransport({url:channelUrl,onError});return new Channel({transport,async})}var websocket_default=createChannel;

export { WebsocketTransport, createChannel, websocket_default };
