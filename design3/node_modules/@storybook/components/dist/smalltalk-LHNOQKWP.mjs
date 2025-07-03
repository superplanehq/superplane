import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_smalltalk=__commonJS({"../../node_modules/highlight.js/lib/languages/smalltalk.js"(exports,module){function smalltalk(hljs){let VAR_IDENT_RE="[a-z][a-zA-Z0-9_]*",CHAR={className:"string",begin:"\\$.{1}"},SYMBOL={className:"symbol",begin:"#"+hljs.UNDERSCORE_IDENT_RE};return {name:"Smalltalk",aliases:["st"],keywords:"self super nil true false thisContext",contains:[hljs.COMMENT('"','"'),hljs.APOS_STRING_MODE,{className:"type",begin:"\\b[A-Z][A-Za-z0-9_]*",relevance:0},{begin:VAR_IDENT_RE+":",relevance:0},hljs.C_NUMBER_MODE,SYMBOL,CHAR,{begin:"\\|[ ]*"+VAR_IDENT_RE+"([ ]+"+VAR_IDENT_RE+")*[ ]*\\|",returnBegin:!0,end:/\|/,illegal:/\S/,contains:[{begin:"(\\|[ ]*)?"+VAR_IDENT_RE}]},{begin:"#\\(",end:"\\)",contains:[hljs.APOS_STRING_MODE,CHAR,hljs.C_NUMBER_MODE,SYMBOL]}]}}module.exports=smalltalk;}});var smalltalkLHNOQKWP = require_smalltalk();

export { smalltalkLHNOQKWP as default };
