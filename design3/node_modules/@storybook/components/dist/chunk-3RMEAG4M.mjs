import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_n4js=__commonJS({"../../node_modules/refractor/lang/n4js.js"(exports,module){module.exports=n4js;n4js.displayName="n4js";n4js.aliases=["n4jsd"];function n4js(Prism){Prism.languages.n4js=Prism.languages.extend("javascript",{keyword:/\b(?:Array|any|boolean|break|case|catch|class|const|constructor|continue|debugger|declare|default|delete|do|else|enum|export|extends|false|finally|for|from|function|get|if|implements|import|in|instanceof|interface|let|module|new|null|number|package|private|protected|public|return|set|static|string|super|switch|this|throw|true|try|typeof|var|void|while|with|yield)\b/}),Prism.languages.insertBefore("n4js","constant",{annotation:{pattern:/@+\w+/,alias:"operator"}}),Prism.languages.n4jsd=Prism.languages.n4js;}}});

export { require_n4js };
