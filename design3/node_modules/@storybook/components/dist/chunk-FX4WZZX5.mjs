import { require_scheme } from './chunk-2NYPKDUU.mjs';
import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_racket=__commonJS({"../../node_modules/refractor/lang/racket.js"(exports,module){var refractorScheme=require_scheme();module.exports=racket;racket.displayName="racket";racket.aliases=["rkt"];function racket(Prism){Prism.register(refractorScheme),Prism.languages.racket=Prism.languages.extend("scheme",{"lambda-parameter":{pattern:/([(\[]lambda\s+[(\[])[^()\[\]'\s]+/,lookbehind:!0}}),Prism.languages.insertBefore("racket","string",{lang:{pattern:/^#lang.+/m,greedy:!0,alias:"keyword"}}),Prism.languages.rkt=Prism.languages.racket;}}});

export { require_racket };
