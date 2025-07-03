
function $parcel$defineInteropFlag(a) {
  Object.defineProperty(a, '__esModule', {value: true, configurable: true});
}

function $parcel$exportWildcard(dest, source) {
  Object.keys(source).forEach(function(key) {
    if (key === 'default' || key === '__esModule' || Object.prototype.hasOwnProperty.call(dest, key)) {
      return;
    }

    Object.defineProperty(dest, key, {
      enumerable: true,
      get: function get() {
        return source[key];
      }
    });
  });

  return dest;
}

function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$defineInteropFlag(module.exports);

$parcel$export(module.exports, "parse", () => $882b6d93070905b3$export$98e6a39c04603d36);
$parcel$export(module.exports, "stringify", () => $882b6d93070905b3$export$fac44ee5b035f737);
$parcel$export(module.exports, "default", () => $882b6d93070905b3$export$2e2bcd8739ae039);
var $cb508b9219b02820$exports = {};

$parcel$defineInteropFlag($cb508b9219b02820$exports);

$parcel$export($cb508b9219b02820$exports, "default", () => $cb508b9219b02820$export$2e2bcd8739ae039);
class $cb508b9219b02820$export$2e2bcd8739ae039 extends Error {
    constructor(filename, msg, lineno, column, css){
        super(filename + ':' + lineno + ':' + column + ': ' + msg);
        this.reason = msg;
        this.filename = filename;
        this.line = lineno;
        this.column = column;
        this.source = css;
    }
}


var $4bafb28828007b46$exports = {};

$parcel$defineInteropFlag($4bafb28828007b46$exports);

$parcel$export($4bafb28828007b46$exports, "default", () => $4bafb28828007b46$export$2e2bcd8739ae039);
/**
 * Store position information for a node
 */ class $4bafb28828007b46$export$2e2bcd8739ae039 {
    constructor(start, end, source){
        this.start = start;
        this.end = end;
        this.source = source;
    }
}


var $d103407e81c97042$exports = {};

$parcel$export($d103407e81c97042$exports, "CssTypes", () => $d103407e81c97042$export$9be5dd6e61d5d73a);
var $d103407e81c97042$export$9be5dd6e61d5d73a = /*#__PURE__*/ function(CssTypes) {
    CssTypes["stylesheet"] = "stylesheet";
    CssTypes["rule"] = "rule";
    CssTypes["declaration"] = "declaration";
    CssTypes["comment"] = "comment";
    CssTypes["container"] = "container";
    CssTypes["charset"] = "charset";
    CssTypes["document"] = "document";
    CssTypes["customMedia"] = "custom-media";
    CssTypes["fontFace"] = "font-face";
    CssTypes["host"] = "host";
    CssTypes["import"] = "import";
    CssTypes["keyframes"] = "keyframes";
    CssTypes["keyframe"] = "keyframe";
    CssTypes["layer"] = "layer";
    CssTypes["media"] = "media";
    CssTypes["namespace"] = "namespace";
    CssTypes["page"] = "page";
    CssTypes["startingStyle"] = "starting-style";
    CssTypes["supports"] = "supports";
    return CssTypes;
}({});


const $6fedb2016a78880b$export$82eb5486fce3d340 = 10000;
const $6fedb2016a78880b$export$c8b95ffeec50f24a = (string, search, position)=>{
    let currentPosition = position;
    let maxLoop = $6fedb2016a78880b$export$82eb5486fce3d340;
    do {
        const all = search.map((v)=>string.indexOf(v, currentPosition));
        all.push(string.indexOf('\\', currentPosition));
        const foundAll = all.filter((v)=>v !== -1);
        if (foundAll.length === 0) return -1;
        const found = Math.min(...foundAll);
        if (string[found] === '\\') {
            currentPosition = found + 2;
            maxLoop--;
        } else return found;
    }while (maxLoop > 0);
    throw new Error('Too many escaping');
};
const $6fedb2016a78880b$export$b8d42a8583d2e477 = (string, search, position)=>{
    let currentSearchPosition = position;
    let maxLoop = $6fedb2016a78880b$export$82eb5486fce3d340;
    do {
        const all = search.map((v)=>string.indexOf(v, currentSearchPosition));
        all.push(string.indexOf('(', currentSearchPosition));
        all.push(string.indexOf('"', currentSearchPosition));
        all.push(string.indexOf("'", currentSearchPosition));
        all.push(string.indexOf('\\', currentSearchPosition));
        const foundAll = all.filter((v)=>v !== -1);
        if (foundAll.length === 0) return -1;
        const firstMatchPos = Math.min(...foundAll);
        const char = string[firstMatchPos];
        switch(char){
            case '\\':
                currentSearchPosition = firstMatchPos + 2;
                break;
            case '(':
                {
                    const endPosition = $6fedb2016a78880b$export$b8d42a8583d2e477(string, [
                        ')'
                    ], firstMatchPos + 1);
                    if (endPosition === -1) return -1;
                    currentSearchPosition = endPosition + 1;
                }
                break;
            case '"':
                {
                    const endQuotePosition = $6fedb2016a78880b$export$c8b95ffeec50f24a(string, [
                        '"'
                    ], firstMatchPos + 1);
                    if (endQuotePosition === -1) return -1;
                    currentSearchPosition = endQuotePosition + 1;
                }
                break;
            case "'":
                {
                    const endQuotePosition = $6fedb2016a78880b$export$c8b95ffeec50f24a(string, [
                        "'"
                    ], firstMatchPos + 1);
                    if (endQuotePosition === -1) return -1;
                    currentSearchPosition = endQuotePosition + 1;
                }
                break;
            default:
                return firstMatchPos;
        }
        maxLoop--;
    }while (maxLoop > 0);
    throw new Error('Too many escaping');
};
const $6fedb2016a78880b$export$801dd37ac183521b = (string, search)=>{
    const result = [];
    let currentPosition = 0;
    while(currentPosition < string.length){
        const index = $6fedb2016a78880b$export$b8d42a8583d2e477(string, search, currentPosition);
        if (index === -1) {
            result.push(string.substring(currentPosition));
            return result;
        }
        result.push(string.substring(currentPosition, index));
        currentPosition = index + 1;
    }
    return result;
};


// http://www.w3.org/TR/CSS21/grammar.html
// https://github.com/visionmedia/css-parse/pull/49#issuecomment-30088027
// New rule => https://www.w3.org/TR/CSS22/syndata.html#comments
// [^] is equivalent to [.\n\r]
const $b499486c7f02abe7$var$commentre = /\/\*[^]*?(?:\*\/|$)/g;
const $b499486c7f02abe7$export$98e6a39c04603d36 = (css, options)=>{
    options = options || {};
    /**
   * Positional.
   */ let lineno = 1;
    let column = 1;
    /**
   * Update lineno and column based on `str`.
   */ function updatePosition(str) {
        const lines = str.match(/\n/g);
        if (lines) lineno += lines.length;
        const i = str.lastIndexOf('\n');
        column = ~i ? str.length - i : column + str.length;
    }
    /**
   * Mark position and patch `node.position`.
   */ function position() {
        const start = {
            line: lineno,
            column: column
        };
        return function(node) {
            node.position = new (0, $4bafb28828007b46$export$2e2bcd8739ae039)(start, {
                line: lineno,
                column: column
            }, options?.source || '');
            whitespace();
            return node;
        };
    }
    /**
   * Error `msg`.
   */ const errorsList = [];
    function error(msg) {
        const err = new (0, $cb508b9219b02820$export$2e2bcd8739ae039)(options?.source || '', msg, lineno, column, css);
        if (options?.silent) errorsList.push(err);
        else throw err;
    }
    /**
   * Parse stylesheet.
   */ function stylesheet() {
        const rulesList = rules();
        const result = {
            type: (0, $d103407e81c97042$export$9be5dd6e61d5d73a).stylesheet,
            stylesheet: {
                source: options?.source,
                rules: rulesList,
                parsingErrors: errorsList
            }
        };
        return result;
    }
    /**
   * Opening brace.
   */ function open() {
        const openMatch = /^{\s*/.exec(css);
        if (openMatch) {
            processMatch(openMatch);
            return true;
        }
        return false;
    }
    /**
   * Closing brace.
   */ function close() {
        const closeMatch = /^}/.exec(css);
        if (closeMatch) {
            processMatch(closeMatch);
            return true;
        }
        return false;
    }
    /**
   * Parse ruleset.
   */ function rules() {
        let node;
        const rules = [];
        whitespace();
        comments(rules);
        while(css.length && css.charAt(0) !== '}' && (node = atrule() || rule()))if (node) {
            rules.push(node);
            comments(rules);
        }
        return rules;
    }
    /**
   * Update position and css string. Return the matches
   */ function processMatch(m) {
        const str = m[0];
        updatePosition(str);
        css = css.slice(str.length);
        return m;
    }
    /**
   * Parse whitespace.
   */ function whitespace() {
        const m = /^\s*/.exec(css);
        if (m) processMatch(m);
    }
    /**
   * Parse comments;
   */ function comments(rules) {
        let c;
        rules = rules || [];
        while(c = comment())if (c) rules.push(c);
        return rules;
    }
    /**
   * Parse comment.
   */ function comment() {
        const pos = position();
        if ('/' !== css.charAt(0) || '*' !== css.charAt(1)) return;
        const m = /^\/\*[^]*?\*\//.exec(css);
        if (!m) return error('End of comment missing');
        processMatch(m);
        return pos({
            type: (0, $d103407e81c97042$export$9be5dd6e61d5d73a).comment,
            comment: m[0].slice(2, -2)
        });
    }
    /**
   * Parse selector.
   */ function selector() {
        const m = /^([^{]+)/.exec(css);
        if (!m) return;
        processMatch(m);
        // remove comment in selector;
        const res = $b499486c7f02abe7$var$trim(m[0]).replace($b499486c7f02abe7$var$commentre, '');
        return (0, $6fedb2016a78880b$export$801dd37ac183521b)(res, [
            ','
        ]).map((v)=>$b499486c7f02abe7$var$trim(v));
    }
    /**
   * Parse declaration.
   */ function declaration() {
        const pos = position();
        // prop
        const propMatch = /^(\*?[-#/*\\\w]+(\[[0-9a-z_-]+\])?)\s*/.exec(css);
        if (!propMatch) return;
        processMatch(propMatch);
        const propValue = $b499486c7f02abe7$var$trim(propMatch[0]);
        // :
        const sepratotorMatch = /^:\s*/.exec(css);
        if (!sepratotorMatch) return error("property missing ':'");
        processMatch(sepratotorMatch);
        // val
        let value = '';
        const endValuePosition = (0, $6fedb2016a78880b$export$b8d42a8583d2e477)(css, [
            ';',
            '}'
        ]);
        if (endValuePosition !== -1) {
            value = css.substring(0, endValuePosition);
            const fakeMatch = [
                value
            ];
            processMatch(fakeMatch);
            value = $b499486c7f02abe7$var$trim(value).replace($b499486c7f02abe7$var$commentre, '');
        }
        const ret = pos({
            type: (0, $d103407e81c97042$export$9be5dd6e61d5d73a).declaration,
            property: propValue.replace($b499486c7f02abe7$var$commentre, ''),
            value: value
        });
        // ;
        const endMatch = /^[;\s]*/.exec(css);
        if (endMatch) processMatch(endMatch);
        return ret;
    }
    /**
   * Parse declarations.
   */ function declarations() {
        const decls = [];
        if (!open()) return error("missing '{'");
        comments(decls);
        // declarations
        let decl;
        while(decl = declaration())if (decl) {
            decls.push(decl);
            comments(decls);
        }
        if (!close()) return error("missing '}'");
        return decls;
    }
    /**
   * Parse keyframe.
   */ function keyframe() {
        let m;
        const vals = [];
        const pos = position();
        while(m = /^((\d+\.\d+|\.\d+|\d+)%?|[a-z]+)\s*/.exec(css)){
            const res = processMatch(m);
            vals.push(res[1]);
            const spacesMatch = /^,\s*/.exec(css);
            if (spacesMatch) processMatch(spacesMatch);
        }
        if (!vals.length) return;
        return pos({
            type: (0, $d103407e81c97042$export$9be5dd6e61d5d73a).keyframe,
            values: vals,
            declarations: declarations() || []
        });
    }
    /**
   * Parse keyframes.
   */ function atkeyframes() {
        const pos = position();
        const m1 = /^@([-\w]+)?keyframes\s*/.exec(css);
        if (!m1) return;
        const vendor = processMatch(m1)[1];
        // identifier
        const m2 = /^([-\w]+)\s*/.exec(css);
        if (!m2) return error('@keyframes missing name');
        const name = processMatch(m2)[1];
        if (!open()) return error("@keyframes missing '{'");
        let frame;
        let frames = comments();
        while(frame = keyframe()){
            frames.push(frame);
            frames = frames.concat(comments());
        }
        if (!close()) return error("@keyframes missing '}'");
        return pos({
            type: (0, $d103407e81c97042$export$9be5dd6e61d5d73a).keyframes,
            name: name,
            vendor: vendor,
            keyframes: frames
        });
    }
    /**
   * Parse supports.
   */ function atsupports() {
        const pos = position();
        const m = /^@supports *([^{]+)/.exec(css);
        if (!m) return;
        const supports = $b499486c7f02abe7$var$trim(processMatch(m)[1]);
        if (!open()) return error("@supports missing '{'");
        const style = comments().concat(rules());
        if (!close()) return error("@supports missing '}'");
        return pos({
            type: (0, $d103407e81c97042$export$9be5dd6e61d5d73a).supports,
            supports: supports,
            rules: style
        });
    }
    /**
   * Parse host.
   */ function athost() {
        const pos = position();
        const m = /^@host\s*/.exec(css);
        if (!m) return;
        processMatch(m);
        if (!open()) return error("@host missing '{'");
        const style = comments().concat(rules());
        if (!close()) return error("@host missing '}'");
        return pos({
            type: (0, $d103407e81c97042$export$9be5dd6e61d5d73a).host,
            rules: style
        });
    }
    /**
   * Parse container.
   */ function atcontainer() {
        const pos = position();
        const m = /^@container *([^{]+)/.exec(css);
        if (!m) return;
        const container = $b499486c7f02abe7$var$trim(processMatch(m)[1]);
        if (!open()) return error("@container missing '{'");
        const style = comments().concat(rules());
        if (!close()) return error("@container missing '}'");
        return pos({
            type: (0, $d103407e81c97042$export$9be5dd6e61d5d73a).container,
            container: container,
            rules: style
        });
    }
    /**
   * Parse container.
   */ function atlayer() {
        const pos = position();
        const m = /^@layer *([^{;@]+)/.exec(css);
        if (!m) return;
        const layer = $b499486c7f02abe7$var$trim(processMatch(m)[1]);
        if (!open()) {
            const m2 = /^[;\s]*/.exec(css);
            if (m2) processMatch(m2);
            return pos({
                type: (0, $d103407e81c97042$export$9be5dd6e61d5d73a).layer,
                layer: layer
            });
        }
        const style = comments().concat(rules());
        if (!close()) return error("@layer missing '}'");
        return pos({
            type: (0, $d103407e81c97042$export$9be5dd6e61d5d73a).layer,
            layer: layer,
            rules: style
        });
    }
    /**
   * Parse media.
   */ function atmedia() {
        const pos = position();
        const m = /^@media *([^{]+)/.exec(css);
        if (!m) return;
        const media = $b499486c7f02abe7$var$trim(processMatch(m)[1]);
        if (!open()) return error("@media missing '{'");
        const style = comments().concat(rules());
        if (!close()) return error("@media missing '}'");
        return pos({
            type: (0, $d103407e81c97042$export$9be5dd6e61d5d73a).media,
            media: media,
            rules: style
        });
    }
    /**
   * Parse custom-media.
   */ function atcustommedia() {
        const pos = position();
        const m = /^@custom-media\s+(--\S+)\s+([^{;\s][^{;]*);/.exec(css);
        if (!m) return;
        const res = processMatch(m);
        return pos({
            type: (0, $d103407e81c97042$export$9be5dd6e61d5d73a).customMedia,
            name: $b499486c7f02abe7$var$trim(res[1]),
            media: $b499486c7f02abe7$var$trim(res[2])
        });
    }
    /**
   * Parse paged media.
   */ function atpage() {
        const pos = position();
        const m = /^@page */.exec(css);
        if (!m) return;
        processMatch(m);
        const sel = selector() || [];
        if (!open()) return error("@page missing '{'");
        let decls = comments();
        // declarations
        let decl;
        while(decl = declaration()){
            decls.push(decl);
            decls = decls.concat(comments());
        }
        if (!close()) return error("@page missing '}'");
        return pos({
            type: (0, $d103407e81c97042$export$9be5dd6e61d5d73a).page,
            selectors: sel,
            declarations: decls
        });
    }
    /**
   * Parse document.
   */ function atdocument() {
        const pos = position();
        const m = /^@([-\w]+)?document *([^{]+)/.exec(css);
        if (!m) return;
        const res = processMatch(m);
        const vendor = $b499486c7f02abe7$var$trim(res[1]);
        const doc = $b499486c7f02abe7$var$trim(res[2]);
        if (!open()) return error("@document missing '{'");
        const style = comments().concat(rules());
        if (!close()) return error("@document missing '}'");
        return pos({
            type: (0, $d103407e81c97042$export$9be5dd6e61d5d73a).document,
            document: doc,
            vendor: vendor,
            rules: style
        });
    }
    /**
   * Parse font-face.
   */ function atfontface() {
        const pos = position();
        const m = /^@font-face\s*/.exec(css);
        if (!m) return;
        processMatch(m);
        if (!open()) return error("@font-face missing '{'");
        let decls = comments();
        // declarations
        let decl;
        while(decl = declaration()){
            decls.push(decl);
            decls = decls.concat(comments());
        }
        if (!close()) return error("@font-face missing '}'");
        return pos({
            type: (0, $d103407e81c97042$export$9be5dd6e61d5d73a).fontFace,
            declarations: decls
        });
    }
    /**
   * Parse starting style.
   */ function atstartingstyle() {
        const pos = position();
        const m = /^@starting-style\s*/.exec(css);
        if (!m) return;
        processMatch(m);
        if (!open()) return error("@starting-style missing '{'");
        const style = comments().concat(rules());
        if (!close()) return error("@starting-style missing '}'");
        return pos({
            type: (0, $d103407e81c97042$export$9be5dd6e61d5d73a).startingStyle,
            rules: style
        });
    }
    /**
   * Parse import
   */ const atimport = _compileAtrule('import');
    /**
   * Parse charset
   */ const atcharset = _compileAtrule('charset');
    /**
   * Parse namespace
   */ const atnamespace = _compileAtrule('namespace');
    /**
   * Parse non-block at-rules
   */ function _compileAtrule(name) {
        const re = new RegExp('^@' + name + '\\s*((?::?[^;\'"]|"(?:\\\\"|[^"])*?"|\'(?:\\\\\'|[^\'])*?\')+)(?:;|$)');
        // ^@import\s*([^;"']|("|')(?:\\\2|.)*?\2)+(;|$)
        return function() {
            const pos = position();
            const m = re.exec(css);
            if (!m) return;
            const res = processMatch(m);
            const ret = {
                type: name
            };
            ret[name] = res[1].trim();
            return pos(ret);
        };
    }
    /**
   * Parse at rule.
   */ function atrule() {
        if (css[0] !== '@') return;
        return atkeyframes() || atmedia() || atcustommedia() || atsupports() || atimport() || atcharset() || atnamespace() || atdocument() || atpage() || athost() || atfontface() || atcontainer() || atstartingstyle() || atlayer();
    }
    /**
   * Parse rule.
   */ function rule() {
        const pos = position();
        const sel = selector();
        if (!sel) return error('selector missing');
        comments();
        return pos({
            type: (0, $d103407e81c97042$export$9be5dd6e61d5d73a).rule,
            selectors: sel,
            declarations: declarations() || []
        });
    }
    return $b499486c7f02abe7$var$addParent(stylesheet());
};
/**
 * Trim `str`.
 */ function $b499486c7f02abe7$var$trim(str) {
    return str ? str.trim() : '';
}
/**
 * Adds non-enumerable parent node reference to each node.
 */ function $b499486c7f02abe7$var$addParent(obj, parent) {
    const isNode = obj && typeof obj.type === 'string';
    const childParent = isNode ? obj : parent;
    for(const k in obj){
        const value = obj[k];
        if (Array.isArray(value)) value.forEach((v)=>{
            $b499486c7f02abe7$var$addParent(v, childParent);
        });
        else if (value && typeof value === 'object') $b499486c7f02abe7$var$addParent(value, childParent);
    }
    if (isNode) Object.defineProperty(obj, 'parent', {
        configurable: true,
        writable: true,
        enumerable: false,
        value: parent || null
    });
    return obj;
}
var $b499486c7f02abe7$export$2e2bcd8739ae039 = $b499486c7f02abe7$export$98e6a39c04603d36;



class $24dc7e49cb76910e$var$Compiler {
    constructor(options){
        this.level = 0;
        this.indentation = '  ';
        this.compress = false;
        if (typeof options?.indent === 'string') this.indentation = options?.indent;
        if (options?.compress) this.compress = true;
    }
    // We disable no-unused-vars for _position. We keep position for potential reintroduction of source-map
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    emit(str, _position) {
        return str;
    }
    /**
   * Increase, decrease or return current indentation.
   */ indent(level) {
        this.level = this.level || 1;
        if (level) {
            this.level += level;
            return '';
        }
        return Array(this.level).join(this.indentation);
    }
    visit(node) {
        switch(node.type){
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).stylesheet:
                return this.stylesheet(node);
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).rule:
                return this.rule(node);
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).declaration:
                return this.declaration(node);
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).comment:
                return this.comment(node);
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).container:
                return this.container(node);
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).charset:
                return this.charset(node);
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).document:
                return this.document(node);
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).customMedia:
                return this.customMedia(node);
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).fontFace:
                return this.fontFace(node);
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).host:
                return this.host(node);
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).import:
                return this.import(node);
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).keyframes:
                return this.keyframes(node);
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).keyframe:
                return this.keyframe(node);
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).layer:
                return this.layer(node);
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).media:
                return this.media(node);
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).namespace:
                return this.namespace(node);
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).page:
                return this.page(node);
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).startingStyle:
                return this.startingStyle(node);
            case (0, $d103407e81c97042$export$9be5dd6e61d5d73a).supports:
                return this.supports(node);
        }
    }
    mapVisit(nodes, delim) {
        let buf = '';
        delim = delim || '';
        for(let i = 0, length = nodes.length; i < length; i++){
            buf += this.visit(nodes[i]);
            if (delim && i < length - 1) buf += this.emit(delim);
        }
        return buf;
    }
    compile(node) {
        if (this.compress) return node.stylesheet.rules.map(this.visit, this).join('');
        return this.stylesheet(node);
    }
    /**
   * Visit stylesheet node.
   */ stylesheet(node) {
        return this.mapVisit(node.stylesheet.rules, '\n\n');
    }
    /**
   * Visit comment node.
   */ comment(node) {
        if (this.compress) return this.emit('', node.position);
        return this.emit(this.indent() + '/*' + node.comment + '*/', node.position);
    }
    /**
   * Visit container node.
   */ container(node) {
        if (this.compress) return this.emit('@container ' + node.container, node.position) + this.emit('{') + this.mapVisit(node.rules) + this.emit('}');
        return this.emit(this.indent() + '@container ' + node.container, node.position) + this.emit(' {\n' + this.indent(1)) + this.mapVisit(node.rules, '\n\n') + this.emit('\n' + this.indent(-1) + this.indent() + '}');
    }
    /**
   * Visit container node.
   */ layer(node) {
        if (this.compress) return this.emit('@layer ' + node.layer, node.position) + (node.rules ? this.emit('{') + this.mapVisit(node.rules) + this.emit('}') : ';');
        return this.emit(this.indent() + '@layer ' + node.layer, node.position) + (node.rules ? this.emit(' {\n' + this.indent(1)) + this.mapVisit(node.rules, '\n\n') + this.emit('\n' + this.indent(-1) + this.indent() + '}') : ';');
    }
    /**
   * Visit import node.
   */ import(node) {
        return this.emit('@import ' + node.import + ';', node.position);
    }
    /**
   * Visit media node.
   */ media(node) {
        if (this.compress) return this.emit('@media ' + node.media, node.position) + this.emit('{') + this.mapVisit(node.rules) + this.emit('}');
        return this.emit(this.indent() + '@media ' + node.media, node.position) + this.emit(' {\n' + this.indent(1)) + this.mapVisit(node.rules, '\n\n') + this.emit('\n' + this.indent(-1) + this.indent() + '}');
    }
    /**
   * Visit document node.
   */ document(node) {
        const doc = '@' + (node.vendor || '') + 'document ' + node.document;
        if (this.compress) return this.emit(doc, node.position) + this.emit('{') + this.mapVisit(node.rules) + this.emit('}');
        return this.emit(doc, node.position) + this.emit("  {\n" + this.indent(1)) + this.mapVisit(node.rules, '\n\n') + this.emit(this.indent(-1) + '\n}');
    }
    /**
   * Visit charset node.
   */ charset(node) {
        return this.emit('@charset ' + node.charset + ';', node.position);
    }
    /**
   * Visit namespace node.
   */ namespace(node) {
        return this.emit('@namespace ' + node.namespace + ';', node.position);
    }
    /**
   * Visit container node.
   */ startingStyle(node) {
        if (this.compress) return this.emit('@starting-style', node.position) + this.emit('{') + this.mapVisit(node.rules) + this.emit('}');
        return this.emit(this.indent() + '@starting-style', node.position) + this.emit(' {\n' + this.indent(1)) + this.mapVisit(node.rules, '\n\n') + this.emit('\n' + this.indent(-1) + this.indent() + '}');
    }
    /**
   * Visit supports node.
   */ supports(node) {
        if (this.compress) return this.emit('@supports ' + node.supports, node.position) + this.emit('{') + this.mapVisit(node.rules) + this.emit('}');
        return this.emit(this.indent() + '@supports ' + node.supports, node.position) + this.emit(' {\n' + this.indent(1)) + this.mapVisit(node.rules, '\n\n') + this.emit('\n' + this.indent(-1) + this.indent() + '}');
    }
    /**
   * Visit keyframes node.
   */ keyframes(node) {
        if (this.compress) return this.emit('@' + (node.vendor || '') + 'keyframes ' + node.name, node.position) + this.emit('{') + this.mapVisit(node.keyframes) + this.emit('}');
        return this.emit('@' + (node.vendor || '') + 'keyframes ' + node.name, node.position) + this.emit(' {\n' + this.indent(1)) + this.mapVisit(node.keyframes, '\n') + this.emit(this.indent(-1) + '}');
    }
    /**
   * Visit keyframe node.
   */ keyframe(node) {
        const decls = node.declarations;
        if (this.compress) return this.emit(node.values.join(','), node.position) + this.emit('{') + this.mapVisit(decls) + this.emit('}');
        return this.emit(this.indent()) + this.emit(node.values.join(', '), node.position) + this.emit(' {\n' + this.indent(1)) + this.mapVisit(decls, '\n') + this.emit(this.indent(-1) + '\n' + this.indent() + '}\n');
    }
    /**
   * Visit page node.
   */ page(node) {
        if (this.compress) {
            const sel = node.selectors.length ? node.selectors.join(', ') : '';
            return this.emit('@page ' + sel, node.position) + this.emit('{') + this.mapVisit(node.declarations) + this.emit('}');
        }
        const sel = node.selectors.length ? node.selectors.join(', ') + ' ' : '';
        return this.emit('@page ' + sel, node.position) + this.emit('{\n') + this.emit(this.indent(1)) + this.mapVisit(node.declarations, '\n') + this.emit(this.indent(-1)) + this.emit('\n}');
    }
    /**
   * Visit font-face node.
   */ fontFace(node) {
        if (this.compress) return this.emit('@font-face', node.position) + this.emit('{') + this.mapVisit(node.declarations) + this.emit('}');
        return this.emit('@font-face ', node.position) + this.emit('{\n') + this.emit(this.indent(1)) + this.mapVisit(node.declarations, '\n') + this.emit(this.indent(-1)) + this.emit('\n}');
    }
    /**
   * Visit host node.
   */ host(node) {
        if (this.compress) return this.emit('@host', node.position) + this.emit('{') + this.mapVisit(node.rules) + this.emit('}');
        return this.emit('@host', node.position) + this.emit(' {\n' + this.indent(1)) + this.mapVisit(node.rules, '\n\n') + this.emit(this.indent(-1) + '\n}');
    }
    /**
   * Visit custom-media node.
   */ customMedia(node) {
        return this.emit('@custom-media ' + node.name + ' ' + node.media + ';', node.position);
    }
    /**
   * Visit rule node.
   */ rule(node) {
        const decls = node.declarations;
        if (!decls.length) return '';
        if (this.compress) return this.emit(node.selectors.join(','), node.position) + this.emit('{') + this.mapVisit(decls) + this.emit('}');
        const indent = this.indent();
        return this.emit(node.selectors.map((s)=>{
            return indent + s;
        }).join(',\n'), node.position) + this.emit(' {\n') + this.emit(this.indent(1)) + this.mapVisit(decls, '\n') + this.emit(this.indent(-1)) + this.emit('\n' + this.indent() + '}');
    }
    /**
   * Visit declaration node.
   */ declaration(node) {
        if (this.compress) return this.emit(node.property + ':' + node.value, node.position) + this.emit(';');
        if (node.property === 'grid-template-areas') return this.emit(this.indent()) + this.emit(node.property + ': ' + node.value.split('\n').join('\n'.padEnd(22) + this.indent()), node.position) + this.emit(';');
        return this.emit(this.indent()) + this.emit(node.property + ': ' + node.value, node.position) + this.emit(';');
    }
}
var $24dc7e49cb76910e$export$2e2bcd8739ae039 = $24dc7e49cb76910e$var$Compiler;


var $fd680ce0c35731f5$export$2e2bcd8739ae039 = (node, options)=>{
    const compiler = new (0, $24dc7e49cb76910e$export$2e2bcd8739ae039)(options || {});
    return compiler.compile(node);
};





const $882b6d93070905b3$export$98e6a39c04603d36 = (0, $b499486c7f02abe7$export$2e2bcd8739ae039);
const $882b6d93070905b3$export$fac44ee5b035f737 = (0, $fd680ce0c35731f5$export$2e2bcd8739ae039);
var $882b6d93070905b3$export$2e2bcd8739ae039 = {
    parse: $882b6d93070905b3$export$98e6a39c04603d36,
    stringify: $882b6d93070905b3$export$fac44ee5b035f737
};
$parcel$exportWildcard(module.exports, $d103407e81c97042$exports);
$parcel$exportWildcard(module.exports, $cb508b9219b02820$exports);
$parcel$exportWildcard(module.exports, $4bafb28828007b46$exports);


//# sourceMappingURL=index.cjs.map
