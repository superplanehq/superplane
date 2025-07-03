import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_bbcode=__commonJS({"../../node_modules/refractor/lang/bbcode.js"(exports,module){module.exports=bbcode;bbcode.displayName="bbcode";bbcode.aliases=["shortcode"];function bbcode(Prism){Prism.languages.bbcode={tag:{pattern:/\[\/?[^\s=\]]+(?:\s*=\s*(?:"[^"]*"|'[^']*'|[^\s'"\]=]+))?(?:\s+[^\s=\]]+\s*=\s*(?:"[^"]*"|'[^']*'|[^\s'"\]=]+))*\s*\]/,inside:{tag:{pattern:/^\[\/?[^\s=\]]+/,inside:{punctuation:/^\[\/?/}},"attr-value":{pattern:/=\s*(?:"[^"]*"|'[^']*'|[^\s'"\]=]+)/,inside:{punctuation:[/^=/,{pattern:/^(\s*)["']|["']$/,lookbehind:!0}]}},punctuation:/\]/,"attr-name":/[^\s=\]]+/}}},Prism.languages.shortcode=Prism.languages.bbcode;}}});

export { require_bbcode };
