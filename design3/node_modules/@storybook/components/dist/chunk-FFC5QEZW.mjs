import { require_json } from './chunk-SWV57YYC.mjs';
import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_json5=__commonJS({"../../node_modules/refractor/lang/json5.js"(exports,module){var refractorJson=require_json();module.exports=json5;json5.displayName="json5";json5.aliases=[];function json5(Prism){Prism.register(refractorJson),function(Prism2){var string=/("|')(?:\\(?:\r\n?|\n|.)|(?!\1)[^\\\r\n])*\1/;Prism2.languages.json5=Prism2.languages.extend("json",{property:[{pattern:RegExp(string.source+"(?=\\s*:)"),greedy:!0},{pattern:/(?!\s)[_$a-zA-Z\xA0-\uFFFF](?:(?!\s)[$\w\xA0-\uFFFF])*(?=\s*:)/,alias:"unquoted"}],string:{pattern:string,greedy:!0},number:/[+-]?\b(?:NaN|Infinity|0x[a-fA-F\d]+)\b|[+-]?(?:\b\d+(?:\.\d*)?|\B\.\d+)(?:[eE][+-]?\d+\b)?/});}(Prism);}}});

export { require_json5 };
