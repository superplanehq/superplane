import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_brainfuck=__commonJS({"../../node_modules/highlight.js/lib/languages/brainfuck.js"(exports,module){function brainfuck(hljs){let LITERAL={className:"literal",begin:/[+-]/,relevance:0};return {name:"Brainfuck",aliases:["bf"],contains:[hljs.COMMENT(`[^\\[\\]\\.,\\+\\-<> \r
]`,`[\\[\\]\\.,\\+\\-<> \r
]`,{returnEnd:!0,relevance:0}),{className:"title",begin:"[\\[\\]]",relevance:0},{className:"string",begin:"[\\.,]",relevance:0},{begin:/(?:\+\+|--)/,contains:[LITERAL]},LITERAL]}}module.exports=brainfuck;}});var brainfuckOHING4LJ = require_brainfuck();

export { brainfuckOHING4LJ as default };
