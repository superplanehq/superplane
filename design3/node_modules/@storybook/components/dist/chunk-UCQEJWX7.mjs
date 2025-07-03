import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_editorconfig=__commonJS({"../../node_modules/refractor/lang/editorconfig.js"(exports,module){module.exports=editorconfig;editorconfig.displayName="editorconfig";editorconfig.aliases=[];function editorconfig(Prism){Prism.languages.editorconfig={comment:/[;#].*/,section:{pattern:/(^[ \t]*)\[.+\]/m,lookbehind:!0,alias:"selector",inside:{regex:/\\\\[\[\]{},!?.*]/,operator:/[!?]|\.\.|\*{1,2}/,punctuation:/[\[\]{},]/}},key:{pattern:/(^[ \t]*)[^\s=]+(?=[ \t]*=)/m,lookbehind:!0,alias:"attr-name"},value:{pattern:/=.*/,alias:"attr-value",inside:{punctuation:/^=/}}};}}});

export { require_editorconfig };
