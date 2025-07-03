'use strict';

var telemetry = require('@storybook/telemetry');
var N = require('fs');

function _interopDefault (e) { return e && e.__esModule ? e : { default: e }; }

var N__default = /*#__PURE__*/_interopDefault(N);

var r=(e=>typeof require<"u"?require:typeof Proxy<"u"?new Proxy(e,{get:(o,n)=>(typeof require<"u"?require:o)[n]}):e)(function(e){if(typeof require<"u")return require.apply(this,arguments);throw new Error('Dynamic require of "'+e+'" is not supported')});var t="STORYBOOK_ADDON_ONBOARDING_CHANNEL";var f=async(e,o)=>{let{disableTelemetry:n}=await o.presets.apply("core",{});if(!n){let a=r.resolve("@storybook/addon-onboarding/package.json"),{version:s}=JSON.parse(N__default.default.readFileSync(a,{encoding:"utf-8"}));e.on(t,({type:i,...p})=>{i==="telemetry"&&telemetry.telemetry("addon-onboarding",{...p,addonVersion:s});});}return e};

exports.experimental_serverChannel = f;
