import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_roboconf=__commonJS({"../../node_modules/refractor/lang/roboconf.js"(exports,module){module.exports=roboconf;roboconf.displayName="roboconf";roboconf.aliases=[];function roboconf(Prism){Prism.languages.roboconf={comment:/#.*/,keyword:{pattern:/(^|\s)(?:(?:external|import)\b|(?:facet|instance of)(?=[ \t]+[\w-]+[ \t]*\{))/,lookbehind:!0},component:{pattern:/[\w-]+(?=[ \t]*\{)/,alias:"variable"},property:/[\w.-]+(?=[ \t]*:)/,value:{pattern:/(=[ \t]*(?![ \t]))[^,;]+/,lookbehind:!0,alias:"attr-value"},optional:{pattern:/\(optional\)/,alias:"builtin"},wildcard:{pattern:/(\.)\*/,lookbehind:!0,alias:"operator"},punctuation:/[{},.;:=]/};}}});

export { require_roboconf };
