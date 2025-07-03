import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_flix=__commonJS({"../../node_modules/highlight.js/lib/languages/flix.js"(exports,module){function flix(hljs){let CHAR={className:"string",begin:/'(.|\\[xXuU][a-zA-Z0-9]+)'/},STRING={className:"string",variants:[{begin:'"',end:'"'}]},METHOD={className:"function",beginKeywords:"def",end:/[:={\[(\n;]/,excludeEnd:!0,contains:[{className:"title",relevance:0,begin:/[^0-9\n\t "'(),.`{}\[\]:;][^\n\t "'(),.`{}\[\]:;]+|[^0-9\n\t "'(),.`{}\[\]:;=]/}]};return {name:"Flix",keywords:{literal:"true false",keyword:"case class def else enum if impl import in lat rel index let match namespace switch type yield with"},contains:[hljs.C_LINE_COMMENT_MODE,hljs.C_BLOCK_COMMENT_MODE,CHAR,STRING,METHOD,hljs.C_NUMBER_MODE]}}module.exports=flix;}});var flixJZXEY4PY = require_flix();

export { flixJZXEY4PY as default };
