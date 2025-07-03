import {
  __commonJS,
  __toESM,
  extractEventHiddenProperties
} from "./chunk-465TF3XA.mjs";

// node_modules/has-symbols/shams.js
var require_shams = __commonJS({
  "node_modules/has-symbols/shams.js"(exports, module) {
    "use strict";
    module.exports = function hasSymbols() {
      if (typeof Symbol !== "function" || typeof Object.getOwnPropertySymbols !== "function") {
        return false;
      }
      if (typeof Symbol.iterator === "symbol") {
        return true;
      }
      var obj = {};
      var sym = Symbol("test");
      var symObj = Object(sym);
      if (typeof sym === "string") {
        return false;
      }
      if (Object.prototype.toString.call(sym) !== "[object Symbol]") {
        return false;
      }
      if (Object.prototype.toString.call(symObj) !== "[object Symbol]") {
        return false;
      }
      var symVal = 42;
      obj[sym] = symVal;
      for (sym in obj) {
        return false;
      }
      if (typeof Object.keys === "function" && Object.keys(obj).length !== 0) {
        return false;
      }
      if (typeof Object.getOwnPropertyNames === "function" && Object.getOwnPropertyNames(obj).length !== 0) {
        return false;
      }
      var syms = Object.getOwnPropertySymbols(obj);
      if (syms.length !== 1 || syms[0] !== sym) {
        return false;
      }
      if (!Object.prototype.propertyIsEnumerable.call(obj, sym)) {
        return false;
      }
      if (typeof Object.getOwnPropertyDescriptor === "function") {
        var descriptor = Object.getOwnPropertyDescriptor(obj, sym);
        if (descriptor.value !== symVal || descriptor.enumerable !== true) {
          return false;
        }
      }
      return true;
    };
  }
});

// node_modules/has-symbols/index.js
var require_has_symbols = __commonJS({
  "node_modules/has-symbols/index.js"(exports, module) {
    "use strict";
    var origSymbol = typeof Symbol !== "undefined" && Symbol;
    var hasSymbolSham = require_shams();
    module.exports = function hasNativeSymbols() {
      if (typeof origSymbol !== "function") {
        return false;
      }
      if (typeof Symbol !== "function") {
        return false;
      }
      if (typeof origSymbol("foo") !== "symbol") {
        return false;
      }
      if (typeof Symbol("bar") !== "symbol") {
        return false;
      }
      return hasSymbolSham();
    };
  }
});

// node_modules/function-bind/implementation.js
var require_implementation = __commonJS({
  "node_modules/function-bind/implementation.js"(exports, module) {
    "use strict";
    var ERROR_MESSAGE = "Function.prototype.bind called on incompatible ";
    var slice = Array.prototype.slice;
    var toStr = Object.prototype.toString;
    var funcType = "[object Function]";
    module.exports = function bind(that) {
      var target = this;
      if (typeof target !== "function" || toStr.call(target) !== funcType) {
        throw new TypeError(ERROR_MESSAGE + target);
      }
      var args2 = slice.call(arguments, 1);
      var bound;
      var binder = function() {
        if (this instanceof bound) {
          var result2 = target.apply(
            this,
            args2.concat(slice.call(arguments))
          );
          if (Object(result2) === result2) {
            return result2;
          }
          return this;
        } else {
          return target.apply(
            that,
            args2.concat(slice.call(arguments))
          );
        }
      };
      var boundLength = Math.max(0, target.length - args2.length);
      var boundArgs = [];
      for (var i = 0; i < boundLength; i++) {
        boundArgs.push("$" + i);
      }
      bound = Function("binder", "return function (" + boundArgs.join(",") + "){ return binder.apply(this,arguments); }")(binder);
      if (target.prototype) {
        var Empty = function Empty2() {
        };
        Empty.prototype = target.prototype;
        bound.prototype = new Empty();
        Empty.prototype = null;
      }
      return bound;
    };
  }
});

// node_modules/function-bind/index.js
var require_function_bind = __commonJS({
  "node_modules/function-bind/index.js"(exports, module) {
    "use strict";
    var implementation = require_implementation();
    module.exports = Function.prototype.bind || implementation;
  }
});

// node_modules/has/src/index.js
var require_src = __commonJS({
  "node_modules/has/src/index.js"(exports, module) {
    "use strict";
    var bind = require_function_bind();
    module.exports = bind.call(Function.call, Object.prototype.hasOwnProperty);
  }
});

// node_modules/get-intrinsic/index.js
var require_get_intrinsic = __commonJS({
  "node_modules/get-intrinsic/index.js"(exports, module) {
    "use strict";
    var undefined2;
    var $SyntaxError = SyntaxError;
    var $Function = Function;
    var $TypeError = TypeError;
    var getEvalledConstructor = function(expressionSyntax) {
      try {
        return $Function('"use strict"; return (' + expressionSyntax + ").constructor;")();
      } catch (e) {
      }
    };
    var $gOPD = Object.getOwnPropertyDescriptor;
    if ($gOPD) {
      try {
        $gOPD({}, "");
      } catch (e) {
        $gOPD = null;
      }
    }
    var throwTypeError = function() {
      throw new $TypeError();
    };
    var ThrowTypeError = $gOPD ? function() {
      try {
        arguments.callee;
        return throwTypeError;
      } catch (calleeThrows) {
        try {
          return $gOPD(arguments, "callee").get;
        } catch (gOPDthrows) {
          return throwTypeError;
        }
      }
    }() : throwTypeError;
    var hasSymbols = require_has_symbols()();
    var getProto = Object.getPrototypeOf || function(x) {
      return x.__proto__;
    };
    var needsEval = {};
    var TypedArray = typeof Uint8Array === "undefined" ? undefined2 : getProto(Uint8Array);
    var INTRINSICS = {
      "%AggregateError%": typeof AggregateError === "undefined" ? undefined2 : AggregateError,
      "%Array%": Array,
      "%ArrayBuffer%": typeof ArrayBuffer === "undefined" ? undefined2 : ArrayBuffer,
      "%ArrayIteratorPrototype%": hasSymbols ? getProto([][Symbol.iterator]()) : undefined2,
      "%AsyncFromSyncIteratorPrototype%": undefined2,
      "%AsyncFunction%": needsEval,
      "%AsyncGenerator%": needsEval,
      "%AsyncGeneratorFunction%": needsEval,
      "%AsyncIteratorPrototype%": needsEval,
      "%Atomics%": typeof Atomics === "undefined" ? undefined2 : Atomics,
      "%BigInt%": typeof BigInt === "undefined" ? undefined2 : BigInt,
      "%Boolean%": Boolean,
      "%DataView%": typeof DataView === "undefined" ? undefined2 : DataView,
      "%Date%": Date,
      "%decodeURI%": decodeURI,
      "%decodeURIComponent%": decodeURIComponent,
      "%encodeURI%": encodeURI,
      "%encodeURIComponent%": encodeURIComponent,
      "%Error%": Error,
      "%eval%": eval,
      "%EvalError%": EvalError,
      "%Float32Array%": typeof Float32Array === "undefined" ? undefined2 : Float32Array,
      "%Float64Array%": typeof Float64Array === "undefined" ? undefined2 : Float64Array,
      "%FinalizationRegistry%": typeof FinalizationRegistry === "undefined" ? undefined2 : FinalizationRegistry,
      "%Function%": $Function,
      "%GeneratorFunction%": needsEval,
      "%Int8Array%": typeof Int8Array === "undefined" ? undefined2 : Int8Array,
      "%Int16Array%": typeof Int16Array === "undefined" ? undefined2 : Int16Array,
      "%Int32Array%": typeof Int32Array === "undefined" ? undefined2 : Int32Array,
      "%isFinite%": isFinite,
      "%isNaN%": isNaN,
      "%IteratorPrototype%": hasSymbols ? getProto(getProto([][Symbol.iterator]())) : undefined2,
      "%JSON%": typeof JSON === "object" ? JSON : undefined2,
      "%Map%": typeof Map === "undefined" ? undefined2 : Map,
      "%MapIteratorPrototype%": typeof Map === "undefined" || !hasSymbols ? undefined2 : getProto((/* @__PURE__ */ new Map())[Symbol.iterator]()),
      "%Math%": Math,
      "%Number%": Number,
      "%Object%": Object,
      "%parseFloat%": parseFloat,
      "%parseInt%": parseInt,
      "%Promise%": typeof Promise === "undefined" ? undefined2 : Promise,
      "%Proxy%": typeof Proxy === "undefined" ? undefined2 : Proxy,
      "%RangeError%": RangeError,
      "%ReferenceError%": ReferenceError,
      "%Reflect%": typeof Reflect === "undefined" ? undefined2 : Reflect,
      "%RegExp%": RegExp,
      "%Set%": typeof Set === "undefined" ? undefined2 : Set,
      "%SetIteratorPrototype%": typeof Set === "undefined" || !hasSymbols ? undefined2 : getProto((/* @__PURE__ */ new Set())[Symbol.iterator]()),
      "%SharedArrayBuffer%": typeof SharedArrayBuffer === "undefined" ? undefined2 : SharedArrayBuffer,
      "%String%": String,
      "%StringIteratorPrototype%": hasSymbols ? getProto(""[Symbol.iterator]()) : undefined2,
      "%Symbol%": hasSymbols ? Symbol : undefined2,
      "%SyntaxError%": $SyntaxError,
      "%ThrowTypeError%": ThrowTypeError,
      "%TypedArray%": TypedArray,
      "%TypeError%": $TypeError,
      "%Uint8Array%": typeof Uint8Array === "undefined" ? undefined2 : Uint8Array,
      "%Uint8ClampedArray%": typeof Uint8ClampedArray === "undefined" ? undefined2 : Uint8ClampedArray,
      "%Uint16Array%": typeof Uint16Array === "undefined" ? undefined2 : Uint16Array,
      "%Uint32Array%": typeof Uint32Array === "undefined" ? undefined2 : Uint32Array,
      "%URIError%": URIError,
      "%WeakMap%": typeof WeakMap === "undefined" ? undefined2 : WeakMap,
      "%WeakRef%": typeof WeakRef === "undefined" ? undefined2 : WeakRef,
      "%WeakSet%": typeof WeakSet === "undefined" ? undefined2 : WeakSet
    };
    var doEval = function doEval2(name2) {
      var value2;
      if (name2 === "%AsyncFunction%") {
        value2 = getEvalledConstructor("async function () {}");
      } else if (name2 === "%GeneratorFunction%") {
        value2 = getEvalledConstructor("function* () {}");
      } else if (name2 === "%AsyncGeneratorFunction%") {
        value2 = getEvalledConstructor("async function* () {}");
      } else if (name2 === "%AsyncGenerator%") {
        var fn = doEval2("%AsyncGeneratorFunction%");
        if (fn) {
          value2 = fn.prototype;
        }
      } else if (name2 === "%AsyncIteratorPrototype%") {
        var gen = doEval2("%AsyncGenerator%");
        if (gen) {
          value2 = getProto(gen.prototype);
        }
      }
      INTRINSICS[name2] = value2;
      return value2;
    };
    var LEGACY_ALIASES = {
      "%ArrayBufferPrototype%": ["ArrayBuffer", "prototype"],
      "%ArrayPrototype%": ["Array", "prototype"],
      "%ArrayProto_entries%": ["Array", "prototype", "entries"],
      "%ArrayProto_forEach%": ["Array", "prototype", "forEach"],
      "%ArrayProto_keys%": ["Array", "prototype", "keys"],
      "%ArrayProto_values%": ["Array", "prototype", "values"],
      "%AsyncFunctionPrototype%": ["AsyncFunction", "prototype"],
      "%AsyncGenerator%": ["AsyncGeneratorFunction", "prototype"],
      "%AsyncGeneratorPrototype%": ["AsyncGeneratorFunction", "prototype", "prototype"],
      "%BooleanPrototype%": ["Boolean", "prototype"],
      "%DataViewPrototype%": ["DataView", "prototype"],
      "%DatePrototype%": ["Date", "prototype"],
      "%ErrorPrototype%": ["Error", "prototype"],
      "%EvalErrorPrototype%": ["EvalError", "prototype"],
      "%Float32ArrayPrototype%": ["Float32Array", "prototype"],
      "%Float64ArrayPrototype%": ["Float64Array", "prototype"],
      "%FunctionPrototype%": ["Function", "prototype"],
      "%Generator%": ["GeneratorFunction", "prototype"],
      "%GeneratorPrototype%": ["GeneratorFunction", "prototype", "prototype"],
      "%Int8ArrayPrototype%": ["Int8Array", "prototype"],
      "%Int16ArrayPrototype%": ["Int16Array", "prototype"],
      "%Int32ArrayPrototype%": ["Int32Array", "prototype"],
      "%JSONParse%": ["JSON", "parse"],
      "%JSONStringify%": ["JSON", "stringify"],
      "%MapPrototype%": ["Map", "prototype"],
      "%NumberPrototype%": ["Number", "prototype"],
      "%ObjectPrototype%": ["Object", "prototype"],
      "%ObjProto_toString%": ["Object", "prototype", "toString"],
      "%ObjProto_valueOf%": ["Object", "prototype", "valueOf"],
      "%PromisePrototype%": ["Promise", "prototype"],
      "%PromiseProto_then%": ["Promise", "prototype", "then"],
      "%Promise_all%": ["Promise", "all"],
      "%Promise_reject%": ["Promise", "reject"],
      "%Promise_resolve%": ["Promise", "resolve"],
      "%RangeErrorPrototype%": ["RangeError", "prototype"],
      "%ReferenceErrorPrototype%": ["ReferenceError", "prototype"],
      "%RegExpPrototype%": ["RegExp", "prototype"],
      "%SetPrototype%": ["Set", "prototype"],
      "%SharedArrayBufferPrototype%": ["SharedArrayBuffer", "prototype"],
      "%StringPrototype%": ["String", "prototype"],
      "%SymbolPrototype%": ["Symbol", "prototype"],
      "%SyntaxErrorPrototype%": ["SyntaxError", "prototype"],
      "%TypedArrayPrototype%": ["TypedArray", "prototype"],
      "%TypeErrorPrototype%": ["TypeError", "prototype"],
      "%Uint8ArrayPrototype%": ["Uint8Array", "prototype"],
      "%Uint8ClampedArrayPrototype%": ["Uint8ClampedArray", "prototype"],
      "%Uint16ArrayPrototype%": ["Uint16Array", "prototype"],
      "%Uint32ArrayPrototype%": ["Uint32Array", "prototype"],
      "%URIErrorPrototype%": ["URIError", "prototype"],
      "%WeakMapPrototype%": ["WeakMap", "prototype"],
      "%WeakSetPrototype%": ["WeakSet", "prototype"]
    };
    var bind = require_function_bind();
    var hasOwn = require_src();
    var $concat = bind.call(Function.call, Array.prototype.concat);
    var $spliceApply = bind.call(Function.apply, Array.prototype.splice);
    var $replace = bind.call(Function.call, String.prototype.replace);
    var $strSlice = bind.call(Function.call, String.prototype.slice);
    var $exec = bind.call(Function.call, RegExp.prototype.exec);
    var rePropName2 = /[^%.[\]]+|\[(?:(-?\d+(?:\.\d+)?)|(["'])((?:(?!\2)[^\\]|\\.)*?)\2)\]|(?=(?:\.|\[\])(?:\.|\[\]|%$))/g;
    var reEscapeChar2 = /\\(\\)?/g;
    var stringToPath2 = function stringToPath3(string) {
      var first = $strSlice(string, 0, 1);
      var last = $strSlice(string, -1);
      if (first === "%" && last !== "%") {
        throw new $SyntaxError("invalid intrinsic syntax, expected closing `%`");
      } else if (last === "%" && first !== "%") {
        throw new $SyntaxError("invalid intrinsic syntax, expected opening `%`");
      }
      var result2 = [];
      $replace(string, rePropName2, function(match, number, quote, subString) {
        result2[result2.length] = quote ? $replace(subString, reEscapeChar2, "$1") : number || match;
      });
      return result2;
    };
    var getBaseIntrinsic = function getBaseIntrinsic2(name2, allowMissing) {
      var intrinsicName = name2;
      var alias;
      if (hasOwn(LEGACY_ALIASES, intrinsicName)) {
        alias = LEGACY_ALIASES[intrinsicName];
        intrinsicName = "%" + alias[0] + "%";
      }
      if (hasOwn(INTRINSICS, intrinsicName)) {
        var value2 = INTRINSICS[intrinsicName];
        if (value2 === needsEval) {
          value2 = doEval(intrinsicName);
        }
        if (typeof value2 === "undefined" && !allowMissing) {
          throw new $TypeError("intrinsic " + name2 + " exists, but is not available. Please file an issue!");
        }
        return {
          alias,
          name: intrinsicName,
          value: value2
        };
      }
      throw new $SyntaxError("intrinsic " + name2 + " does not exist!");
    };
    module.exports = function GetIntrinsic(name2, allowMissing) {
      if (typeof name2 !== "string" || name2.length === 0) {
        throw new $TypeError("intrinsic name must be a non-empty string");
      }
      if (arguments.length > 1 && typeof allowMissing !== "boolean") {
        throw new $TypeError('"allowMissing" argument must be a boolean');
      }
      if ($exec(/^%?[^%]*%?$/, name2) === null) {
        throw new $SyntaxError("`%` may not be present anywhere but at the beginning and end of the intrinsic name");
      }
      var parts = stringToPath2(name2);
      var intrinsicBaseName = parts.length > 0 ? parts[0] : "";
      var intrinsic = getBaseIntrinsic("%" + intrinsicBaseName + "%", allowMissing);
      var intrinsicRealName = intrinsic.name;
      var value2 = intrinsic.value;
      var skipFurtherCaching = false;
      var alias = intrinsic.alias;
      if (alias) {
        intrinsicBaseName = alias[0];
        $spliceApply(parts, $concat([0, 1], alias));
      }
      for (var i = 1, isOwn = true; i < parts.length; i += 1) {
        var part = parts[i];
        var first = $strSlice(part, 0, 1);
        var last = $strSlice(part, -1);
        if ((first === '"' || first === "'" || first === "`" || (last === '"' || last === "'" || last === "`")) && first !== last) {
          throw new $SyntaxError("property names with quotes must have matching quotes");
        }
        if (part === "constructor" || !isOwn) {
          skipFurtherCaching = true;
        }
        intrinsicBaseName += "." + part;
        intrinsicRealName = "%" + intrinsicBaseName + "%";
        if (hasOwn(INTRINSICS, intrinsicRealName)) {
          value2 = INTRINSICS[intrinsicRealName];
        } else if (value2 != null) {
          if (!(part in value2)) {
            if (!allowMissing) {
              throw new $TypeError("base intrinsic for " + name2 + " exists, but the property is not available.");
            }
            return void 0;
          }
          if ($gOPD && i + 1 >= parts.length) {
            var desc = $gOPD(value2, part);
            isOwn = !!desc;
            if (isOwn && "get" in desc && !("originalValue" in desc.get)) {
              value2 = desc.get;
            } else {
              value2 = value2[part];
            }
          } else {
            isOwn = hasOwn(value2, part);
            value2 = value2[part];
          }
          if (isOwn && !skipFurtherCaching) {
            INTRINSICS[intrinsicRealName] = value2;
          }
        }
      }
      return value2;
    };
  }
});

// node_modules/call-bind/index.js
var require_call_bind = __commonJS({
  "node_modules/call-bind/index.js"(exports, module) {
    "use strict";
    var bind = require_function_bind();
    var GetIntrinsic = require_get_intrinsic();
    var $apply = GetIntrinsic("%Function.prototype.apply%");
    var $call = GetIntrinsic("%Function.prototype.call%");
    var $reflectApply = GetIntrinsic("%Reflect.apply%", true) || bind.call($call, $apply);
    var $gOPD = GetIntrinsic("%Object.getOwnPropertyDescriptor%", true);
    var $defineProperty = GetIntrinsic("%Object.defineProperty%", true);
    var $max = GetIntrinsic("%Math.max%");
    if ($defineProperty) {
      try {
        $defineProperty({}, "a", { value: 1 });
      } catch (e) {
        $defineProperty = null;
      }
    }
    module.exports = function callBind(originalFunction) {
      var func = $reflectApply(bind, $call, arguments);
      if ($gOPD && $defineProperty) {
        var desc = $gOPD(func, "length");
        if (desc.configurable) {
          $defineProperty(
            func,
            "length",
            { value: 1 + $max(0, originalFunction.length - (arguments.length - 1)) }
          );
        }
      }
      return func;
    };
    var applyBind = function applyBind2() {
      return $reflectApply(bind, $apply, arguments);
    };
    if ($defineProperty) {
      $defineProperty(module.exports, "apply", { value: applyBind });
    } else {
      module.exports.apply = applyBind;
    }
  }
});

// node_modules/call-bind/callBound.js
var require_callBound = __commonJS({
  "node_modules/call-bind/callBound.js"(exports, module) {
    "use strict";
    var GetIntrinsic = require_get_intrinsic();
    var callBind = require_call_bind();
    var $indexOf = callBind(GetIntrinsic("String.prototype.indexOf"));
    module.exports = function callBoundIntrinsic(name2, allowMissing) {
      var intrinsic = GetIntrinsic(name2, !!allowMissing);
      if (typeof intrinsic === "function" && $indexOf(name2, ".prototype.") > -1) {
        return callBind(intrinsic);
      }
      return intrinsic;
    };
  }
});

// node_modules/has-tostringtag/shams.js
var require_shams2 = __commonJS({
  "node_modules/has-tostringtag/shams.js"(exports, module) {
    "use strict";
    var hasSymbols = require_shams();
    module.exports = function hasToStringTagShams() {
      return hasSymbols() && !!Symbol.toStringTag;
    };
  }
});

// node_modules/is-regex/index.js
var require_is_regex = __commonJS({
  "node_modules/is-regex/index.js"(exports, module) {
    "use strict";
    var callBound = require_callBound();
    var hasToStringTag = require_shams2()();
    var has;
    var $exec;
    var isRegexMarker;
    var badStringifier;
    if (hasToStringTag) {
      has = callBound("Object.prototype.hasOwnProperty");
      $exec = callBound("RegExp.prototype.exec");
      isRegexMarker = {};
      throwRegexMarker = function() {
        throw isRegexMarker;
      };
      badStringifier = {
        toString: throwRegexMarker,
        valueOf: throwRegexMarker
      };
      if (typeof Symbol.toPrimitive === "symbol") {
        badStringifier[Symbol.toPrimitive] = throwRegexMarker;
      }
    }
    var throwRegexMarker;
    var $toString = callBound("Object.prototype.toString");
    var gOPD = Object.getOwnPropertyDescriptor;
    var regexClass = "[object RegExp]";
    module.exports = hasToStringTag ? function isRegex(value2) {
      if (!value2 || typeof value2 !== "object") {
        return false;
      }
      var descriptor = gOPD(value2, "lastIndex");
      var hasLastIndexDataProperty = descriptor && has(descriptor, "value");
      if (!hasLastIndexDataProperty) {
        return false;
      }
      try {
        $exec(value2, badStringifier);
      } catch (e) {
        return e === isRegexMarker;
      }
    } : function isRegex(value2) {
      if (!value2 || typeof value2 !== "object" && typeof value2 !== "function") {
        return false;
      }
      return $toString(value2) === regexClass;
    };
  }
});

// node_modules/is-function/index.js
var require_is_function = __commonJS({
  "node_modules/is-function/index.js"(exports, module) {
    module.exports = isFunction3;
    var toString2 = Object.prototype.toString;
    function isFunction3(fn) {
      if (!fn) {
        return false;
      }
      var string = toString2.call(fn);
      return string === "[object Function]" || typeof fn === "function" && string !== "[object RegExp]" || typeof window !== "undefined" && (fn === window.setTimeout || fn === window.alert || fn === window.confirm || fn === window.prompt);
    }
  }
});

// node_modules/is-symbol/index.js
var require_is_symbol = __commonJS({
  "node_modules/is-symbol/index.js"(exports, module) {
    "use strict";
    var toStr = Object.prototype.toString;
    var hasSymbols = require_has_symbols()();
    if (hasSymbols) {
      symToStr = Symbol.prototype.toString;
      symStringRegex = /^Symbol\(.*\)$/;
      isSymbolObject = function isRealSymbolObject(value2) {
        if (typeof value2.valueOf() !== "symbol") {
          return false;
        }
        return symStringRegex.test(symToStr.call(value2));
      };
      module.exports = function isSymbol3(value2) {
        if (typeof value2 === "symbol") {
          return true;
        }
        if (toStr.call(value2) !== "[object Symbol]") {
          return false;
        }
        try {
          return isSymbolObject(value2);
        } catch (e) {
          return false;
        }
      };
    } else {
      module.exports = function isSymbol3(value2) {
        return false;
      };
    }
    var symToStr;
    var symStringRegex;
    var isSymbolObject;
  }
});

// src/index.ts
var import_is_regex = __toESM(require_is_regex());
var import_is_function = __toESM(require_is_function());
var import_is_symbol = __toESM(require_is_symbol());

// node_modules/isobject/index.js
function isObject(val) {
  return val != null && typeof val === "object" && Array.isArray(val) === false;
}

// node_modules/lodash-es/_freeGlobal.js
var freeGlobal = typeof global == "object" && global && global.Object === Object && global;
var freeGlobal_default = freeGlobal;

// node_modules/lodash-es/_root.js
var freeSelf = typeof self == "object" && self && self.Object === Object && self;
var root2 = freeGlobal_default || freeSelf || Function("return this")();
var root_default = root2;

// node_modules/lodash-es/_Symbol.js
var Symbol2 = root_default.Symbol;
var Symbol_default = Symbol2;

// node_modules/lodash-es/_getRawTag.js
var objectProto = Object.prototype;
var hasOwnProperty = objectProto.hasOwnProperty;
var nativeObjectToString = objectProto.toString;
var symToStringTag = Symbol_default ? Symbol_default.toStringTag : void 0;
function getRawTag(value2) {
  var isOwn = hasOwnProperty.call(value2, symToStringTag), tag = value2[symToStringTag];
  try {
    value2[symToStringTag] = void 0;
    var unmasked = true;
  } catch (e) {
  }
  var result2 = nativeObjectToString.call(value2);
  if (unmasked) {
    if (isOwn) {
      value2[symToStringTag] = tag;
    } else {
      delete value2[symToStringTag];
    }
  }
  return result2;
}
var getRawTag_default = getRawTag;

// node_modules/lodash-es/_objectToString.js
var objectProto2 = Object.prototype;
var nativeObjectToString2 = objectProto2.toString;
function objectToString(value2) {
  return nativeObjectToString2.call(value2);
}
var objectToString_default = objectToString;

// node_modules/lodash-es/_baseGetTag.js
var nullTag = "[object Null]";
var undefinedTag = "[object Undefined]";
var symToStringTag2 = Symbol_default ? Symbol_default.toStringTag : void 0;
function baseGetTag(value2) {
  if (value2 == null) {
    return value2 === void 0 ? undefinedTag : nullTag;
  }
  return symToStringTag2 && symToStringTag2 in Object(value2) ? getRawTag_default(value2) : objectToString_default(value2);
}
var baseGetTag_default = baseGetTag;

// node_modules/lodash-es/isObjectLike.js
function isObjectLike(value2) {
  return value2 != null && typeof value2 == "object";
}
var isObjectLike_default = isObjectLike;

// node_modules/lodash-es/isSymbol.js
var symbolTag = "[object Symbol]";
function isSymbol(value2) {
  return typeof value2 == "symbol" || isObjectLike_default(value2) && baseGetTag_default(value2) == symbolTag;
}
var isSymbol_default = isSymbol;

// node_modules/lodash-es/_arrayMap.js
function arrayMap(array, iteratee) {
  var index = -1, length = array == null ? 0 : array.length, result2 = Array(length);
  while (++index < length) {
    result2[index] = iteratee(array[index], index, array);
  }
  return result2;
}
var arrayMap_default = arrayMap;

// node_modules/lodash-es/isArray.js
var isArray = Array.isArray;
var isArray_default = isArray;

// node_modules/lodash-es/_baseToString.js
var INFINITY = 1 / 0;
var symbolProto = Symbol_default ? Symbol_default.prototype : void 0;
var symbolToString = symbolProto ? symbolProto.toString : void 0;
function baseToString(value2) {
  if (typeof value2 == "string") {
    return value2;
  }
  if (isArray_default(value2)) {
    return arrayMap_default(value2, baseToString) + "";
  }
  if (isSymbol_default(value2)) {
    return symbolToString ? symbolToString.call(value2) : "";
  }
  var result2 = value2 + "";
  return result2 == "0" && 1 / value2 == -INFINITY ? "-0" : result2;
}
var baseToString_default = baseToString;

// node_modules/lodash-es/isObject.js
function isObject2(value2) {
  var type = typeof value2;
  return value2 != null && (type == "object" || type == "function");
}
var isObject_default = isObject2;

// node_modules/lodash-es/isFunction.js
var asyncTag = "[object AsyncFunction]";
var funcTag = "[object Function]";
var genTag = "[object GeneratorFunction]";
var proxyTag = "[object Proxy]";
function isFunction(value2) {
  if (!isObject_default(value2)) {
    return false;
  }
  var tag = baseGetTag_default(value2);
  return tag == funcTag || tag == genTag || tag == asyncTag || tag == proxyTag;
}
var isFunction_default = isFunction;

// node_modules/lodash-es/_coreJsData.js
var coreJsData = root_default["__core-js_shared__"];
var coreJsData_default = coreJsData;

// node_modules/lodash-es/_isMasked.js
var maskSrcKey = function() {
  var uid = /[^.]+$/.exec(coreJsData_default && coreJsData_default.keys && coreJsData_default.keys.IE_PROTO || "");
  return uid ? "Symbol(src)_1." + uid : "";
}();
function isMasked(func) {
  return !!maskSrcKey && maskSrcKey in func;
}
var isMasked_default = isMasked;

// node_modules/lodash-es/_toSource.js
var funcProto = Function.prototype;
var funcToString = funcProto.toString;
function toSource(func) {
  if (func != null) {
    try {
      return funcToString.call(func);
    } catch (e) {
    }
    try {
      return func + "";
    } catch (e) {
    }
  }
  return "";
}
var toSource_default = toSource;

// node_modules/lodash-es/_baseIsNative.js
var reRegExpChar = /[\\^$.*+?()[\]{}|]/g;
var reIsHostCtor = /^\[object .+?Constructor\]$/;
var funcProto2 = Function.prototype;
var objectProto3 = Object.prototype;
var funcToString2 = funcProto2.toString;
var hasOwnProperty2 = objectProto3.hasOwnProperty;
var reIsNative = RegExp(
  "^" + funcToString2.call(hasOwnProperty2).replace(reRegExpChar, "\\$&").replace(/hasOwnProperty|(function).*?(?=\\\()| for .+?(?=\\\])/g, "$1.*?") + "$"
);
function baseIsNative(value2) {
  if (!isObject_default(value2) || isMasked_default(value2)) {
    return false;
  }
  var pattern = isFunction_default(value2) ? reIsNative : reIsHostCtor;
  return pattern.test(toSource_default(value2));
}
var baseIsNative_default = baseIsNative;

// node_modules/lodash-es/_getValue.js
function getValue(object, key2) {
  return object == null ? void 0 : object[key2];
}
var getValue_default = getValue;

// node_modules/lodash-es/_getNative.js
function getNative(object, key2) {
  var value2 = getValue_default(object, key2);
  return baseIsNative_default(value2) ? value2 : void 0;
}
var getNative_default = getNative;

// node_modules/lodash-es/eq.js
function eq(value2, other) {
  return value2 === other || value2 !== value2 && other !== other;
}
var eq_default = eq;

// node_modules/lodash-es/_isKey.js
var reIsDeepProp = /\.|\[(?:[^[\]]*|(["'])(?:(?!\1)[^\\]|\\.)*?\1)\]/;
var reIsPlainProp = /^\w*$/;
function isKey(value2, object) {
  if (isArray_default(value2)) {
    return false;
  }
  var type = typeof value2;
  if (type == "number" || type == "symbol" || type == "boolean" || value2 == null || isSymbol_default(value2)) {
    return true;
  }
  return reIsPlainProp.test(value2) || !reIsDeepProp.test(value2) || object != null && value2 in Object(object);
}
var isKey_default = isKey;

// node_modules/lodash-es/_nativeCreate.js
var nativeCreate = getNative_default(Object, "create");
var nativeCreate_default = nativeCreate;

// node_modules/lodash-es/_hashClear.js
function hashClear() {
  this.__data__ = nativeCreate_default ? nativeCreate_default(null) : {};
  this.size = 0;
}
var hashClear_default = hashClear;

// node_modules/lodash-es/_hashDelete.js
function hashDelete(key2) {
  var result2 = this.has(key2) && delete this.__data__[key2];
  this.size -= result2 ? 1 : 0;
  return result2;
}
var hashDelete_default = hashDelete;

// node_modules/lodash-es/_hashGet.js
var HASH_UNDEFINED = "__lodash_hash_undefined__";
var objectProto4 = Object.prototype;
var hasOwnProperty3 = objectProto4.hasOwnProperty;
function hashGet(key2) {
  var data = this.__data__;
  if (nativeCreate_default) {
    var result2 = data[key2];
    return result2 === HASH_UNDEFINED ? void 0 : result2;
  }
  return hasOwnProperty3.call(data, key2) ? data[key2] : void 0;
}
var hashGet_default = hashGet;

// node_modules/lodash-es/_hashHas.js
var objectProto5 = Object.prototype;
var hasOwnProperty4 = objectProto5.hasOwnProperty;
function hashHas(key2) {
  var data = this.__data__;
  return nativeCreate_default ? data[key2] !== void 0 : hasOwnProperty4.call(data, key2);
}
var hashHas_default = hashHas;

// node_modules/lodash-es/_hashSet.js
var HASH_UNDEFINED2 = "__lodash_hash_undefined__";
function hashSet(key2, value2) {
  var data = this.__data__;
  this.size += this.has(key2) ? 0 : 1;
  data[key2] = nativeCreate_default && value2 === void 0 ? HASH_UNDEFINED2 : value2;
  return this;
}
var hashSet_default = hashSet;

// node_modules/lodash-es/_Hash.js
function Hash(entries) {
  var index = -1, length = entries == null ? 0 : entries.length;
  this.clear();
  while (++index < length) {
    var entry = entries[index];
    this.set(entry[0], entry[1]);
  }
}
Hash.prototype.clear = hashClear_default;
Hash.prototype["delete"] = hashDelete_default;
Hash.prototype.get = hashGet_default;
Hash.prototype.has = hashHas_default;
Hash.prototype.set = hashSet_default;
var Hash_default = Hash;

// node_modules/lodash-es/_listCacheClear.js
function listCacheClear() {
  this.__data__ = [];
  this.size = 0;
}
var listCacheClear_default = listCacheClear;

// node_modules/lodash-es/_assocIndexOf.js
function assocIndexOf(array, key2) {
  var length = array.length;
  while (length--) {
    if (eq_default(array[length][0], key2)) {
      return length;
    }
  }
  return -1;
}
var assocIndexOf_default = assocIndexOf;

// node_modules/lodash-es/_listCacheDelete.js
var arrayProto = Array.prototype;
var splice = arrayProto.splice;
function listCacheDelete(key2) {
  var data = this.__data__, index = assocIndexOf_default(data, key2);
  if (index < 0) {
    return false;
  }
  var lastIndex = data.length - 1;
  if (index == lastIndex) {
    data.pop();
  } else {
    splice.call(data, index, 1);
  }
  --this.size;
  return true;
}
var listCacheDelete_default = listCacheDelete;

// node_modules/lodash-es/_listCacheGet.js
function listCacheGet(key2) {
  var data = this.__data__, index = assocIndexOf_default(data, key2);
  return index < 0 ? void 0 : data[index][1];
}
var listCacheGet_default = listCacheGet;

// node_modules/lodash-es/_listCacheHas.js
function listCacheHas(key2) {
  return assocIndexOf_default(this.__data__, key2) > -1;
}
var listCacheHas_default = listCacheHas;

// node_modules/lodash-es/_listCacheSet.js
function listCacheSet(key2, value2) {
  var data = this.__data__, index = assocIndexOf_default(data, key2);
  if (index < 0) {
    ++this.size;
    data.push([key2, value2]);
  } else {
    data[index][1] = value2;
  }
  return this;
}
var listCacheSet_default = listCacheSet;

// node_modules/lodash-es/_ListCache.js
function ListCache(entries) {
  var index = -1, length = entries == null ? 0 : entries.length;
  this.clear();
  while (++index < length) {
    var entry = entries[index];
    this.set(entry[0], entry[1]);
  }
}
ListCache.prototype.clear = listCacheClear_default;
ListCache.prototype["delete"] = listCacheDelete_default;
ListCache.prototype.get = listCacheGet_default;
ListCache.prototype.has = listCacheHas_default;
ListCache.prototype.set = listCacheSet_default;
var ListCache_default = ListCache;

// node_modules/lodash-es/_Map.js
var Map2 = getNative_default(root_default, "Map");
var Map_default = Map2;

// node_modules/lodash-es/_mapCacheClear.js
function mapCacheClear() {
  this.size = 0;
  this.__data__ = {
    "hash": new Hash_default(),
    "map": new (Map_default || ListCache_default)(),
    "string": new Hash_default()
  };
}
var mapCacheClear_default = mapCacheClear;

// node_modules/lodash-es/_isKeyable.js
function isKeyable(value2) {
  var type = typeof value2;
  return type == "string" || type == "number" || type == "symbol" || type == "boolean" ? value2 !== "__proto__" : value2 === null;
}
var isKeyable_default = isKeyable;

// node_modules/lodash-es/_getMapData.js
function getMapData(map, key2) {
  var data = map.__data__;
  return isKeyable_default(key2) ? data[typeof key2 == "string" ? "string" : "hash"] : data.map;
}
var getMapData_default = getMapData;

// node_modules/lodash-es/_mapCacheDelete.js
function mapCacheDelete(key2) {
  var result2 = getMapData_default(this, key2)["delete"](key2);
  this.size -= result2 ? 1 : 0;
  return result2;
}
var mapCacheDelete_default = mapCacheDelete;

// node_modules/lodash-es/_mapCacheGet.js
function mapCacheGet(key2) {
  return getMapData_default(this, key2).get(key2);
}
var mapCacheGet_default = mapCacheGet;

// node_modules/lodash-es/_mapCacheHas.js
function mapCacheHas(key2) {
  return getMapData_default(this, key2).has(key2);
}
var mapCacheHas_default = mapCacheHas;

// node_modules/lodash-es/_mapCacheSet.js
function mapCacheSet(key2, value2) {
  var data = getMapData_default(this, key2), size = data.size;
  data.set(key2, value2);
  this.size += data.size == size ? 0 : 1;
  return this;
}
var mapCacheSet_default = mapCacheSet;

// node_modules/lodash-es/_MapCache.js
function MapCache(entries) {
  var index = -1, length = entries == null ? 0 : entries.length;
  this.clear();
  while (++index < length) {
    var entry = entries[index];
    this.set(entry[0], entry[1]);
  }
}
MapCache.prototype.clear = mapCacheClear_default;
MapCache.prototype["delete"] = mapCacheDelete_default;
MapCache.prototype.get = mapCacheGet_default;
MapCache.prototype.has = mapCacheHas_default;
MapCache.prototype.set = mapCacheSet_default;
var MapCache_default = MapCache;

// node_modules/lodash-es/memoize.js
var FUNC_ERROR_TEXT = "Expected a function";
function memoize(func, resolver) {
  if (typeof func != "function" || resolver != null && typeof resolver != "function") {
    throw new TypeError(FUNC_ERROR_TEXT);
  }
  var memoized = function() {
    var args2 = arguments, key2 = resolver ? resolver.apply(this, args2) : args2[0], cache = memoized.cache;
    if (cache.has(key2)) {
      return cache.get(key2);
    }
    var result2 = func.apply(this, args2);
    memoized.cache = cache.set(key2, result2) || cache;
    return result2;
  };
  memoized.cache = new (memoize.Cache || MapCache_default)();
  return memoized;
}
memoize.Cache = MapCache_default;
var memoize_default = memoize;

// node_modules/lodash-es/_memoizeCapped.js
var MAX_MEMOIZE_SIZE = 500;
function memoizeCapped(func) {
  var result2 = memoize_default(func, function(key2) {
    if (cache.size === MAX_MEMOIZE_SIZE) {
      cache.clear();
    }
    return key2;
  });
  var cache = result2.cache;
  return result2;
}
var memoizeCapped_default = memoizeCapped;

// node_modules/lodash-es/_stringToPath.js
var rePropName = /[^.[\]]+|\[(?:(-?\d+(?:\.\d+)?)|(["'])((?:(?!\2)[^\\]|\\.)*?)\2)\]|(?=(?:\.|\[\])(?:\.|\[\]|$))/g;
var reEscapeChar = /\\(\\)?/g;
var stringToPath = memoizeCapped_default(function(string) {
  var result2 = [];
  if (string.charCodeAt(0) === 46) {
    result2.push("");
  }
  string.replace(rePropName, function(match, number, quote, subString) {
    result2.push(quote ? subString.replace(reEscapeChar, "$1") : number || match);
  });
  return result2;
});
var stringToPath_default = stringToPath;

// node_modules/lodash-es/toString.js
function toString(value2) {
  return value2 == null ? "" : baseToString_default(value2);
}
var toString_default = toString;

// node_modules/lodash-es/_castPath.js
function castPath(value2, object) {
  if (isArray_default(value2)) {
    return value2;
  }
  return isKey_default(value2, object) ? [value2] : stringToPath_default(toString_default(value2));
}
var castPath_default = castPath;

// node_modules/lodash-es/_toKey.js
var INFINITY2 = 1 / 0;
function toKey(value2) {
  if (typeof value2 == "string" || isSymbol_default(value2)) {
    return value2;
  }
  var result2 = value2 + "";
  return result2 == "0" && 1 / value2 == -INFINITY2 ? "-0" : result2;
}
var toKey_default = toKey;

// node_modules/lodash-es/_baseGet.js
function baseGet(object, path) {
  path = castPath_default(path, object);
  var index = 0, length = path.length;
  while (object != null && index < length) {
    object = object[toKey_default(path[index++])];
  }
  return index && index == length ? object : void 0;
}
var baseGet_default = baseGet;

// node_modules/lodash-es/get.js
function get(object, path, defaultValue) {
  var result2 = object == null ? void 0 : baseGet_default(object, path);
  return result2 === void 0 ? defaultValue : result2;
}
var get_default = get;

// src/index.ts
import memoize2 from "memoizerific";
var isObject3 = isObject;
var removeCodeComments = (code) => {
  let inQuoteChar = null;
  let inBlockComment = false;
  let inLineComment = false;
  let inRegexLiteral = false;
  let newCode = "";
  if (code.indexOf("//") >= 0 || code.indexOf("/*") >= 0) {
    for (let i = 0; i < code.length; i += 1) {
      if (!inQuoteChar && !inBlockComment && !inLineComment && !inRegexLiteral) {
        if (code[i] === '"' || code[i] === "'" || code[i] === "`") {
          inQuoteChar = code[i];
        } else if (code[i] === "/" && code[i + 1] === "*") {
          inBlockComment = true;
        } else if (code[i] === "/" && code[i + 1] === "/") {
          inLineComment = true;
        } else if (code[i] === "/" && code[i + 1] !== "/") {
          inRegexLiteral = true;
        }
      } else {
        if (inQuoteChar && (code[i] === inQuoteChar && code[i - 1] !== "\\" || code[i] === "\n" && inQuoteChar !== "`")) {
          inQuoteChar = null;
        }
        if (inRegexLiteral && (code[i] === "/" && code[i - 1] !== "\\" || code[i] === "\n")) {
          inRegexLiteral = false;
        }
        if (inBlockComment && code[i - 1] === "/" && code[i - 2] === "*") {
          inBlockComment = false;
        }
        if (inLineComment && code[i] === "\n") {
          inLineComment = false;
        }
      }
      if (!inBlockComment && !inLineComment) {
        newCode += code[i];
      }
    }
  } else {
    newCode = code;
  }
  return newCode;
};
var cleanCode = memoize2(1e4)(
  (code) => removeCodeComments(code).replace(/\n\s*/g, "").trim()
);
var convertShorthandMethods = function convertShorthandMethods2(key2, stringified) {
  const fnHead = stringified.slice(0, stringified.indexOf("{"));
  const fnBody = stringified.slice(stringified.indexOf("{"));
  if (fnHead.includes("=>")) {
    return stringified;
  }
  if (fnHead.includes("function")) {
    return stringified;
  }
  let modifiedHead = fnHead;
  modifiedHead = modifiedHead.replace(key2, "function");
  return modifiedHead + fnBody;
};
var dateFormat = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{3})?Z$/;
var isJSON = (input) => input.match(/^[\[\{\"\}].*[\]\}\"]$/);
function convertUnconventionalData(data) {
  if (!isObject3(data)) {
    return data;
  }
  let result2 = data;
  let wasMutated = false;
  if (typeof Event !== "undefined" && data instanceof Event) {
    result2 = extractEventHiddenProperties(result2);
    wasMutated = true;
  }
  result2 = Object.keys(result2).reduce((acc, key2) => {
    try {
      if (result2[key2]) {
        result2[key2].toJSON;
      }
      acc[key2] = result2[key2];
    } catch (err) {
      wasMutated = true;
    }
    return acc;
  }, {});
  return wasMutated ? result2 : data;
}
var replacer = function replacer2(options2) {
  let objects;
  let map;
  let stack;
  let keys;
  return function replace(key2, value2) {
    try {
      if (key2 === "") {
        keys = [];
        objects = /* @__PURE__ */ new Map([[value2, "[]"]]);
        map = /* @__PURE__ */ new Map();
        stack = [];
        return value2;
      }
      const origin = map.get(this) || this;
      while (stack.length && origin !== stack[0]) {
        stack.shift();
        keys.pop();
      }
      if (typeof value2 === "boolean") {
        return value2;
      }
      if (value2 === void 0) {
        if (!options2.allowUndefined) {
          return void 0;
        }
        return "_undefined_";
      }
      if (value2 === null) {
        return null;
      }
      if (typeof value2 === "number") {
        if (value2 === -Infinity) {
          return "_-Infinity_";
        }
        if (value2 === Infinity) {
          return "_Infinity_";
        }
        if (Number.isNaN(value2)) {
          return "_NaN_";
        }
        return value2;
      }
      if (typeof value2 === "bigint") {
        return `_bigint_${value2.toString()}`;
      }
      if (typeof value2 === "string") {
        if (dateFormat.test(value2)) {
          if (!options2.allowDate) {
            return void 0;
          }
          return `_date_${value2}`;
        }
        return value2;
      }
      if ((0, import_is_regex.default)(value2)) {
        if (!options2.allowRegExp) {
          return void 0;
        }
        return `_regexp_${value2.flags}|${value2.source}`;
      }
      if ((0, import_is_function.default)(value2)) {
        if (!options2.allowFunction) {
          return void 0;
        }
        const { name: name2 } = value2;
        const stringified = value2.toString();
        if (!stringified.match(
          /(\[native code\]|WEBPACK_IMPORTED_MODULE|__webpack_exports__|__webpack_require__)/
        )) {
          return `_function_${name2}|${cleanCode(convertShorthandMethods(key2, stringified))}`;
        }
        return `_function_${name2}|${(() => {
        }).toString()}`;
      }
      if ((0, import_is_symbol.default)(value2)) {
        if (!options2.allowSymbol) {
          return void 0;
        }
        const globalRegistryKey = Symbol.keyFor(value2);
        if (globalRegistryKey !== void 0) {
          return `_gsymbol_${globalRegistryKey}`;
        }
        return `_symbol_${value2.toString().slice(7, -1)}`;
      }
      if (stack.length >= options2.maxDepth) {
        if (Array.isArray(value2)) {
          return `[Array(${value2.length})]`;
        }
        return "[Object]";
      }
      if (value2 === this) {
        return `_duplicate_${JSON.stringify(keys)}`;
      }
      if (value2 instanceof Error && options2.allowError) {
        return {
          __isConvertedError__: true,
          errorProperties: {
            ...value2.cause ? { cause: value2.cause } : {},
            ...value2,
            name: value2.name,
            message: value2.message,
            stack: value2.stack,
            "_constructor-name_": value2.constructor.name
          }
        };
      }
      if (value2.constructor && value2.constructor.name && value2.constructor.name !== "Object" && !Array.isArray(value2) && !options2.allowClass) {
        return void 0;
      }
      const found = objects.get(value2);
      if (!found) {
        const converted = Array.isArray(value2) ? value2 : convertUnconventionalData(value2);
        if (value2.constructor && value2.constructor.name && value2.constructor.name !== "Object" && !Array.isArray(value2) && options2.allowClass) {
          try {
            Object.assign(converted, { "_constructor-name_": value2.constructor.name });
          } catch (e) {
          }
        }
        keys.push(key2);
        stack.unshift(converted);
        objects.set(value2, JSON.stringify(keys));
        if (value2 !== converted) {
          map.set(value2, converted);
        }
        return converted;
      }
      return `_duplicate_${found}`;
    } catch (e) {
      return void 0;
    }
  };
};
var reviver2 = function reviver(options) {
  const refs = [];
  let root;
  return function revive(key, value) {
    if (key === "") {
      root = value;
      refs.forEach(({ target, container, replacement }) => {
        const replacementArr = isJSON(replacement) ? JSON.parse(replacement) : replacement.split(".");
        if (replacementArr.length === 0) {
          container[target] = root;
        } else {
          container[target] = get_default(root, replacementArr);
        }
      });
    }
    if (key === "_constructor-name_") {
      return value;
    }
    if (isObject3(value) && value.__isConvertedError__) {
      const { message, ...properties } = value.errorProperties;
      const error = new Error(message);
      Object.assign(error, properties);
      return error;
    }
    if (isObject3(value) && value["_constructor-name_"] && options.allowFunction) {
      const name2 = value["_constructor-name_"];
      if (name2 !== "Object") {
        const Fn = new Function(`return function ${name2.replace(/[^a-zA-Z0-9$_]+/g, "")}(){}`)();
        Object.setPrototypeOf(value, new Fn());
      }
      delete value["_constructor-name_"];
      return value;
    }
    if (typeof value === "string" && value.startsWith("_function_") && options.allowFunction) {
      const [, name, source] = value.match(/_function_([^|]*)\|(.*)/) || [];
      const sourceSanitized = source.replace(/[(\(\))|\\| |\]|`]*$/, "");
      if (!options.lazyEval) {
        return eval(`(${sourceSanitized})`);
      }
      const result = (...args) => {
        const f = eval(`(${sourceSanitized})`);
        return f(...args);
      };
      Object.defineProperty(result, "toString", {
        value: () => sourceSanitized
      });
      Object.defineProperty(result, "name", {
        value: name
      });
      return result;
    }
    if (typeof value === "string" && value.startsWith("_regexp_") && options.allowRegExp) {
      const [, flags, source2] = value.match(/_regexp_([^|]*)\|(.*)/) || [];
      return new RegExp(source2, flags);
    }
    if (typeof value === "string" && value.startsWith("_date_") && options.allowDate) {
      return new Date(value.replace("_date_", ""));
    }
    if (typeof value === "string" && value.startsWith("_duplicate_")) {
      refs.push({ target: key, container: this, replacement: value.replace(/^_duplicate_/, "") });
      return null;
    }
    if (typeof value === "string" && value.startsWith("_symbol_") && options.allowSymbol) {
      return Symbol(value.replace("_symbol_", ""));
    }
    if (typeof value === "string" && value.startsWith("_gsymbol_") && options.allowSymbol) {
      return Symbol.for(value.replace("_gsymbol_", ""));
    }
    if (typeof value === "string" && value === "_-Infinity_") {
      return -Infinity;
    }
    if (typeof value === "string" && value === "_Infinity_") {
      return Infinity;
    }
    if (typeof value === "string" && value === "_NaN_") {
      return NaN;
    }
    if (typeof value === "string" && value.startsWith("_bigint_") && typeof BigInt === "function") {
      return BigInt(value.replace("_bigint_", ""));
    }
    return value;
  };
};
var defaultOptions = {
  maxDepth: 10,
  space: void 0,
  allowFunction: true,
  allowRegExp: true,
  allowDate: true,
  allowClass: true,
  allowError: true,
  allowUndefined: true,
  allowSymbol: true,
  lazyEval: true
};
var stringify = (data, options2 = {}) => {
  const mergedOptions = { ...defaultOptions, ...options2 };
  return JSON.stringify(convertUnconventionalData(data), replacer(mergedOptions), options2.space);
};
var mutator = () => {
  const mutated = /* @__PURE__ */ new Map();
  return function mutateUndefined(value2) {
    if (isObject3(value2)) {
      Object.entries(value2).forEach(([k, v]) => {
        if (v === "_undefined_") {
          value2[k] = void 0;
        } else if (!mutated.get(v)) {
          mutated.set(v, true);
          mutateUndefined(v);
        }
      });
    }
    if (Array.isArray(value2)) {
      value2.forEach((v, index) => {
        if (v === "_undefined_") {
          mutated.set(v, true);
          value2[index] = void 0;
        } else if (!mutated.get(v)) {
          mutated.set(v, true);
          mutateUndefined(v);
        }
      });
    }
  };
};
var parse = (data, options2 = {}) => {
  const mergedOptions = { ...defaultOptions, ...options2 };
  const result2 = JSON.parse(data, reviver2(mergedOptions));
  mutator()(result2);
  return result2;
};
export {
  isJSON,
  parse,
  replacer,
  reviver2 as reviver,
  stringify
};
/*!
 * isobject <https://github.com/jonschlinkert/isobject>
 *
 * Copyright (c) 2014-2017, Jon Schlinkert.
 * Released under the MIT License.
 */
/**
 * @license
 * Lodash (Custom Build) <https://lodash.com/>
 * Build: `lodash modularize exports="es" -o ./`
 * Copyright OpenJS Foundation and other contributors <https://openjsf.org/>
 * Released under MIT license <https://lodash.com/license>
 * Based on Underscore.js 1.8.3 <http://underscorejs.org/LICENSE>
 * Copyright Jeremy Ashkenas, DocumentCloud and Investigative Reporters & Editors
 */
