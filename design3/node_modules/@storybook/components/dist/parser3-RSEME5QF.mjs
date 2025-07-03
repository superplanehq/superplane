import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_parser3=__commonJS({"../../node_modules/highlight.js/lib/languages/parser3.js"(exports,module){function parser3(hljs){let CURLY_SUBCOMMENT=hljs.COMMENT(/\{/,/\}/,{contains:["self"]});return {name:"Parser3",subLanguage:"xml",relevance:0,contains:[hljs.COMMENT("^#","$"),hljs.COMMENT(/\^rem\{/,/\}/,{relevance:10,contains:[CURLY_SUBCOMMENT]}),{className:"meta",begin:"^@(?:BASE|USE|CLASS|OPTIONS)$",relevance:10},{className:"title",begin:"@[\\w\\-]+\\[[\\w^;\\-]*\\](?:\\[[\\w^;\\-]*\\])?(?:.*)$"},{className:"variable",begin:/\$\{?[\w\-.:]+\}?/},{className:"keyword",begin:/\^[\w\-.:]+/},{className:"number",begin:"\\^#[0-9a-fA-F]+"},hljs.C_NUMBER_MODE]}}module.exports=parser3;}});var parser3RSEME5QF = require_parser3();

export { parser3RSEME5QF as default };
