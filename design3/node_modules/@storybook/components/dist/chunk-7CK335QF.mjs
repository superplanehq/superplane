import { require_php } from './chunk-KTLKLE2L.mjs';
import { require_javadoclike } from './chunk-ZPDD6KTK.mjs';
import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_phpdoc=__commonJS({"../../node_modules/refractor/lang/phpdoc.js"(exports,module){var refractorPhp=require_php(),refractorJavadoclike=require_javadoclike();module.exports=phpdoc;phpdoc.displayName="phpdoc";phpdoc.aliases=[];function phpdoc(Prism){Prism.register(refractorPhp),Prism.register(refractorJavadoclike),function(Prism2){var typeExpression=/(?:\b[a-zA-Z]\w*|[|\\[\]])+/.source;Prism2.languages.phpdoc=Prism2.languages.extend("javadoclike",{parameter:{pattern:RegExp("(@(?:global|param|property(?:-read|-write)?|var)\\s+(?:"+typeExpression+"\\s+)?)\\$\\w+"),lookbehind:!0}}),Prism2.languages.insertBefore("phpdoc","keyword",{"class-name":[{pattern:RegExp("(@(?:global|package|param|property(?:-read|-write)?|return|subpackage|throws|var)\\s+)"+typeExpression),lookbehind:!0,inside:{keyword:/\b(?:array|bool|boolean|callback|double|false|float|int|integer|mixed|null|object|resource|self|string|true|void)\b/,punctuation:/[|\\[\]()]/}}]}),Prism2.languages.javadoclike.addSupport("php",Prism2.languages.phpdoc);}(Prism);}}});

export { require_phpdoc };
