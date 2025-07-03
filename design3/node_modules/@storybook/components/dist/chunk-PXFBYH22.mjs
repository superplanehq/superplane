import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_xml_doc=__commonJS({"../../node_modules/refractor/lang/xml-doc.js"(exports,module){module.exports=xmlDoc;xmlDoc.displayName="xmlDoc";xmlDoc.aliases=[];function xmlDoc(Prism){(function(Prism2){function insertDocComment(lang,docComment){Prism2.languages[lang]&&Prism2.languages.insertBefore(lang,"comment",{"doc-comment":docComment});}var tag=Prism2.languages.markup.tag,slashDocComment={pattern:/\/\/\/.*/,greedy:!0,alias:"comment",inside:{tag}},tickDocComment={pattern:/'''.*/,greedy:!0,alias:"comment",inside:{tag}};insertDocComment("csharp",slashDocComment),insertDocComment("fsharp",slashDocComment),insertDocComment("vbnet",tickDocComment);})(Prism);}}});

export { require_xml_doc };
