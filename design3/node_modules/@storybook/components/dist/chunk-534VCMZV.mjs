import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_ignore=__commonJS({"../../node_modules/refractor/lang/ignore.js"(exports,module){module.exports=ignore;ignore.displayName="ignore";ignore.aliases=["gitignore","hgignore","npmignore"];function ignore(Prism){(function(Prism2){Prism2.languages.ignore={comment:/^#.*/m,entry:{pattern:/\S(?:.*(?:(?:\\ )|\S))?/,alias:"string",inside:{operator:/^!|\*\*?|\?/,regex:{pattern:/(^|[^\\])\[[^\[\]]*\]/,lookbehind:!0},punctuation:/\//}}},Prism2.languages.gitignore=Prism2.languages.ignore,Prism2.languages.hgignore=Prism2.languages.ignore,Prism2.languages.npmignore=Prism2.languages.ignore;})(Prism);}}});

export { require_ignore };
