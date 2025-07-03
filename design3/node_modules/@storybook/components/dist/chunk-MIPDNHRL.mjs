import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_systemd=__commonJS({"../../node_modules/refractor/lang/systemd.js"(exports,module){module.exports=systemd;systemd.displayName="systemd";systemd.aliases=[];function systemd(Prism){(function(Prism2){var comment={pattern:/^[;#].*/m,greedy:!0},quotesSource=/"(?:[^\r\n"\\]|\\(?:[^\r]|\r\n?))*"(?!\S)/.source;Prism2.languages.systemd={comment,section:{pattern:/^\[[^\n\r\[\]]*\](?=[ \t]*$)/m,greedy:!0,inside:{punctuation:/^\[|\]$/,"section-name":{pattern:/[\s\S]+/,alias:"selector"}}},key:{pattern:/^[^\s=]+(?=[ \t]*=)/m,greedy:!0,alias:"attr-name"},value:{pattern:RegExp(/(=[ \t]*(?!\s))/.source+"(?:"+quotesSource+`|(?=[^"\r
]))(?:`+(/[^\s\\]/.source+'|[ 	]+(?:(?![ 	"])|'+quotesSource+")|"+/\\[\r\n]+(?:[#;].*[\r\n]+)*(?![#;])/.source)+")*"),lookbehind:!0,greedy:!0,alias:"attr-value",inside:{comment,quoted:{pattern:RegExp(/(^|\s)/.source+quotesSource),lookbehind:!0,greedy:!0},punctuation:/\\$/m,boolean:{pattern:/^(?:false|no|off|on|true|yes)$/,greedy:!0}}},punctuation:/=/};})(Prism);}}});

export { require_systemd };
