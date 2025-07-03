import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_nix=__commonJS({"../../node_modules/highlight.js/lib/languages/nix.js"(exports,module){function nix(hljs){let NIX_KEYWORDS={keyword:"rec with let in inherit assert if else then",literal:"true false or and null",built_in:"import abort baseNameOf dirOf isNull builtins map removeAttrs throw toString derivation"},ANTIQUOTE={className:"subst",begin:/\$\{/,end:/\}/,keywords:NIX_KEYWORDS},ATTRS={begin:/[a-zA-Z0-9-_]+(\s*=)/,returnBegin:!0,relevance:0,contains:[{className:"attr",begin:/\S+/}]},STRING={className:"string",contains:[ANTIQUOTE],variants:[{begin:"''",end:"''"},{begin:'"',end:'"'}]},EXPRESSIONS=[hljs.NUMBER_MODE,hljs.HASH_COMMENT_MODE,hljs.C_BLOCK_COMMENT_MODE,STRING,ATTRS];return ANTIQUOTE.contains=EXPRESSIONS,{name:"Nix",aliases:["nixos"],keywords:NIX_KEYWORDS,contains:EXPRESSIONS}}module.exports=nix;}});var nixLHIXO4QU = require_nix();

export { nixLHIXO4QU as default };
