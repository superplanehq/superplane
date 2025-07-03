import { __commonJS } from './chunk-JRLSWQMA.mjs';

var require_git=__commonJS({"../../node_modules/refractor/lang/git.js"(exports,module){module.exports=git;git.displayName="git";git.aliases=[];function git(Prism){Prism.languages.git={comment:/^#.*/m,deleted:/^[-–].*/m,inserted:/^\+.*/m,string:/("|')(?:\\.|(?!\1)[^\\\r\n])*\1/,command:{pattern:/^.*\$ git .*$/m,inside:{parameter:/\s--?\w+/}},coord:/^@@.*@@$/m,"commit-sha1":/^commit \w{40}$/m};}}});

export { require_git };
