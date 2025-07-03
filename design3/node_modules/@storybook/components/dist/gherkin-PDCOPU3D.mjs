import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_gherkin=__commonJS({"../../node_modules/highlight.js/lib/languages/gherkin.js"(exports,module){function gherkin(hljs){return {name:"Gherkin",aliases:["feature"],keywords:"Feature Background Ability Business Need Scenario Scenarios Scenario Outline Scenario Template Examples Given And Then But When",contains:[{className:"symbol",begin:"\\*",relevance:0},{className:"meta",begin:"@[^@\\s]+"},{begin:"\\|",end:"\\|\\w*$",contains:[{className:"string",begin:"[^|]+"}]},{className:"variable",begin:"<",end:">"},hljs.HASH_COMMENT_MODE,{className:"string",begin:'"""',end:'"""'},hljs.QUOTE_STRING_MODE]}}module.exports=gherkin;}});var gherkinPDCOPU3D = require_gherkin();

export { gherkinPDCOPU3D as default };
