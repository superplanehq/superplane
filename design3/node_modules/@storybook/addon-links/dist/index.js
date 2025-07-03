'use strict';

var tsDedent = require('ts-dedent');
var global = require('@storybook/global');
var previewApi = require('@storybook/preview-api');
var coreEvents = require('@storybook/core-events');
var csf = require('@storybook/csf');

var PARAM_KEY="links";var{document,HTMLElement}=global.global;function parseQuery(queryString){let query={},pairs=(queryString[0]==="?"?queryString.substring(1):queryString).split("&").filter(Boolean);for(let i=0;i<pairs.length;i++){let pair=pairs[i].split("=");query[decodeURIComponent(pair[0])]=decodeURIComponent(pair[1]||"");}return query}var navigate=params=>previewApi.addons.getChannel().emit(coreEvents.SELECT_STORY,params),hrefTo=(title,name)=>new Promise(resolve=>{let{location}=document,query=parseQuery(location.search),existingId=[].concat(query.id)[0],titleToLink=title||existingId.split("--",2)[0],path=`/story/${csf.toId(titleToLink,name)}`,sbPath=location.pathname.replace(/iframe\.html$/,""),url=`${location.origin+sbPath}?${Object.entries({path}).map(item=>`${item[0]}=${item[1]}`).join("&")}`;resolve(url);}),valueOrCall=args=>value=>typeof value=="function"?value(...args):value,linkTo=(idOrTitle,nameInput)=>(...args)=>{let resolver=valueOrCall(args),title=resolver(idOrTitle),name=nameInput?resolver(nameInput):!1;title?.match(/--/)&&!name?navigate({storyId:title}):name&&title?navigate({kind:title,story:name}):title?navigate({kind:title}):name&&navigate({story:name});},linksListener=e=>{let{target}=e;if(!(target instanceof HTMLElement))return;let element=target,{sbKind:kind,sbStory:story}=element.dataset;(kind||story)&&(e.preventDefault(),navigate({kind,story}));},hasListener=!1,on=()=>{hasListener||(hasListener=!0,document.addEventListener("click",linksListener));},off=()=>{hasListener&&(hasListener=!1,document.removeEventListener("click",linksListener));},withLinks=previewApi.makeDecorator({name:"withLinks",parameterName:PARAM_KEY,wrapper:(getStory,context)=>(on(),previewApi.addons.getChannel().once(coreEvents.STORY_CHANGED,off),getStory(context))});var hasWarned=!1;function LinkTo(){return hasWarned||(console.error(tsDedent.dedent`
      LinkTo has moved to addon-links/react:
      import LinkTo from '@storybook/addon-links/react';
    `),hasWarned=!0),null}

exports.LinkTo = LinkTo;
exports.hrefTo = hrefTo;
exports.linkTo = linkTo;
exports.navigate = navigate;
exports.withLinks = withLinks;
