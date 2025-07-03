import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_ebnf=__commonJS({"../../node_modules/highlight.js/lib/languages/ebnf.js"(exports,module){function ebnf(hljs){let commentMode=hljs.COMMENT(/\(\*/,/\*\)/),nonTerminalMode={className:"attribute",begin:/^[ ]*[a-zA-Z]+([\s_-]+[a-zA-Z]+)*/},ruleBodyMode={begin:/=/,end:/[.;]/,contains:[commentMode,{className:"meta",begin:/\?.*\?/},{className:"string",variants:[hljs.APOS_STRING_MODE,hljs.QUOTE_STRING_MODE,{begin:"`",end:"`"}]}]};return {name:"Extended Backus-Naur Form",illegal:/\S/,contains:[commentMode,nonTerminalMode,ruleBodyMode]}}module.exports=ebnf;}});var ebnf75RALJTN = require_ebnf();

export { ebnf75RALJTN as default };
