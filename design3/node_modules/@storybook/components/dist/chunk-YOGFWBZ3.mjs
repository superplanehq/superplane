import { require_markup_templating } from './chunk-X4QSBS7E.mjs';
import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_ejs=__commonJS({"../../node_modules/refractor/lang/ejs.js"(exports,module){var refractorMarkupTemplating=require_markup_templating();module.exports=ejs;ejs.displayName="ejs";ejs.aliases=["eta"];function ejs(Prism){Prism.register(refractorMarkupTemplating),function(Prism2){Prism2.languages.ejs={delimiter:{pattern:/^<%[-_=]?|[-_]?%>$/,alias:"punctuation"},comment:/^#[\s\S]*/,"language-javascript":{pattern:/[\s\S]+/,inside:Prism2.languages.javascript}},Prism2.hooks.add("before-tokenize",function(env){var ejsPattern=/<%(?!%)[\s\S]+?%>/g;Prism2.languages["markup-templating"].buildPlaceholders(env,"ejs",ejsPattern);}),Prism2.hooks.add("after-tokenize",function(env){Prism2.languages["markup-templating"].tokenizePlaceholders(env,"ejs");}),Prism2.languages.eta=Prism2.languages.ejs;}(Prism);}}});

export { require_ejs };
