var l=Object.defineProperty;var n=Object.getOwnPropertyDescriptor;var p=Object.getOwnPropertyNames;var m=Object.prototype.hasOwnProperty;var q=(t,e)=>{for(var o in e)l(t,o,{get:e[o],enumerable:!0})},u=(t,e,o,s)=>{if(e&&typeof e=="object"||typeof e=="function")for(let r of p(e))!m.call(t,r)&&r!==o&&l(t,r,{get:()=>e[r],enumerable:!(s=n(e,r))||s.enumerable});return t};var w=t=>u(l({},"__esModule",{value:!0}),t);var h={};q(h,{closest:()=>y,matches:()=>x,querySelector:()=>S,querySelectorAll:()=>a});module.exports=w(h);var c=require("./js/matcher.js");/*!
 * DOM Selector - A CSS selector engine.
 * @license MIT
 * @copyright asamuzaK (Kazz)
 * @see {@link https://github.com/asamuzaK/domSelector/blob/main/LICENSE}
 */const x=(t,e,o)=>new c.Matcher(t,e,o).matches(),y=(t,e,o)=>new c.Matcher(t,e,o).closest(),S=(t,e,o)=>new c.Matcher(t,e,o).querySelector(),a=(t,e,o)=>new c.Matcher(t,e,o).querySelectorAll();0&&(module.exports={closest,matches,querySelector,querySelectorAll});
//# sourceMappingURL=index.js.map
