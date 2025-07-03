import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_step21=__commonJS({"../../node_modules/highlight.js/lib/languages/step21.js"(exports,module){function step21(hljs){return {name:"STEP Part 21",aliases:["p21","step","stp"],case_insensitive:!0,keywords:{$pattern:"[A-Z_][A-Z0-9_.]*",keyword:"HEADER ENDSEC DATA"},contains:[{className:"meta",begin:"ISO-10303-21;",relevance:10},{className:"meta",begin:"END-ISO-10303-21;",relevance:10},hljs.C_LINE_COMMENT_MODE,hljs.C_BLOCK_COMMENT_MODE,hljs.COMMENT("/\\*\\*!","\\*/"),hljs.C_NUMBER_MODE,hljs.inherit(hljs.APOS_STRING_MODE,{illegal:null}),hljs.inherit(hljs.QUOTE_STRING_MODE,{illegal:null}),{className:"string",begin:"'",end:"'"},{className:"symbol",variants:[{begin:"#",end:"\\d+",illegal:"\\W"}]}]}}module.exports=step21;}});var step21CG7HMK55 = require_step21();

export { step21CG7HMK55 as default };
