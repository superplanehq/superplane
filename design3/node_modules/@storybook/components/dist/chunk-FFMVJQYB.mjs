import { require_c } from './chunk-77CDIRSA.mjs';
import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_bison=__commonJS({"../../node_modules/refractor/lang/bison.js"(exports,module){var refractorC=require_c();module.exports=bison;bison.displayName="bison";bison.aliases=[];function bison(Prism){Prism.register(refractorC),Prism.languages.bison=Prism.languages.extend("c",{}),Prism.languages.insertBefore("bison","comment",{bison:{pattern:/^(?:[^%]|%(?!%))*%%[\s\S]*?%%/,inside:{c:{pattern:/%\{[\s\S]*?%\}|\{(?:\{[^}]*\}|[^{}])*\}/,inside:{delimiter:{pattern:/^%?\{|%?\}$/,alias:"punctuation"},"bison-variable":{pattern:/[$@](?:<[^\s>]+>)?[\w$]+/,alias:"variable",inside:{punctuation:/<|>/}},rest:Prism.languages.c}},comment:Prism.languages.c.comment,string:Prism.languages.c.string,property:/\S+(?=:)/,keyword:/%\w+/,number:{pattern:/(^|[^@])\b(?:0x[\da-f]+|\d+)/i,lookbehind:!0},punctuation:/%[%?]|[|:;\[\]<>]/}}});}}});

export { require_bison };
