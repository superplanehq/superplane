import { require_haskell } from './chunk-JHPNCVC3.mjs';
import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_idris=__commonJS({"../../node_modules/refractor/lang/idris.js"(exports,module){var refractorHaskell=require_haskell();module.exports=idris;idris.displayName="idris";idris.aliases=["idr"];function idris(Prism){Prism.register(refractorHaskell),Prism.languages.idris=Prism.languages.extend("haskell",{comment:{pattern:/(?:(?:--|\|\|\|).*$|\{-[\s\S]*?-\})/m},keyword:/\b(?:Type|case|class|codata|constructor|corecord|data|do|dsl|else|export|if|implementation|implicit|import|impossible|in|infix|infixl|infixr|instance|interface|let|module|mutual|namespace|of|parameters|partial|postulate|private|proof|public|quoteGoal|record|rewrite|syntax|then|total|using|where|with)\b/,builtin:void 0}),Prism.languages.insertBefore("idris","keyword",{"import-statement":{pattern:/(^\s*import\s+)(?:[A-Z][\w']*)(?:\.[A-Z][\w']*)*/m,lookbehind:!0,inside:{punctuation:/\./}}}),Prism.languages.idr=Prism.languages.idris;}}});

export { require_idris };
