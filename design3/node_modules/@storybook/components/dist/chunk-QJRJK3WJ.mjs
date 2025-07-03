import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_go_module=__commonJS({"../../node_modules/refractor/lang/go-module.js"(exports,module){module.exports=goModule;goModule.displayName="goModule";goModule.aliases=[];function goModule(Prism){Prism.languages["go-mod"]=Prism.languages["go-module"]={comment:{pattern:/\/\/.*/,greedy:!0},version:{pattern:/(^|[\s()[\],])v\d+\.\d+\.\d+(?:[+-][-+.\w]*)?(?![^\s()[\],])/,lookbehind:!0,alias:"number"},"go-version":{pattern:/((?:^|\s)go\s+)\d+(?:\.\d+){1,2}/,lookbehind:!0,alias:"number"},keyword:{pattern:/^([ \t]*)(?:exclude|go|module|replace|require|retract)\b/m,lookbehind:!0},operator:/=>/,punctuation:/[()[\],]/};}}});

export { require_go_module };
