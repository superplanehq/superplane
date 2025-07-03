import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_profile=__commonJS({"../../node_modules/highlight.js/lib/languages/profile.js"(exports,module){function profile(hljs){return {name:"Python profiler",contains:[hljs.C_NUMBER_MODE,{begin:"[a-zA-Z_][\\da-zA-Z_]+\\.[\\da-zA-Z_]{1,3}",end:":",excludeEnd:!0},{begin:"(ncalls|tottime|cumtime)",end:"$",keywords:"ncalls tottime|10 cumtime|10 filename",relevance:10},{begin:"function calls",end:"$",contains:[hljs.C_NUMBER_MODE],relevance:10},hljs.APOS_STRING_MODE,hljs.QUOTE_STRING_MODE,{className:"string",begin:"\\(",end:"\\)$",excludeBegin:!0,excludeEnd:!0,relevance:0}]}}module.exports=profile;}});var profileSRXMKWNO = require_profile();

export { profileSRXMKWNO as default };
