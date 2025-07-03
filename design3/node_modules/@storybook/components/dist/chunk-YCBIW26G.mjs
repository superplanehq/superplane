import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_ebnf=__commonJS({"../../node_modules/refractor/lang/ebnf.js"(exports,module){module.exports=ebnf;ebnf.displayName="ebnf";ebnf.aliases=[];function ebnf(Prism){Prism.languages.ebnf={comment:/\(\*[\s\S]*?\*\)/,string:{pattern:/"[^"\r\n]*"|'[^'\r\n]*'/,greedy:!0},special:{pattern:/\?[^?\r\n]*\?/,greedy:!0,alias:"class-name"},definition:{pattern:/^([\t ]*)[a-z]\w*(?:[ \t]+[a-z]\w*)*(?=\s*=)/im,lookbehind:!0,alias:["rule","keyword"]},rule:/\b[a-z]\w*(?:[ \t]+[a-z]\w*)*\b/i,punctuation:/\([:/]|[:/]\)|[.,;()[\]{}]/,operator:/[-=|*/!]/};}}});

export { require_ebnf };
