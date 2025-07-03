import { require_c } from './chunk-77CDIRSA.mjs';
import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_objectivec=__commonJS({"../../node_modules/refractor/lang/objectivec.js"(exports,module){var refractorC=require_c();module.exports=objectivec;objectivec.displayName="objectivec";objectivec.aliases=["objc"];function objectivec(Prism){Prism.register(refractorC),Prism.languages.objectivec=Prism.languages.extend("c",{string:{pattern:/@?"(?:\\(?:\r\n|[\s\S])|[^"\\\r\n])*"/,greedy:!0},keyword:/\b(?:asm|auto|break|case|char|const|continue|default|do|double|else|enum|extern|float|for|goto|if|in|inline|int|long|register|return|self|short|signed|sizeof|static|struct|super|switch|typedef|typeof|union|unsigned|void|volatile|while)\b|(?:@interface|@end|@implementation|@protocol|@class|@public|@protected|@private|@property|@try|@catch|@finally|@throw|@synthesize|@dynamic|@selector)\b/,operator:/-[->]?|\+\+?|!=?|<<?=?|>>?=?|==?|&&?|\|\|?|[~^%?*\/@]/}),delete Prism.languages.objectivec["class-name"],Prism.languages.objc=Prism.languages.objectivec;}}});

export { require_objectivec };
