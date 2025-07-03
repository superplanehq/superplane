var __create = Object.create;
var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __getProtoOf = Object.getPrototypeOf;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __require = /* @__PURE__ */ ((x) => typeof require !== "undefined" ? require : typeof Proxy !== "undefined" ? new Proxy(x, {
  get: (a, b) => (typeof require !== "undefined" ? require : a)[b]
}) : x)(function(x) {
  if (typeof require !== "undefined")
    return require.apply(this, arguments);
  throw new Error('Dynamic require of "' + x + '" is not supported');
});
var __commonJS = (cb, mod) => function __require2() {
  return mod || (0, cb[__getOwnPropNames(cb)[0]])((mod = { exports: {} }).exports, mod), mod.exports;
};
var __copyProps = (to, from, except, desc) => {
  if (from && typeof from === "object" || typeof from === "function") {
    for (let key of __getOwnPropNames(from))
      if (!__hasOwnProp.call(to, key) && key !== except)
        __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
  }
  return to;
};
var __toESM = (mod, isNodeMode, target) => (target = mod != null ? __create(__getProtoOf(mod)) : {}, __copyProps(
  // If the importer is in node compatibility mode or this is not an ESM
  // file that has been converted to a CommonJS file using a Babel-
  // compatible transform (i.e. "__esModule" has not been set), then set
  // "default" to the CommonJS "module.exports" for node compatibility.
  isNodeMode || !mod || !mod.__esModule ? __defProp(target, "default", { value: mod, enumerable: true }) : target,
  mod
));

// node_modules/.pnpm/defer-to-connect@2.0.1/node_modules/defer-to-connect/dist/source/index.js
var require_source = __commonJS({
  "node_modules/.pnpm/defer-to-connect@2.0.1/node_modules/defer-to-connect/dist/source/index.js"(exports, module) {
    "use strict";
    Object.defineProperty(exports, "__esModule", { value: true });
    function isTLSSocket(socket) {
      return socket.encrypted;
    }
    var deferToConnect2 = (socket, fn) => {
      let listeners;
      if (typeof fn === "function") {
        const connect = fn;
        listeners = { connect };
      } else {
        listeners = fn;
      }
      const hasConnectListener = typeof listeners.connect === "function";
      const hasSecureConnectListener = typeof listeners.secureConnect === "function";
      const hasCloseListener = typeof listeners.close === "function";
      const onConnect = () => {
        if (hasConnectListener) {
          listeners.connect();
        }
        if (isTLSSocket(socket) && hasSecureConnectListener) {
          if (socket.authorized) {
            listeners.secureConnect();
          } else if (!socket.authorizationError) {
            socket.once("secureConnect", listeners.secureConnect);
          }
        }
        if (hasCloseListener) {
          socket.once("close", listeners.close);
        }
      };
      if (socket.writable && !socket.connecting) {
        onConnect();
      } else if (socket.connecting) {
        socket.once("connect", onConnect);
      } else if (socket.destroyed && hasCloseListener) {
        listeners.close(socket._hadError);
      }
    };
    exports.default = deferToConnect2;
    module.exports = deferToConnect2;
    module.exports.default = deferToConnect2;
  }
});

// node_modules/.pnpm/get-stream@6.0.1/node_modules/get-stream/buffer-stream.js
var require_buffer_stream = __commonJS({
  "node_modules/.pnpm/get-stream@6.0.1/node_modules/get-stream/buffer-stream.js"(exports, module) {
    "use strict";
    var { PassThrough: PassThroughStream2 } = __require("stream");
    module.exports = (options) => {
      options = { ...options };
      const { array } = options;
      let { encoding } = options;
      const isBuffer = encoding === "buffer";
      let objectMode = false;
      if (array) {
        objectMode = !(encoding || isBuffer);
      } else {
        encoding = encoding || "utf8";
      }
      if (isBuffer) {
        encoding = null;
      }
      const stream2 = new PassThroughStream2({ objectMode });
      if (encoding) {
        stream2.setEncoding(encoding);
      }
      let length = 0;
      const chunks = [];
      stream2.on("data", (chunk) => {
        chunks.push(chunk);
        if (objectMode) {
          length = chunks.length;
        } else {
          length += chunk.length;
        }
      });
      stream2.getBufferedValue = () => {
        if (array) {
          return chunks;
        }
        return isBuffer ? Buffer.concat(chunks, length) : chunks.join("");
      };
      stream2.getBufferedLength = () => length;
      return stream2;
    };
  }
});

// node_modules/.pnpm/get-stream@6.0.1/node_modules/get-stream/index.js
var require_get_stream = __commonJS({
  "node_modules/.pnpm/get-stream@6.0.1/node_modules/get-stream/index.js"(exports, module) {
    "use strict";
    var { constants: BufferConstants } = __require("buffer");
    var stream2 = __require("stream");
    var { promisify: promisify4 } = __require("util");
    var bufferStream = require_buffer_stream();
    var streamPipelinePromisified = promisify4(stream2.pipeline);
    var MaxBufferError = class extends Error {
      constructor() {
        super("maxBuffer exceeded");
        this.name = "MaxBufferError";
      }
    };
    async function getStream2(inputStream, options) {
      if (!inputStream) {
        throw new Error("Expected a stream");
      }
      options = {
        maxBuffer: Infinity,
        ...options
      };
      const { maxBuffer } = options;
      const stream3 = bufferStream(options);
      await new Promise((resolve, reject) => {
        const rejectPromise = (error) => {
          if (error && stream3.getBufferedLength() <= BufferConstants.MAX_LENGTH) {
            error.bufferedData = stream3.getBufferedValue();
          }
          reject(error);
        };
        (async () => {
          try {
            await streamPipelinePromisified(inputStream, stream3);
            resolve();
          } catch (error) {
            rejectPromise(error);
          }
        })();
        stream3.on("data", () => {
          if (stream3.getBufferedLength() > maxBuffer) {
            rejectPromise(new MaxBufferError());
          }
        });
      });
      return stream3.getBufferedValue();
    }
    module.exports = getStream2;
    module.exports.buffer = (stream3, options) => getStream2(stream3, { ...options, encoding: "buffer" });
    module.exports.array = (stream3, options) => getStream2(stream3, { ...options, array: true });
    module.exports.MaxBufferError = MaxBufferError;
  }
});

// node_modules/.pnpm/http-cache-semantics@4.1.1/node_modules/http-cache-semantics/index.js
var require_http_cache_semantics = __commonJS({
  "node_modules/.pnpm/http-cache-semantics@4.1.1/node_modules/http-cache-semantics/index.js"(exports, module) {
    "use strict";
    var statusCodeCacheableByDefault = /* @__PURE__ */ new Set([
      200,
      203,
      204,
      206,
      300,
      301,
      308,
      404,
      405,
      410,
      414,
      501
    ]);
    var understoodStatuses = /* @__PURE__ */ new Set([
      200,
      203,
      204,
      300,
      301,
      302,
      303,
      307,
      308,
      404,
      405,
      410,
      414,
      501
    ]);
    var errorStatusCodes = /* @__PURE__ */ new Set([
      500,
      502,
      503,
      504
    ]);
    var hopByHopHeaders = {
      date: true,
      // included, because we add Age update Date
      connection: true,
      "keep-alive": true,
      "proxy-authenticate": true,
      "proxy-authorization": true,
      te: true,
      trailer: true,
      "transfer-encoding": true,
      upgrade: true
    };
    var excludedFromRevalidationUpdate = {
      // Since the old body is reused, it doesn't make sense to change properties of the body
      "content-length": true,
      "content-encoding": true,
      "transfer-encoding": true,
      "content-range": true
    };
    function toNumberOrZero(s) {
      const n = parseInt(s, 10);
      return isFinite(n) ? n : 0;
    }
    function isErrorResponse(response) {
      if (!response) {
        return true;
      }
      return errorStatusCodes.has(response.status);
    }
    function parseCacheControl(header) {
      const cc = {};
      if (!header)
        return cc;
      const parts = header.trim().split(/,/);
      for (const part of parts) {
        const [k, v] = part.split(/=/, 2);
        cc[k.trim()] = v === void 0 ? true : v.trim().replace(/^"|"$/g, "");
      }
      return cc;
    }
    function formatCacheControl(cc) {
      let parts = [];
      for (const k in cc) {
        const v = cc[k];
        parts.push(v === true ? k : k + "=" + v);
      }
      if (!parts.length) {
        return void 0;
      }
      return parts.join(", ");
    }
    module.exports = class CachePolicy {
      constructor(req, res, {
        shared,
        cacheHeuristic,
        immutableMinTimeToLive,
        ignoreCargoCult,
        _fromObject
      } = {}) {
        if (_fromObject) {
          this._fromObject(_fromObject);
          return;
        }
        if (!res || !res.headers) {
          throw Error("Response headers missing");
        }
        this._assertRequestHasHeaders(req);
        this._responseTime = this.now();
        this._isShared = shared !== false;
        this._cacheHeuristic = void 0 !== cacheHeuristic ? cacheHeuristic : 0.1;
        this._immutableMinTtl = void 0 !== immutableMinTimeToLive ? immutableMinTimeToLive : 24 * 3600 * 1e3;
        this._status = "status" in res ? res.status : 200;
        this._resHeaders = res.headers;
        this._rescc = parseCacheControl(res.headers["cache-control"]);
        this._method = "method" in req ? req.method : "GET";
        this._url = req.url;
        this._host = req.headers.host;
        this._noAuthorization = !req.headers.authorization;
        this._reqHeaders = res.headers.vary ? req.headers : null;
        this._reqcc = parseCacheControl(req.headers["cache-control"]);
        if (ignoreCargoCult && "pre-check" in this._rescc && "post-check" in this._rescc) {
          delete this._rescc["pre-check"];
          delete this._rescc["post-check"];
          delete this._rescc["no-cache"];
          delete this._rescc["no-store"];
          delete this._rescc["must-revalidate"];
          this._resHeaders = Object.assign({}, this._resHeaders, {
            "cache-control": formatCacheControl(this._rescc)
          });
          delete this._resHeaders.expires;
          delete this._resHeaders.pragma;
        }
        if (res.headers["cache-control"] == null && /no-cache/.test(res.headers.pragma)) {
          this._rescc["no-cache"] = true;
        }
      }
      now() {
        return Date.now();
      }
      storable() {
        return !!(!this._reqcc["no-store"] && // A cache MUST NOT store a response to any request, unless:
        // The request method is understood by the cache and defined as being cacheable, and
        ("GET" === this._method || "HEAD" === this._method || "POST" === this._method && this._hasExplicitExpiration()) && // the response status code is understood by the cache, and
        understoodStatuses.has(this._status) && // the "no-store" cache directive does not appear in request or response header fields, and
        !this._rescc["no-store"] && // the "private" response directive does not appear in the response, if the cache is shared, and
        (!this._isShared || !this._rescc.private) && // the Authorization header field does not appear in the request, if the cache is shared,
        (!this._isShared || this._noAuthorization || this._allowsStoringAuthenticated()) && // the response either:
        // contains an Expires header field, or
        (this._resHeaders.expires || // contains a max-age response directive, or
        // contains a s-maxage response directive and the cache is shared, or
        // contains a public response directive.
        this._rescc["max-age"] || this._isShared && this._rescc["s-maxage"] || this._rescc.public || // has a status code that is defined as cacheable by default
        statusCodeCacheableByDefault.has(this._status)));
      }
      _hasExplicitExpiration() {
        return this._isShared && this._rescc["s-maxage"] || this._rescc["max-age"] || this._resHeaders.expires;
      }
      _assertRequestHasHeaders(req) {
        if (!req || !req.headers) {
          throw Error("Request headers missing");
        }
      }
      satisfiesWithoutRevalidation(req) {
        this._assertRequestHasHeaders(req);
        const requestCC = parseCacheControl(req.headers["cache-control"]);
        if (requestCC["no-cache"] || /no-cache/.test(req.headers.pragma)) {
          return false;
        }
        if (requestCC["max-age"] && this.age() > requestCC["max-age"]) {
          return false;
        }
        if (requestCC["min-fresh"] && this.timeToLive() < 1e3 * requestCC["min-fresh"]) {
          return false;
        }
        if (this.stale()) {
          const allowsStale = requestCC["max-stale"] && !this._rescc["must-revalidate"] && (true === requestCC["max-stale"] || requestCC["max-stale"] > this.age() - this.maxAge());
          if (!allowsStale) {
            return false;
          }
        }
        return this._requestMatches(req, false);
      }
      _requestMatches(req, allowHeadMethod) {
        return (!this._url || this._url === req.url) && this._host === req.headers.host && // the request method associated with the stored response allows it to be used for the presented request, and
        (!req.method || this._method === req.method || allowHeadMethod && "HEAD" === req.method) && // selecting header fields nominated by the stored response (if any) match those presented, and
        this._varyMatches(req);
      }
      _allowsStoringAuthenticated() {
        return this._rescc["must-revalidate"] || this._rescc.public || this._rescc["s-maxage"];
      }
      _varyMatches(req) {
        if (!this._resHeaders.vary) {
          return true;
        }
        if (this._resHeaders.vary === "*") {
          return false;
        }
        const fields = this._resHeaders.vary.trim().toLowerCase().split(/\s*,\s*/);
        for (const name of fields) {
          if (req.headers[name] !== this._reqHeaders[name])
            return false;
        }
        return true;
      }
      _copyWithoutHopByHopHeaders(inHeaders) {
        const headers = {};
        for (const name in inHeaders) {
          if (hopByHopHeaders[name])
            continue;
          headers[name] = inHeaders[name];
        }
        if (inHeaders.connection) {
          const tokens = inHeaders.connection.trim().split(/\s*,\s*/);
          for (const name of tokens) {
            delete headers[name];
          }
        }
        if (headers.warning) {
          const warnings = headers.warning.split(/,/).filter((warning) => {
            return !/^\s*1[0-9][0-9]/.test(warning);
          });
          if (!warnings.length) {
            delete headers.warning;
          } else {
            headers.warning = warnings.join(",").trim();
          }
        }
        return headers;
      }
      responseHeaders() {
        const headers = this._copyWithoutHopByHopHeaders(this._resHeaders);
        const age = this.age();
        if (age > 3600 * 24 && !this._hasExplicitExpiration() && this.maxAge() > 3600 * 24) {
          headers.warning = (headers.warning ? `${headers.warning}, ` : "") + '113 - "rfc7234 5.5.4"';
        }
        headers.age = `${Math.round(age)}`;
        headers.date = new Date(this.now()).toUTCString();
        return headers;
      }
      /**
       * Value of the Date response header or current time if Date was invalid
       * @return timestamp
       */
      date() {
        const serverDate = Date.parse(this._resHeaders.date);
        if (isFinite(serverDate)) {
          return serverDate;
        }
        return this._responseTime;
      }
      /**
       * Value of the Age header, in seconds, updated for the current time.
       * May be fractional.
       *
       * @return Number
       */
      age() {
        let age = this._ageValue();
        const residentTime = (this.now() - this._responseTime) / 1e3;
        return age + residentTime;
      }
      _ageValue() {
        return toNumberOrZero(this._resHeaders.age);
      }
      /**
       * Value of applicable max-age (or heuristic equivalent) in seconds. This counts since response's `Date`.
       *
       * For an up-to-date value, see `timeToLive()`.
       *
       * @return Number
       */
      maxAge() {
        if (!this.storable() || this._rescc["no-cache"]) {
          return 0;
        }
        if (this._isShared && (this._resHeaders["set-cookie"] && !this._rescc.public && !this._rescc.immutable)) {
          return 0;
        }
        if (this._resHeaders.vary === "*") {
          return 0;
        }
        if (this._isShared) {
          if (this._rescc["proxy-revalidate"]) {
            return 0;
          }
          if (this._rescc["s-maxage"]) {
            return toNumberOrZero(this._rescc["s-maxage"]);
          }
        }
        if (this._rescc["max-age"]) {
          return toNumberOrZero(this._rescc["max-age"]);
        }
        const defaultMinTtl = this._rescc.immutable ? this._immutableMinTtl : 0;
        const serverDate = this.date();
        if (this._resHeaders.expires) {
          const expires = Date.parse(this._resHeaders.expires);
          if (Number.isNaN(expires) || expires < serverDate) {
            return 0;
          }
          return Math.max(defaultMinTtl, (expires - serverDate) / 1e3);
        }
        if (this._resHeaders["last-modified"]) {
          const lastModified = Date.parse(this._resHeaders["last-modified"]);
          if (isFinite(lastModified) && serverDate > lastModified) {
            return Math.max(
              defaultMinTtl,
              (serverDate - lastModified) / 1e3 * this._cacheHeuristic
            );
          }
        }
        return defaultMinTtl;
      }
      timeToLive() {
        const age = this.maxAge() - this.age();
        const staleIfErrorAge = age + toNumberOrZero(this._rescc["stale-if-error"]);
        const staleWhileRevalidateAge = age + toNumberOrZero(this._rescc["stale-while-revalidate"]);
        return Math.max(0, age, staleIfErrorAge, staleWhileRevalidateAge) * 1e3;
      }
      stale() {
        return this.maxAge() <= this.age();
      }
      _useStaleIfError() {
        return this.maxAge() + toNumberOrZero(this._rescc["stale-if-error"]) > this.age();
      }
      useStaleWhileRevalidate() {
        return this.maxAge() + toNumberOrZero(this._rescc["stale-while-revalidate"]) > this.age();
      }
      static fromObject(obj) {
        return new this(void 0, void 0, { _fromObject: obj });
      }
      _fromObject(obj) {
        if (this._responseTime)
          throw Error("Reinitialized");
        if (!obj || obj.v !== 1)
          throw Error("Invalid serialization");
        this._responseTime = obj.t;
        this._isShared = obj.sh;
        this._cacheHeuristic = obj.ch;
        this._immutableMinTtl = obj.imm !== void 0 ? obj.imm : 24 * 3600 * 1e3;
        this._status = obj.st;
        this._resHeaders = obj.resh;
        this._rescc = obj.rescc;
        this._method = obj.m;
        this._url = obj.u;
        this._host = obj.h;
        this._noAuthorization = obj.a;
        this._reqHeaders = obj.reqh;
        this._reqcc = obj.reqcc;
      }
      toObject() {
        return {
          v: 1,
          t: this._responseTime,
          sh: this._isShared,
          ch: this._cacheHeuristic,
          imm: this._immutableMinTtl,
          st: this._status,
          resh: this._resHeaders,
          rescc: this._rescc,
          m: this._method,
          u: this._url,
          h: this._host,
          a: this._noAuthorization,
          reqh: this._reqHeaders,
          reqcc: this._reqcc
        };
      }
      /**
       * Headers for sending to the origin server to revalidate stale response.
       * Allows server to return 304 to allow reuse of the previous response.
       *
       * Hop by hop headers are always stripped.
       * Revalidation headers may be added or removed, depending on request.
       */
      revalidationHeaders(incomingReq) {
        this._assertRequestHasHeaders(incomingReq);
        const headers = this._copyWithoutHopByHopHeaders(incomingReq.headers);
        delete headers["if-range"];
        if (!this._requestMatches(incomingReq, true) || !this.storable()) {
          delete headers["if-none-match"];
          delete headers["if-modified-since"];
          return headers;
        }
        if (this._resHeaders.etag) {
          headers["if-none-match"] = headers["if-none-match"] ? `${headers["if-none-match"]}, ${this._resHeaders.etag}` : this._resHeaders.etag;
        }
        const forbidsWeakValidators = headers["accept-ranges"] || headers["if-match"] || headers["if-unmodified-since"] || this._method && this._method != "GET";
        if (forbidsWeakValidators) {
          delete headers["if-modified-since"];
          if (headers["if-none-match"]) {
            const etags = headers["if-none-match"].split(/,/).filter((etag) => {
              return !/^\s*W\//.test(etag);
            });
            if (!etags.length) {
              delete headers["if-none-match"];
            } else {
              headers["if-none-match"] = etags.join(",").trim();
            }
          }
        } else if (this._resHeaders["last-modified"] && !headers["if-modified-since"]) {
          headers["if-modified-since"] = this._resHeaders["last-modified"];
        }
        return headers;
      }
      /**
       * Creates new CachePolicy with information combined from the previews response,
       * and the new revalidation response.
       *
       * Returns {policy, modified} where modified is a boolean indicating
       * whether the response body has been modified, and old cached body can't be used.
       *
       * @return {Object} {policy: CachePolicy, modified: Boolean}
       */
      revalidatedPolicy(request, response) {
        this._assertRequestHasHeaders(request);
        if (this._useStaleIfError() && isErrorResponse(response)) {
          return {
            modified: false,
            matches: false,
            policy: this
          };
        }
        if (!response || !response.headers) {
          throw Error("Response headers missing");
        }
        let matches = false;
        if (response.status !== void 0 && response.status != 304) {
          matches = false;
        } else if (response.headers.etag && !/^\s*W\//.test(response.headers.etag)) {
          matches = this._resHeaders.etag && this._resHeaders.etag.replace(/^\s*W\//, "") === response.headers.etag;
        } else if (this._resHeaders.etag && response.headers.etag) {
          matches = this._resHeaders.etag.replace(/^\s*W\//, "") === response.headers.etag.replace(/^\s*W\//, "");
        } else if (this._resHeaders["last-modified"]) {
          matches = this._resHeaders["last-modified"] === response.headers["last-modified"];
        } else {
          if (!this._resHeaders.etag && !this._resHeaders["last-modified"] && !response.headers.etag && !response.headers["last-modified"]) {
            matches = true;
          }
        }
        if (!matches) {
          return {
            policy: new this.constructor(request, response),
            // Client receiving 304 without body, even if it's invalid/mismatched has no option
            // but to reuse a cached body. We don't have a good way to tell clients to do
            // error recovery in such case.
            modified: response.status != 304,
            matches: false
          };
        }
        const headers = {};
        for (const k in this._resHeaders) {
          headers[k] = k in response.headers && !excludedFromRevalidationUpdate[k] ? response.headers[k] : this._resHeaders[k];
        }
        const newResponse = Object.assign({}, response, {
          status: this._status,
          method: this._method,
          headers
        });
        return {
          policy: new this.constructor(request, newResponse, {
            shared: this._isShared,
            cacheHeuristic: this._cacheHeuristic,
            immutableMinTimeToLive: this._immutableMinTtl
          }),
          modified: false,
          matches: true
        };
      }
    };
  }
});

// node_modules/.pnpm/json-buffer@3.0.1/node_modules/json-buffer/index.js
var require_json_buffer = __commonJS({
  "node_modules/.pnpm/json-buffer@3.0.1/node_modules/json-buffer/index.js"(exports) {
    exports.stringify = function stringify(o) {
      if ("undefined" == typeof o)
        return o;
      if (o && Buffer.isBuffer(o))
        return JSON.stringify(":base64:" + o.toString("base64"));
      if (o && o.toJSON)
        o = o.toJSON();
      if (o && "object" === typeof o) {
        var s = "";
        var array = Array.isArray(o);
        s = array ? "[" : "{";
        var first = true;
        for (var k in o) {
          var ignore = "function" == typeof o[k] || !array && "undefined" === typeof o[k];
          if (Object.hasOwnProperty.call(o, k) && !ignore) {
            if (!first)
              s += ",";
            first = false;
            if (array) {
              if (o[k] == void 0)
                s += "null";
              else
                s += stringify(o[k]);
            } else if (o[k] !== void 0) {
              s += stringify(k) + ":" + stringify(o[k]);
            }
          }
        }
        s += array ? "]" : "}";
        return s;
      } else if ("string" === typeof o) {
        return JSON.stringify(/^:/.test(o) ? ":" + o : o);
      } else if ("undefined" === typeof o) {
        return "null";
      } else
        return JSON.stringify(o);
    };
    exports.parse = function(s) {
      return JSON.parse(s, function(key, value) {
        if ("string" === typeof value) {
          if (/^:base64:/.test(value))
            return Buffer.from(value.substring(8), "base64");
          else
            return /^:/.test(value) ? value.substring(1) : value;
        }
        return value;
      });
    };
  }
});

// node_modules/.pnpm/keyv@4.5.2/node_modules/keyv/src/index.js
var require_src = __commonJS({
  "node_modules/.pnpm/keyv@4.5.2/node_modules/keyv/src/index.js"(exports, module) {
    "use strict";
    var EventEmitter3 = __require("events");
    var JSONB = require_json_buffer();
    var loadStore = (options) => {
      const adapters = {
        redis: "@keyv/redis",
        rediss: "@keyv/redis",
        mongodb: "@keyv/mongo",
        mongo: "@keyv/mongo",
        sqlite: "@keyv/sqlite",
        postgresql: "@keyv/postgres",
        postgres: "@keyv/postgres",
        mysql: "@keyv/mysql",
        etcd: "@keyv/etcd",
        offline: "@keyv/offline",
        tiered: "@keyv/tiered"
      };
      if (options.adapter || options.uri) {
        const adapter = options.adapter || /^[^:+]*/.exec(options.uri)[0];
        return new (__require(adapters[adapter]))(options);
      }
      return /* @__PURE__ */ new Map();
    };
    var iterableAdapters = [
      "sqlite",
      "postgres",
      "mysql",
      "mongo",
      "redis",
      "tiered"
    ];
    var Keyv2 = class extends EventEmitter3 {
      constructor(uri, { emitErrors = true, ...options } = {}) {
        super();
        this.opts = {
          namespace: "keyv",
          serialize: JSONB.stringify,
          deserialize: JSONB.parse,
          ...typeof uri === "string" ? { uri } : uri,
          ...options
        };
        if (!this.opts.store) {
          const adapterOptions = { ...this.opts };
          this.opts.store = loadStore(adapterOptions);
        }
        if (this.opts.compression) {
          const compression = this.opts.compression;
          this.opts.serialize = compression.serialize.bind(compression);
          this.opts.deserialize = compression.deserialize.bind(compression);
        }
        if (typeof this.opts.store.on === "function" && emitErrors) {
          this.opts.store.on("error", (error) => this.emit("error", error));
        }
        this.opts.store.namespace = this.opts.namespace;
        const generateIterator = (iterator) => async function* () {
          for await (const [key, raw] of typeof iterator === "function" ? iterator(this.opts.store.namespace) : iterator) {
            const data = this.opts.deserialize(raw);
            if (this.opts.store.namespace && !key.includes(this.opts.store.namespace)) {
              continue;
            }
            if (typeof data.expires === "number" && Date.now() > data.expires) {
              this.delete(key);
              continue;
            }
            yield [this._getKeyUnprefix(key), data.value];
          }
        };
        if (typeof this.opts.store[Symbol.iterator] === "function" && this.opts.store instanceof Map) {
          this.iterator = generateIterator(this.opts.store);
        } else if (typeof this.opts.store.iterator === "function" && this.opts.store.opts && this._checkIterableAdaptar()) {
          this.iterator = generateIterator(this.opts.store.iterator.bind(this.opts.store));
        }
      }
      _checkIterableAdaptar() {
        return iterableAdapters.includes(this.opts.store.opts.dialect) || iterableAdapters.findIndex((element) => this.opts.store.opts.url.includes(element)) >= 0;
      }
      _getKeyPrefix(key) {
        return `${this.opts.namespace}:${key}`;
      }
      _getKeyPrefixArray(keys) {
        return keys.map((key) => `${this.opts.namespace}:${key}`);
      }
      _getKeyUnprefix(key) {
        return key.split(":").splice(1).join(":");
      }
      get(key, options) {
        const { store } = this.opts;
        const isArray = Array.isArray(key);
        const keyPrefixed = isArray ? this._getKeyPrefixArray(key) : this._getKeyPrefix(key);
        if (isArray && store.getMany === void 0) {
          const promises = [];
          for (const key2 of keyPrefixed) {
            promises.push(
              Promise.resolve().then(() => store.get(key2)).then((data) => typeof data === "string" ? this.opts.deserialize(data) : this.opts.compression ? this.opts.deserialize(data) : data).then((data) => {
                if (data === void 0 || data === null) {
                  return void 0;
                }
                if (typeof data.expires === "number" && Date.now() > data.expires) {
                  return this.delete(key2).then(() => void 0);
                }
                return options && options.raw ? data : data.value;
              })
            );
          }
          return Promise.allSettled(promises).then((values) => {
            const data = [];
            for (const value of values) {
              data.push(value.value);
            }
            return data;
          });
        }
        return Promise.resolve().then(() => isArray ? store.getMany(keyPrefixed) : store.get(keyPrefixed)).then((data) => typeof data === "string" ? this.opts.deserialize(data) : this.opts.compression ? this.opts.deserialize(data) : data).then((data) => {
          if (data === void 0 || data === null) {
            return void 0;
          }
          if (isArray) {
            const result = [];
            for (let row of data) {
              if (typeof row === "string") {
                row = this.opts.deserialize(row);
              }
              if (row === void 0 || row === null) {
                result.push(void 0);
                continue;
              }
              if (typeof row.expires === "number" && Date.now() > row.expires) {
                this.delete(key).then(() => void 0);
                result.push(void 0);
              } else {
                result.push(options && options.raw ? row : row.value);
              }
            }
            return result;
          }
          if (typeof data.expires === "number" && Date.now() > data.expires) {
            return this.delete(key).then(() => void 0);
          }
          return options && options.raw ? data : data.value;
        });
      }
      set(key, value, ttl2) {
        const keyPrefixed = this._getKeyPrefix(key);
        if (typeof ttl2 === "undefined") {
          ttl2 = this.opts.ttl;
        }
        if (ttl2 === 0) {
          ttl2 = void 0;
        }
        const { store } = this.opts;
        return Promise.resolve().then(() => {
          const expires = typeof ttl2 === "number" ? Date.now() + ttl2 : null;
          if (typeof value === "symbol") {
            this.emit("error", "symbol cannot be serialized");
          }
          value = { value, expires };
          return this.opts.serialize(value);
        }).then((value2) => store.set(keyPrefixed, value2, ttl2)).then(() => true);
      }
      delete(key) {
        const { store } = this.opts;
        if (Array.isArray(key)) {
          const keyPrefixed2 = this._getKeyPrefixArray(key);
          if (store.deleteMany === void 0) {
            const promises = [];
            for (const key2 of keyPrefixed2) {
              promises.push(store.delete(key2));
            }
            return Promise.allSettled(promises).then((values) => values.every((x) => x.value === true));
          }
          return Promise.resolve().then(() => store.deleteMany(keyPrefixed2));
        }
        const keyPrefixed = this._getKeyPrefix(key);
        return Promise.resolve().then(() => store.delete(keyPrefixed));
      }
      clear() {
        const { store } = this.opts;
        return Promise.resolve().then(() => store.clear());
      }
      has(key) {
        const keyPrefixed = this._getKeyPrefix(key);
        const { store } = this.opts;
        return Promise.resolve().then(async () => {
          if (typeof store.has === "function") {
            return store.has(keyPrefixed);
          }
          const value = await store.get(keyPrefixed);
          return value !== void 0;
        });
      }
      disconnect() {
        const { store } = this.opts;
        if (typeof store.disconnect === "function") {
          return store.disconnect();
        }
      }
    };
    module.exports = Keyv2;
  }
});

// node_modules/.pnpm/mimic-response@3.1.0/node_modules/mimic-response/index.js
var require_mimic_response = __commonJS({
  "node_modules/.pnpm/mimic-response@3.1.0/node_modules/mimic-response/index.js"(exports, module) {
    "use strict";
    var knownProperties2 = [
      "aborted",
      "complete",
      "headers",
      "httpVersion",
      "httpVersionMinor",
      "httpVersionMajor",
      "method",
      "rawHeaders",
      "rawTrailers",
      "setTimeout",
      "socket",
      "statusCode",
      "statusMessage",
      "trailers",
      "url"
    ];
    module.exports = (fromStream, toStream) => {
      if (toStream._readableState.autoDestroy) {
        throw new Error("The second stream must have the `autoDestroy` option set to `false`");
      }
      const fromProperties = new Set(Object.keys(fromStream).concat(knownProperties2));
      const properties = {};
      for (const property of fromProperties) {
        if (property in toStream) {
          continue;
        }
        properties[property] = {
          get() {
            const value = fromStream[property];
            const isFunction2 = typeof value === "function";
            return isFunction2 ? value.bind(fromStream) : value;
          },
          set(value) {
            fromStream[property] = value;
          },
          enumerable: true,
          configurable: false
        };
      }
      Object.defineProperties(toStream, properties);
      fromStream.once("aborted", () => {
        toStream.destroy();
        toStream.emit("aborted");
      });
      fromStream.once("close", () => {
        if (fromStream.complete) {
          if (toStream.readable) {
            toStream.once("end", () => {
              toStream.emit("close");
            });
          } else {
            toStream.emit("close");
          }
        } else {
          toStream.emit("close");
        }
      });
      return toStream;
    };
  }
});

// node_modules/.pnpm/decompress-response@6.0.0/node_modules/decompress-response/index.js
var require_decompress_response = __commonJS({
  "node_modules/.pnpm/decompress-response@6.0.0/node_modules/decompress-response/index.js"(exports, module) {
    "use strict";
    var { Transform, PassThrough } = __require("stream");
    var zlib = __require("zlib");
    var mimicResponse2 = require_mimic_response();
    module.exports = (response) => {
      const contentEncoding = (response.headers["content-encoding"] || "").toLowerCase();
      if (!["gzip", "deflate", "br"].includes(contentEncoding)) {
        return response;
      }
      const isBrotli = contentEncoding === "br";
      if (isBrotli && typeof zlib.createBrotliDecompress !== "function") {
        response.destroy(new Error("Brotli is not supported on Node.js < 12"));
        return response;
      }
      let isEmpty = true;
      const checker = new Transform({
        transform(data, _encoding, callback) {
          isEmpty = false;
          callback(null, data);
        },
        flush(callback) {
          callback();
        }
      });
      const finalStream = new PassThrough({
        autoDestroy: false,
        destroy(error, callback) {
          response.destroy();
          callback(error);
        }
      });
      const decompressStream = isBrotli ? zlib.createBrotliDecompress() : zlib.createUnzip();
      decompressStream.once("error", (error) => {
        if (isEmpty && !response.readable) {
          finalStream.end();
          return;
        }
        finalStream.destroy(error);
      });
      mimicResponse2(response, finalStream);
      response.pipe(checker).pipe(decompressStream).pipe(finalStream);
      return finalStream;
    };
  }
});

// node_modules/.pnpm/quick-lru@5.1.1/node_modules/quick-lru/index.js
var require_quick_lru = __commonJS({
  "node_modules/.pnpm/quick-lru@5.1.1/node_modules/quick-lru/index.js"(exports, module) {
    "use strict";
    var QuickLRU = class {
      constructor(options = {}) {
        if (!(options.maxSize && options.maxSize > 0)) {
          throw new TypeError("`maxSize` must be a number greater than 0");
        }
        this.maxSize = options.maxSize;
        this.onEviction = options.onEviction;
        this.cache = /* @__PURE__ */ new Map();
        this.oldCache = /* @__PURE__ */ new Map();
        this._size = 0;
      }
      _set(key, value) {
        this.cache.set(key, value);
        this._size++;
        if (this._size >= this.maxSize) {
          this._size = 0;
          if (typeof this.onEviction === "function") {
            for (const [key2, value2] of this.oldCache.entries()) {
              this.onEviction(key2, value2);
            }
          }
          this.oldCache = this.cache;
          this.cache = /* @__PURE__ */ new Map();
        }
      }
      get(key) {
        if (this.cache.has(key)) {
          return this.cache.get(key);
        }
        if (this.oldCache.has(key)) {
          const value = this.oldCache.get(key);
          this.oldCache.delete(key);
          this._set(key, value);
          return value;
        }
      }
      set(key, value) {
        if (this.cache.has(key)) {
          this.cache.set(key, value);
        } else {
          this._set(key, value);
        }
        return this;
      }
      has(key) {
        return this.cache.has(key) || this.oldCache.has(key);
      }
      peek(key) {
        if (this.cache.has(key)) {
          return this.cache.get(key);
        }
        if (this.oldCache.has(key)) {
          return this.oldCache.get(key);
        }
      }
      delete(key) {
        const deleted = this.cache.delete(key);
        if (deleted) {
          this._size--;
        }
        return this.oldCache.delete(key) || deleted;
      }
      clear() {
        this.cache.clear();
        this.oldCache.clear();
        this._size = 0;
      }
      *keys() {
        for (const [key] of this) {
          yield key;
        }
      }
      *values() {
        for (const [, value] of this) {
          yield value;
        }
      }
      *[Symbol.iterator]() {
        for (const item of this.cache) {
          yield item;
        }
        for (const item of this.oldCache) {
          const [key] = item;
          if (!this.cache.has(key)) {
            yield item;
          }
        }
      }
      get size() {
        let oldCacheSize = 0;
        for (const key of this.oldCache.keys()) {
          if (!this.cache.has(key)) {
            oldCacheSize++;
          }
        }
        return Math.min(this._size + oldCacheSize, this.maxSize);
      }
    };
    module.exports = QuickLRU;
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/delay-async-destroy.js
var require_delay_async_destroy = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/delay-async-destroy.js"(exports, module) {
    "use strict";
    module.exports = (stream2) => {
      if (stream2.listenerCount("error") !== 0) {
        return stream2;
      }
      stream2.__destroy = stream2._destroy;
      stream2._destroy = (...args) => {
        const callback = args.pop();
        stream2.__destroy(...args, async (error) => {
          await Promise.resolve();
          callback(error);
        });
      };
      const onError = (error) => {
        Promise.resolve().then(() => {
          stream2.emit("error", error);
        });
      };
      stream2.once("error", onError);
      Promise.resolve().then(() => {
        stream2.off("error", onError);
      });
      return stream2;
    };
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/agent.js
var require_agent = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/agent.js"(exports, module) {
    "use strict";
    var { URL: URL4 } = __require("url");
    var EventEmitter3 = __require("events");
    var tls = __require("tls");
    var http22 = __require("http2");
    var QuickLRU = require_quick_lru();
    var delayAsyncDestroy = require_delay_async_destroy();
    var kCurrentStreamCount = Symbol("currentStreamCount");
    var kRequest = Symbol("request");
    var kOriginSet = Symbol("cachedOriginSet");
    var kGracefullyClosing = Symbol("gracefullyClosing");
    var kLength = Symbol("length");
    var nameKeys = [
      // Not an Agent option actually
      "createConnection",
      // `http2.connect()` options
      "maxDeflateDynamicTableSize",
      "maxSettings",
      "maxSessionMemory",
      "maxHeaderListPairs",
      "maxOutstandingPings",
      "maxReservedRemoteStreams",
      "maxSendHeaderBlockLength",
      "paddingStrategy",
      "peerMaxConcurrentStreams",
      "settings",
      // `tls.connect()` source options
      "family",
      "localAddress",
      "rejectUnauthorized",
      // `tls.connect()` secure context options
      "pskCallback",
      "minDHSize",
      // `tls.connect()` destination options
      // - `servername` is automatically validated, skip it
      // - `host` and `port` just describe the destination server,
      "path",
      "socket",
      // `tls.createSecureContext()` options
      "ca",
      "cert",
      "sigalgs",
      "ciphers",
      "clientCertEngine",
      "crl",
      "dhparam",
      "ecdhCurve",
      "honorCipherOrder",
      "key",
      "privateKeyEngine",
      "privateKeyIdentifier",
      "maxVersion",
      "minVersion",
      "pfx",
      "secureOptions",
      "secureProtocol",
      "sessionIdContext",
      "ticketKeys"
    ];
    var getSortedIndex = (array, value, compare) => {
      let low = 0;
      let high = array.length;
      while (low < high) {
        const mid = low + high >>> 1;
        if (compare(array[mid], value)) {
          low = mid + 1;
        } else {
          high = mid;
        }
      }
      return low;
    };
    var compareSessions = (a, b) => a.remoteSettings.maxConcurrentStreams > b.remoteSettings.maxConcurrentStreams;
    var closeCoveredSessions = (where, session) => {
      for (let index = 0; index < where.length; index++) {
        const coveredSession = where[index];
        if (
          // Unfortunately `.every()` returns true for an empty array
          coveredSession[kOriginSet].length > 0 && coveredSession[kOriginSet].length < session[kOriginSet].length && coveredSession[kOriginSet].every((origin) => session[kOriginSet].includes(origin)) && coveredSession[kCurrentStreamCount] + session[kCurrentStreamCount] <= session.remoteSettings.maxConcurrentStreams
        ) {
          gracefullyClose(coveredSession);
        }
      }
    };
    var closeSessionIfCovered = (where, coveredSession) => {
      for (let index = 0; index < where.length; index++) {
        const session = where[index];
        if (coveredSession[kOriginSet].length > 0 && coveredSession[kOriginSet].length < session[kOriginSet].length && coveredSession[kOriginSet].every((origin) => session[kOriginSet].includes(origin)) && coveredSession[kCurrentStreamCount] + session[kCurrentStreamCount] <= session.remoteSettings.maxConcurrentStreams) {
          gracefullyClose(coveredSession);
          return true;
        }
      }
      return false;
    };
    var gracefullyClose = (session) => {
      session[kGracefullyClosing] = true;
      if (session[kCurrentStreamCount] === 0) {
        session.close();
      }
    };
    var Agent = class extends EventEmitter3 {
      constructor({ timeout = 0, maxSessions = Number.POSITIVE_INFINITY, maxEmptySessions = 10, maxCachedTlsSessions = 100 } = {}) {
        super();
        this.sessions = {};
        this.queue = {};
        this.timeout = timeout;
        this.maxSessions = maxSessions;
        this.maxEmptySessions = maxEmptySessions;
        this._emptySessionCount = 0;
        this._sessionCount = 0;
        this.settings = {
          enablePush: false,
          initialWindowSize: 1024 * 1024 * 32
          // 32MB, see https://github.com/nodejs/node/issues/38426
        };
        this.tlsSessionCache = new QuickLRU({ maxSize: maxCachedTlsSessions });
      }
      get protocol() {
        return "https:";
      }
      normalizeOptions(options) {
        let normalized = "";
        for (let index = 0; index < nameKeys.length; index++) {
          const key = nameKeys[index];
          normalized += ":";
          if (options && options[key] !== void 0) {
            normalized += options[key];
          }
        }
        return normalized;
      }
      _processQueue() {
        if (this._sessionCount >= this.maxSessions) {
          this.closeEmptySessions(this.maxSessions - this._sessionCount + 1);
          return;
        }
        for (const normalizedOptions in this.queue) {
          for (const normalizedOrigin in this.queue[normalizedOptions]) {
            const item = this.queue[normalizedOptions][normalizedOrigin];
            if (!item.completed) {
              item.completed = true;
              item();
            }
          }
        }
      }
      _isBetterSession(thisStreamCount, thatStreamCount) {
        return thisStreamCount > thatStreamCount;
      }
      _accept(session, listeners, normalizedOrigin, options) {
        let index = 0;
        while (index < listeners.length && session[kCurrentStreamCount] < session.remoteSettings.maxConcurrentStreams) {
          listeners[index].resolve(session);
          index++;
        }
        listeners.splice(0, index);
        if (listeners.length > 0) {
          this.getSession(normalizedOrigin, options, listeners);
          listeners.length = 0;
        }
      }
      getSession(origin, options, listeners) {
        return new Promise((resolve, reject) => {
          if (Array.isArray(listeners) && listeners.length > 0) {
            listeners = [...listeners];
            resolve();
          } else {
            listeners = [{ resolve, reject }];
          }
          try {
            if (typeof origin === "string") {
              origin = new URL4(origin);
            } else if (!(origin instanceof URL4)) {
              throw new TypeError("The `origin` argument needs to be a string or an URL object");
            }
            if (options) {
              const { servername } = options;
              const { hostname } = origin;
              if (servername && hostname !== servername) {
                throw new Error(`Origin ${hostname} differs from servername ${servername}`);
              }
            }
          } catch (error) {
            for (let index = 0; index < listeners.length; index++) {
              listeners[index].reject(error);
            }
            return;
          }
          const normalizedOptions = this.normalizeOptions(options);
          const normalizedOrigin = origin.origin;
          if (normalizedOptions in this.sessions) {
            const sessions = this.sessions[normalizedOptions];
            let maxConcurrentStreams = -1;
            let currentStreamsCount = -1;
            let optimalSession;
            for (let index = 0; index < sessions.length; index++) {
              const session = sessions[index];
              const sessionMaxConcurrentStreams = session.remoteSettings.maxConcurrentStreams;
              if (sessionMaxConcurrentStreams < maxConcurrentStreams) {
                break;
              }
              if (!session[kOriginSet].includes(normalizedOrigin)) {
                continue;
              }
              const sessionCurrentStreamsCount = session[kCurrentStreamCount];
              if (sessionCurrentStreamsCount >= sessionMaxConcurrentStreams || session[kGracefullyClosing] || session.destroyed) {
                continue;
              }
              if (!optimalSession) {
                maxConcurrentStreams = sessionMaxConcurrentStreams;
              }
              if (this._isBetterSession(sessionCurrentStreamsCount, currentStreamsCount)) {
                optimalSession = session;
                currentStreamsCount = sessionCurrentStreamsCount;
              }
            }
            if (optimalSession) {
              this._accept(optimalSession, listeners, normalizedOrigin, options);
              return;
            }
          }
          if (normalizedOptions in this.queue) {
            if (normalizedOrigin in this.queue[normalizedOptions]) {
              this.queue[normalizedOptions][normalizedOrigin].listeners.push(...listeners);
              return;
            }
          } else {
            this.queue[normalizedOptions] = {
              [kLength]: 0
            };
          }
          const removeFromQueue = () => {
            if (normalizedOptions in this.queue && this.queue[normalizedOptions][normalizedOrigin] === entry) {
              delete this.queue[normalizedOptions][normalizedOrigin];
              if (--this.queue[normalizedOptions][kLength] === 0) {
                delete this.queue[normalizedOptions];
              }
            }
          };
          const entry = async () => {
            this._sessionCount++;
            const name = `${normalizedOrigin}:${normalizedOptions}`;
            let receivedSettings = false;
            let socket;
            try {
              const computedOptions = { ...options };
              if (computedOptions.settings === void 0) {
                computedOptions.settings = this.settings;
              }
              if (computedOptions.session === void 0) {
                computedOptions.session = this.tlsSessionCache.get(name);
              }
              const createConnection = computedOptions.createConnection || this.createConnection;
              socket = await createConnection.call(this, origin, computedOptions);
              computedOptions.createConnection = () => socket;
              const session = http22.connect(origin, computedOptions);
              session[kCurrentStreamCount] = 0;
              session[kGracefullyClosing] = false;
              const getOriginSet = () => {
                const { socket: socket2 } = session;
                let originSet;
                if (socket2.servername === false) {
                  socket2.servername = socket2.remoteAddress;
                  originSet = session.originSet;
                  socket2.servername = false;
                } else {
                  originSet = session.originSet;
                }
                return originSet;
              };
              const isFree = () => session[kCurrentStreamCount] < session.remoteSettings.maxConcurrentStreams;
              session.socket.once("session", (tlsSession) => {
                this.tlsSessionCache.set(name, tlsSession);
              });
              session.once("error", (error) => {
                for (let index = 0; index < listeners.length; index++) {
                  listeners[index].reject(error);
                }
                this.tlsSessionCache.delete(name);
              });
              session.setTimeout(this.timeout, () => {
                session.destroy();
              });
              session.once("close", () => {
                this._sessionCount--;
                if (receivedSettings) {
                  this._emptySessionCount--;
                  const where = this.sessions[normalizedOptions];
                  if (where.length === 1) {
                    delete this.sessions[normalizedOptions];
                  } else {
                    where.splice(where.indexOf(session), 1);
                  }
                } else {
                  removeFromQueue();
                  const error = new Error("Session closed without receiving a SETTINGS frame");
                  error.code = "HTTP2WRAPPER_NOSETTINGS";
                  for (let index = 0; index < listeners.length; index++) {
                    listeners[index].reject(error);
                  }
                }
                this._processQueue();
              });
              const processListeners = () => {
                const queue = this.queue[normalizedOptions];
                if (!queue) {
                  return;
                }
                const originSet = session[kOriginSet];
                for (let index = 0; index < originSet.length; index++) {
                  const origin2 = originSet[index];
                  if (origin2 in queue) {
                    const { listeners: listeners2, completed } = queue[origin2];
                    let index2 = 0;
                    while (index2 < listeners2.length && isFree()) {
                      listeners2[index2].resolve(session);
                      index2++;
                    }
                    queue[origin2].listeners.splice(0, index2);
                    if (queue[origin2].listeners.length === 0 && !completed) {
                      delete queue[origin2];
                      if (--queue[kLength] === 0) {
                        delete this.queue[normalizedOptions];
                        break;
                      }
                    }
                    if (!isFree()) {
                      break;
                    }
                  }
                }
              };
              session.on("origin", () => {
                session[kOriginSet] = getOriginSet() || [];
                session[kGracefullyClosing] = false;
                closeSessionIfCovered(this.sessions[normalizedOptions], session);
                if (session[kGracefullyClosing] || !isFree()) {
                  return;
                }
                processListeners();
                if (!isFree()) {
                  return;
                }
                closeCoveredSessions(this.sessions[normalizedOptions], session);
              });
              session.once("remoteSettings", () => {
                if (entry.destroyed) {
                  const error = new Error("Agent has been destroyed");
                  for (let index = 0; index < listeners.length; index++) {
                    listeners[index].reject(error);
                  }
                  session.destroy();
                  return;
                }
                if (session.setLocalWindowSize) {
                  session.setLocalWindowSize(1024 * 1024 * 4);
                }
                session[kOriginSet] = getOriginSet() || [];
                if (session.socket.encrypted) {
                  const mainOrigin = session[kOriginSet][0];
                  if (mainOrigin !== normalizedOrigin) {
                    const error = new Error(`Requested origin ${normalizedOrigin} does not match server ${mainOrigin}`);
                    for (let index = 0; index < listeners.length; index++) {
                      listeners[index].reject(error);
                    }
                    session.destroy();
                    return;
                  }
                }
                removeFromQueue();
                {
                  const where = this.sessions;
                  if (normalizedOptions in where) {
                    const sessions = where[normalizedOptions];
                    sessions.splice(getSortedIndex(sessions, session, compareSessions), 0, session);
                  } else {
                    where[normalizedOptions] = [session];
                  }
                }
                receivedSettings = true;
                this._emptySessionCount++;
                this.emit("session", session);
                this._accept(session, listeners, normalizedOrigin, options);
                if (session[kCurrentStreamCount] === 0 && this._emptySessionCount > this.maxEmptySessions) {
                  this.closeEmptySessions(this._emptySessionCount - this.maxEmptySessions);
                }
                session.on("remoteSettings", () => {
                  if (!isFree()) {
                    return;
                  }
                  processListeners();
                  if (!isFree()) {
                    return;
                  }
                  closeCoveredSessions(this.sessions[normalizedOptions], session);
                });
              });
              session[kRequest] = session.request;
              session.request = (headers, streamOptions) => {
                if (session[kGracefullyClosing]) {
                  throw new Error("The session is gracefully closing. No new streams are allowed.");
                }
                const stream2 = session[kRequest](headers, streamOptions);
                session.ref();
                if (session[kCurrentStreamCount]++ === 0) {
                  this._emptySessionCount--;
                }
                stream2.once("close", () => {
                  if (--session[kCurrentStreamCount] === 0) {
                    this._emptySessionCount++;
                    session.unref();
                    if (this._emptySessionCount > this.maxEmptySessions || session[kGracefullyClosing]) {
                      session.close();
                      return;
                    }
                  }
                  if (session.destroyed || session.closed) {
                    return;
                  }
                  if (isFree() && !closeSessionIfCovered(this.sessions[normalizedOptions], session)) {
                    closeCoveredSessions(this.sessions[normalizedOptions], session);
                    processListeners();
                    if (session[kCurrentStreamCount] === 0) {
                      this._processQueue();
                    }
                  }
                });
                return stream2;
              };
            } catch (error) {
              removeFromQueue();
              this._sessionCount--;
              for (let index = 0; index < listeners.length; index++) {
                listeners[index].reject(error);
              }
            }
          };
          entry.listeners = listeners;
          entry.completed = false;
          entry.destroyed = false;
          this.queue[normalizedOptions][normalizedOrigin] = entry;
          this.queue[normalizedOptions][kLength]++;
          this._processQueue();
        });
      }
      request(origin, options, headers, streamOptions) {
        return new Promise((resolve, reject) => {
          this.getSession(origin, options, [{
            reject,
            resolve: (session) => {
              try {
                const stream2 = session.request(headers, streamOptions);
                delayAsyncDestroy(stream2);
                resolve(stream2);
              } catch (error) {
                reject(error);
              }
            }
          }]);
        });
      }
      async createConnection(origin, options) {
        return Agent.connect(origin, options);
      }
      static connect(origin, options) {
        options.ALPNProtocols = ["h2"];
        const port = origin.port || 443;
        const host = origin.hostname;
        if (typeof options.servername === "undefined") {
          options.servername = host;
        }
        const socket = tls.connect(port, host, options);
        if (options.socket) {
          socket._peername = {
            family: void 0,
            address: void 0,
            port
          };
        }
        return socket;
      }
      closeEmptySessions(maxCount = Number.POSITIVE_INFINITY) {
        let closedCount = 0;
        const { sessions } = this;
        for (const key in sessions) {
          const thisSessions = sessions[key];
          for (let index = 0; index < thisSessions.length; index++) {
            const session = thisSessions[index];
            if (session[kCurrentStreamCount] === 0) {
              closedCount++;
              session.close();
              if (closedCount >= maxCount) {
                return closedCount;
              }
            }
          }
        }
        return closedCount;
      }
      destroy(reason) {
        const { sessions, queue } = this;
        for (const key in sessions) {
          const thisSessions = sessions[key];
          for (let index = 0; index < thisSessions.length; index++) {
            thisSessions[index].destroy(reason);
          }
        }
        for (const normalizedOptions in queue) {
          const entries2 = queue[normalizedOptions];
          for (const normalizedOrigin in entries2) {
            entries2[normalizedOrigin].destroyed = true;
          }
        }
        this.queue = {};
        this.tlsSessionCache.clear();
      }
      get emptySessionCount() {
        return this._emptySessionCount;
      }
      get pendingSessionCount() {
        return this._sessionCount - this._emptySessionCount;
      }
      get sessionCount() {
        return this._sessionCount;
      }
    };
    Agent.kCurrentStreamCount = kCurrentStreamCount;
    Agent.kGracefullyClosing = kGracefullyClosing;
    module.exports = {
      Agent,
      globalAgent: new Agent()
    };
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/incoming-message.js
var require_incoming_message = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/incoming-message.js"(exports, module) {
    "use strict";
    var { Readable } = __require("stream");
    var IncomingMessage = class extends Readable {
      constructor(socket, highWaterMark) {
        super({
          emitClose: false,
          autoDestroy: true,
          highWaterMark
        });
        this.statusCode = null;
        this.statusMessage = "";
        this.httpVersion = "2.0";
        this.httpVersionMajor = 2;
        this.httpVersionMinor = 0;
        this.headers = {};
        this.trailers = {};
        this.req = null;
        this.aborted = false;
        this.complete = false;
        this.upgrade = null;
        this.rawHeaders = [];
        this.rawTrailers = [];
        this.socket = socket;
        this._dumped = false;
      }
      get connection() {
        return this.socket;
      }
      set connection(value) {
        this.socket = value;
      }
      _destroy(error, callback) {
        if (!this.readableEnded) {
          this.aborted = true;
        }
        callback();
        this.req._request.destroy(error);
      }
      setTimeout(ms, callback) {
        this.req.setTimeout(ms, callback);
        return this;
      }
      _dump() {
        if (!this._dumped) {
          this._dumped = true;
          this.removeAllListeners("data");
          this.resume();
        }
      }
      _read() {
        if (this.req) {
          this.req._request.resume();
        }
      }
    };
    module.exports = IncomingMessage;
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/proxy-events.js
var require_proxy_events = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/proxy-events.js"(exports, module) {
    "use strict";
    module.exports = (from, to, events) => {
      for (const event of events) {
        from.on(event, (...args) => to.emit(event, ...args));
      }
    };
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/errors.js
var require_errors = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/errors.js"(exports, module) {
    "use strict";
    var makeError = (Base, key, getMessage) => {
      module.exports[key] = class NodeError extends Base {
        constructor(...args) {
          super(typeof getMessage === "string" ? getMessage : getMessage(args));
          this.name = `${super.name} [${key}]`;
          this.code = key;
        }
      };
    };
    makeError(TypeError, "ERR_INVALID_ARG_TYPE", (args) => {
      const type = args[0].includes(".") ? "property" : "argument";
      let valid = args[1];
      const isManyTypes = Array.isArray(valid);
      if (isManyTypes) {
        valid = `${valid.slice(0, -1).join(", ")} or ${valid.slice(-1)}`;
      }
      return `The "${args[0]}" ${type} must be ${isManyTypes ? "one of" : "of"} type ${valid}. Received ${typeof args[2]}`;
    });
    makeError(
      TypeError,
      "ERR_INVALID_PROTOCOL",
      (args) => `Protocol "${args[0]}" not supported. Expected "${args[1]}"`
    );
    makeError(
      Error,
      "ERR_HTTP_HEADERS_SENT",
      (args) => `Cannot ${args[0]} headers after they are sent to the client`
    );
    makeError(
      TypeError,
      "ERR_INVALID_HTTP_TOKEN",
      (args) => `${args[0]} must be a valid HTTP token [${args[1]}]`
    );
    makeError(
      TypeError,
      "ERR_HTTP_INVALID_HEADER_VALUE",
      (args) => `Invalid value "${args[0]} for header "${args[1]}"`
    );
    makeError(
      TypeError,
      "ERR_INVALID_CHAR",
      (args) => `Invalid character in ${args[0]} [${args[1]}]`
    );
    makeError(
      Error,
      "ERR_HTTP2_NO_SOCKET_MANIPULATION",
      "HTTP/2 sockets should not be directly manipulated (e.g. read and written)"
    );
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/is-request-pseudo-header.js
var require_is_request_pseudo_header = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/is-request-pseudo-header.js"(exports, module) {
    "use strict";
    module.exports = (header) => {
      switch (header) {
        case ":method":
        case ":scheme":
        case ":authority":
        case ":path":
          return true;
        default:
          return false;
      }
    };
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/validate-header-name.js
var require_validate_header_name = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/validate-header-name.js"(exports, module) {
    "use strict";
    var { ERR_INVALID_HTTP_TOKEN } = require_errors();
    var isRequestPseudoHeader = require_is_request_pseudo_header();
    var isValidHttpToken = /^[\^`\-\w!#$%&*+.|~]+$/;
    module.exports = (name) => {
      if (typeof name !== "string" || !isValidHttpToken.test(name) && !isRequestPseudoHeader(name)) {
        throw new ERR_INVALID_HTTP_TOKEN("Header name", name);
      }
    };
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/validate-header-value.js
var require_validate_header_value = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/validate-header-value.js"(exports, module) {
    "use strict";
    var {
      ERR_HTTP_INVALID_HEADER_VALUE,
      ERR_INVALID_CHAR
    } = require_errors();
    var isInvalidHeaderValue = /[^\t\u0020-\u007E\u0080-\u00FF]/;
    module.exports = (name, value) => {
      if (typeof value === "undefined") {
        throw new ERR_HTTP_INVALID_HEADER_VALUE(value, name);
      }
      if (isInvalidHeaderValue.test(value)) {
        throw new ERR_INVALID_CHAR("header content", name);
      }
    };
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/proxy-socket-handler.js
var require_proxy_socket_handler = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/proxy-socket-handler.js"(exports, module) {
    "use strict";
    var { ERR_HTTP2_NO_SOCKET_MANIPULATION } = require_errors();
    var proxySocketHandler = {
      has(stream2, property) {
        const reference = stream2.session === void 0 ? stream2 : stream2.session.socket;
        return property in stream2 || property in reference;
      },
      get(stream2, property) {
        switch (property) {
          case "on":
          case "once":
          case "end":
          case "emit":
          case "destroy":
            return stream2[property].bind(stream2);
          case "writable":
          case "destroyed":
            return stream2[property];
          case "readable":
            if (stream2.destroyed) {
              return false;
            }
            return stream2.readable;
          case "setTimeout": {
            const { session } = stream2;
            if (session !== void 0) {
              return session.setTimeout.bind(session);
            }
            return stream2.setTimeout.bind(stream2);
          }
          case "write":
          case "read":
          case "pause":
          case "resume":
            throw new ERR_HTTP2_NO_SOCKET_MANIPULATION();
          default: {
            const reference = stream2.session === void 0 ? stream2 : stream2.session.socket;
            const value = reference[property];
            return typeof value === "function" ? value.bind(reference) : value;
          }
        }
      },
      getPrototypeOf(stream2) {
        if (stream2.session !== void 0) {
          return Reflect.getPrototypeOf(stream2.session.socket);
        }
        return Reflect.getPrototypeOf(stream2);
      },
      set(stream2, property, value) {
        switch (property) {
          case "writable":
          case "readable":
          case "destroyed":
          case "on":
          case "once":
          case "end":
          case "emit":
          case "destroy":
            stream2[property] = value;
            return true;
          case "setTimeout": {
            const { session } = stream2;
            if (session === void 0) {
              stream2.setTimeout = value;
            } else {
              session.setTimeout = value;
            }
            return true;
          }
          case "write":
          case "read":
          case "pause":
          case "resume":
            throw new ERR_HTTP2_NO_SOCKET_MANIPULATION();
          default: {
            const reference = stream2.session === void 0 ? stream2 : stream2.session.socket;
            reference[property] = value;
            return true;
          }
        }
      }
    };
    module.exports = proxySocketHandler;
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/client-request.js
var require_client_request = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/client-request.js"(exports, module) {
    "use strict";
    var { URL: URL4, urlToHttpOptions } = __require("url");
    var http22 = __require("http2");
    var { Writable } = __require("stream");
    var { Agent, globalAgent } = require_agent();
    var IncomingMessage = require_incoming_message();
    var proxyEvents2 = require_proxy_events();
    var {
      ERR_INVALID_ARG_TYPE,
      ERR_INVALID_PROTOCOL,
      ERR_HTTP_HEADERS_SENT
    } = require_errors();
    var validateHeaderName = require_validate_header_name();
    var validateHeaderValue = require_validate_header_value();
    var proxySocketHandler = require_proxy_socket_handler();
    var {
      HTTP2_HEADER_STATUS,
      HTTP2_HEADER_METHOD,
      HTTP2_HEADER_PATH,
      HTTP2_HEADER_AUTHORITY,
      HTTP2_METHOD_CONNECT
    } = http22.constants;
    var kHeaders = Symbol("headers");
    var kOrigin = Symbol("origin");
    var kSession = Symbol("session");
    var kOptions = Symbol("options");
    var kFlushedHeaders = Symbol("flushedHeaders");
    var kJobs = Symbol("jobs");
    var kPendingAgentPromise = Symbol("pendingAgentPromise");
    var ClientRequest = class extends Writable {
      constructor(input, options, callback) {
        super({
          autoDestroy: false,
          emitClose: false
        });
        if (typeof input === "string") {
          input = urlToHttpOptions(new URL4(input));
        } else if (input instanceof URL4) {
          input = urlToHttpOptions(input);
        } else {
          input = { ...input };
        }
        if (typeof options === "function" || options === void 0) {
          callback = options;
          options = input;
        } else {
          options = Object.assign(input, options);
        }
        if (options.h2session) {
          this[kSession] = options.h2session;
          if (this[kSession].destroyed) {
            throw new Error("The session has been closed already");
          }
          this.protocol = this[kSession].socket.encrypted ? "https:" : "http:";
        } else if (options.agent === false) {
          this.agent = new Agent({ maxEmptySessions: 0 });
        } else if (typeof options.agent === "undefined" || options.agent === null) {
          this.agent = globalAgent;
        } else if (typeof options.agent.request === "function") {
          this.agent = options.agent;
        } else {
          throw new ERR_INVALID_ARG_TYPE("options.agent", ["http2wrapper.Agent-like Object", "undefined", "false"], options.agent);
        }
        if (this.agent) {
          this.protocol = this.agent.protocol;
        }
        if (options.protocol && options.protocol !== this.protocol) {
          throw new ERR_INVALID_PROTOCOL(options.protocol, this.protocol);
        }
        if (!options.port) {
          options.port = options.defaultPort || this.agent && this.agent.defaultPort || 443;
        }
        options.host = options.hostname || options.host || "localhost";
        delete options.hostname;
        const { timeout } = options;
        options.timeout = void 0;
        this[kHeaders] = /* @__PURE__ */ Object.create(null);
        this[kJobs] = [];
        this[kPendingAgentPromise] = void 0;
        this.socket = null;
        this.connection = null;
        this.method = options.method || "GET";
        if (!(this.method === "CONNECT" && (options.path === "/" || options.path === void 0))) {
          this.path = options.path;
        }
        this.res = null;
        this.aborted = false;
        this.reusedSocket = false;
        const { headers } = options;
        if (headers) {
          for (const header in headers) {
            this.setHeader(header, headers[header]);
          }
        }
        if (options.auth && !("authorization" in this[kHeaders])) {
          this[kHeaders].authorization = "Basic " + Buffer.from(options.auth).toString("base64");
        }
        options.session = options.tlsSession;
        options.path = options.socketPath;
        this[kOptions] = options;
        this[kOrigin] = new URL4(`${this.protocol}//${options.servername || options.host}:${options.port}`);
        const reuseSocket = options._reuseSocket;
        if (reuseSocket) {
          options.createConnection = (...args) => {
            if (reuseSocket.destroyed) {
              return this.agent.createConnection(...args);
            }
            return reuseSocket;
          };
          this.agent.getSession(this[kOrigin], this[kOptions]).catch(() => {
          });
        }
        if (timeout) {
          this.setTimeout(timeout);
        }
        if (callback) {
          this.once("response", callback);
        }
        this[kFlushedHeaders] = false;
      }
      get method() {
        return this[kHeaders][HTTP2_HEADER_METHOD];
      }
      set method(value) {
        if (value) {
          this[kHeaders][HTTP2_HEADER_METHOD] = value.toUpperCase();
        }
      }
      get path() {
        const header = this.method === "CONNECT" ? HTTP2_HEADER_AUTHORITY : HTTP2_HEADER_PATH;
        return this[kHeaders][header];
      }
      set path(value) {
        if (value) {
          const header = this.method === "CONNECT" ? HTTP2_HEADER_AUTHORITY : HTTP2_HEADER_PATH;
          this[kHeaders][header] = value;
        }
      }
      get host() {
        return this[kOrigin].hostname;
      }
      set host(_value) {
      }
      get _mustNotHaveABody() {
        return this.method === "GET" || this.method === "HEAD" || this.method === "DELETE";
      }
      _write(chunk, encoding, callback) {
        if (this._mustNotHaveABody) {
          callback(new Error("The GET, HEAD and DELETE methods must NOT have a body"));
          return;
        }
        this.flushHeaders();
        const callWrite = () => this._request.write(chunk, encoding, callback);
        if (this._request) {
          callWrite();
        } else {
          this[kJobs].push(callWrite);
        }
      }
      _final(callback) {
        this.flushHeaders();
        const callEnd = () => {
          if (this._mustNotHaveABody || this.method === "CONNECT") {
            callback();
            return;
          }
          this._request.end(callback);
        };
        if (this._request) {
          callEnd();
        } else {
          this[kJobs].push(callEnd);
        }
      }
      abort() {
        if (this.res && this.res.complete) {
          return;
        }
        if (!this.aborted) {
          process.nextTick(() => this.emit("abort"));
        }
        this.aborted = true;
        this.destroy();
      }
      async _destroy(error, callback) {
        if (this.res) {
          this.res._dump();
        }
        if (this._request) {
          this._request.destroy();
        } else {
          process.nextTick(() => {
            this.emit("close");
          });
        }
        try {
          await this[kPendingAgentPromise];
        } catch (internalError) {
          if (this.aborted) {
            error = internalError;
          }
        }
        callback(error);
      }
      async flushHeaders() {
        if (this[kFlushedHeaders] || this.destroyed) {
          return;
        }
        this[kFlushedHeaders] = true;
        const isConnectMethod = this.method === HTTP2_METHOD_CONNECT;
        const onStream = (stream2) => {
          this._request = stream2;
          if (this.destroyed) {
            stream2.destroy();
            return;
          }
          if (!isConnectMethod) {
            proxyEvents2(stream2, this, ["timeout", "continue"]);
          }
          stream2.once("error", (error) => {
            this.destroy(error);
          });
          stream2.once("aborted", () => {
            const { res } = this;
            if (res) {
              res.aborted = true;
              res.emit("aborted");
              res.destroy();
            } else {
              this.destroy(new Error("The server aborted the HTTP/2 stream"));
            }
          });
          const onResponse = (headers, flags, rawHeaders) => {
            const response = new IncomingMessage(this.socket, stream2.readableHighWaterMark);
            this.res = response;
            response.url = `${this[kOrigin].origin}${this.path}`;
            response.req = this;
            response.statusCode = headers[HTTP2_HEADER_STATUS];
            response.headers = headers;
            response.rawHeaders = rawHeaders;
            response.once("end", () => {
              response.complete = true;
              response.socket = null;
              response.connection = null;
            });
            if (isConnectMethod) {
              response.upgrade = true;
              if (this.emit("connect", response, stream2, Buffer.alloc(0))) {
                this.emit("close");
              } else {
                stream2.destroy();
              }
            } else {
              stream2.on("data", (chunk) => {
                if (!response._dumped && !response.push(chunk)) {
                  stream2.pause();
                }
              });
              stream2.once("end", () => {
                if (!this.aborted) {
                  response.push(null);
                }
              });
              if (!this.emit("response", response)) {
                response._dump();
              }
            }
          };
          stream2.once("response", onResponse);
          stream2.once("headers", (headers) => this.emit("information", { statusCode: headers[HTTP2_HEADER_STATUS] }));
          stream2.once("trailers", (trailers, flags, rawTrailers) => {
            const { res } = this;
            if (res === null) {
              onResponse(trailers, flags, rawTrailers);
              return;
            }
            res.trailers = trailers;
            res.rawTrailers = rawTrailers;
          });
          stream2.once("close", () => {
            const { aborted, res } = this;
            if (res) {
              if (aborted) {
                res.aborted = true;
                res.emit("aborted");
                res.destroy();
              }
              const finish = () => {
                res.emit("close");
                this.destroy();
                this.emit("close");
              };
              if (res.readable) {
                res.once("end", finish);
              } else {
                finish();
              }
              return;
            }
            if (!this.destroyed) {
              this.destroy(new Error("The HTTP/2 stream has been early terminated"));
              this.emit("close");
              return;
            }
            this.destroy();
            this.emit("close");
          });
          this.socket = new Proxy(stream2, proxySocketHandler);
          for (const job of this[kJobs]) {
            job();
          }
          this[kJobs].length = 0;
          this.emit("socket", this.socket);
        };
        if (!(HTTP2_HEADER_AUTHORITY in this[kHeaders]) && !isConnectMethod) {
          this[kHeaders][HTTP2_HEADER_AUTHORITY] = this[kOrigin].host;
        }
        if (this[kSession]) {
          try {
            onStream(this[kSession].request(this[kHeaders]));
          } catch (error) {
            this.destroy(error);
          }
        } else {
          this.reusedSocket = true;
          try {
            const promise = this.agent.request(this[kOrigin], this[kOptions], this[kHeaders]);
            this[kPendingAgentPromise] = promise;
            onStream(await promise);
            this[kPendingAgentPromise] = false;
          } catch (error) {
            this[kPendingAgentPromise] = false;
            this.destroy(error);
          }
        }
      }
      get connection() {
        return this.socket;
      }
      set connection(value) {
        this.socket = value;
      }
      getHeaderNames() {
        return Object.keys(this[kHeaders]);
      }
      hasHeader(name) {
        if (typeof name !== "string") {
          throw new ERR_INVALID_ARG_TYPE("name", "string", name);
        }
        return Boolean(this[kHeaders][name.toLowerCase()]);
      }
      getHeader(name) {
        if (typeof name !== "string") {
          throw new ERR_INVALID_ARG_TYPE("name", "string", name);
        }
        return this[kHeaders][name.toLowerCase()];
      }
      get headersSent() {
        return this[kFlushedHeaders];
      }
      removeHeader(name) {
        if (typeof name !== "string") {
          throw new ERR_INVALID_ARG_TYPE("name", "string", name);
        }
        if (this.headersSent) {
          throw new ERR_HTTP_HEADERS_SENT("remove");
        }
        delete this[kHeaders][name.toLowerCase()];
      }
      setHeader(name, value) {
        if (this.headersSent) {
          throw new ERR_HTTP_HEADERS_SENT("set");
        }
        validateHeaderName(name);
        validateHeaderValue(name, value);
        const lowercased = name.toLowerCase();
        if (lowercased === "connection") {
          if (value.toLowerCase() === "keep-alive") {
            return;
          }
          throw new Error(`Invalid 'connection' header: ${value}`);
        }
        if (lowercased === "host" && this.method === "CONNECT") {
          this[kHeaders][HTTP2_HEADER_AUTHORITY] = value;
        } else {
          this[kHeaders][lowercased] = value;
        }
      }
      setNoDelay() {
      }
      setSocketKeepAlive() {
      }
      setTimeout(ms, callback) {
        const applyTimeout = () => this._request.setTimeout(ms, callback);
        if (this._request) {
          applyTimeout();
        } else {
          this[kJobs].push(applyTimeout);
        }
        return this;
      }
      get maxHeadersCount() {
        if (!this.destroyed && this._request) {
          return this._request.session.localSettings.maxHeaderListSize;
        }
        return void 0;
      }
      set maxHeadersCount(_value) {
      }
    };
    module.exports = ClientRequest;
  }
});

// node_modules/.pnpm/resolve-alpn@1.2.1/node_modules/resolve-alpn/index.js
var require_resolve_alpn = __commonJS({
  "node_modules/.pnpm/resolve-alpn@1.2.1/node_modules/resolve-alpn/index.js"(exports, module) {
    "use strict";
    var tls = __require("tls");
    module.exports = (options = {}, connect = tls.connect) => new Promise((resolve, reject) => {
      let timeout = false;
      let socket;
      const callback = async () => {
        await socketPromise;
        socket.off("timeout", onTimeout);
        socket.off("error", reject);
        if (options.resolveSocket) {
          resolve({ alpnProtocol: socket.alpnProtocol, socket, timeout });
          if (timeout) {
            await Promise.resolve();
            socket.emit("timeout");
          }
        } else {
          socket.destroy();
          resolve({ alpnProtocol: socket.alpnProtocol, timeout });
        }
      };
      const onTimeout = async () => {
        timeout = true;
        callback();
      };
      const socketPromise = (async () => {
        try {
          socket = await connect(options, callback);
          socket.on("error", reject);
          socket.once("timeout", onTimeout);
        } catch (error) {
          reject(error);
        }
      })();
    });
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/calculate-server-name.js
var require_calculate_server_name = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/calculate-server-name.js"(exports, module) {
    "use strict";
    var { isIP } = __require("net");
    var assert2 = __require("assert");
    var getHost = (host) => {
      if (host[0] === "[") {
        const idx2 = host.indexOf("]");
        assert2(idx2 !== -1);
        return host.slice(1, idx2);
      }
      const idx = host.indexOf(":");
      if (idx === -1) {
        return host;
      }
      return host.slice(0, idx);
    };
    module.exports = (host) => {
      const servername = getHost(host);
      if (isIP(servername)) {
        return "";
      }
      return servername;
    };
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/auto.js
var require_auto = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/auto.js"(exports, module) {
    "use strict";
    var { URL: URL4, urlToHttpOptions } = __require("url");
    var http3 = __require("http");
    var https2 = __require("https");
    var resolveALPN = require_resolve_alpn();
    var QuickLRU = require_quick_lru();
    var { Agent, globalAgent } = require_agent();
    var Http2ClientRequest = require_client_request();
    var calculateServerName = require_calculate_server_name();
    var delayAsyncDestroy = require_delay_async_destroy();
    var cache = new QuickLRU({ maxSize: 100 });
    var queue = /* @__PURE__ */ new Map();
    var installSocket = (agent2, socket, options) => {
      socket._httpMessage = { shouldKeepAlive: true };
      const onFree = () => {
        agent2.emit("free", socket, options);
      };
      socket.on("free", onFree);
      const onClose = () => {
        agent2.removeSocket(socket, options);
      };
      socket.on("close", onClose);
      const onTimeout = () => {
        const { freeSockets } = agent2;
        for (const sockets of Object.values(freeSockets)) {
          if (sockets.includes(socket)) {
            socket.destroy();
            return;
          }
        }
      };
      socket.on("timeout", onTimeout);
      const onRemove = () => {
        agent2.removeSocket(socket, options);
        socket.off("close", onClose);
        socket.off("free", onFree);
        socket.off("timeout", onTimeout);
        socket.off("agentRemove", onRemove);
      };
      socket.on("agentRemove", onRemove);
      agent2.emit("free", socket, options);
    };
    var createResolveProtocol = (cache2, queue2 = /* @__PURE__ */ new Map(), connect = void 0) => {
      return async (options) => {
        const name = `${options.host}:${options.port}:${options.ALPNProtocols.sort()}`;
        if (!cache2.has(name)) {
          if (queue2.has(name)) {
            const result = await queue2.get(name);
            return { alpnProtocol: result.alpnProtocol };
          }
          const { path } = options;
          options.path = options.socketPath;
          const resultPromise = resolveALPN(options, connect);
          queue2.set(name, resultPromise);
          try {
            const result = await resultPromise;
            cache2.set(name, result.alpnProtocol);
            queue2.delete(name);
            options.path = path;
            return result;
          } catch (error) {
            queue2.delete(name);
            options.path = path;
            throw error;
          }
        }
        return { alpnProtocol: cache2.get(name) };
      };
    };
    var defaultResolveProtocol = createResolveProtocol(cache, queue);
    module.exports = async (input, options, callback) => {
      if (typeof input === "string") {
        input = urlToHttpOptions(new URL4(input));
      } else if (input instanceof URL4) {
        input = urlToHttpOptions(input);
      } else {
        input = { ...input };
      }
      if (typeof options === "function" || options === void 0) {
        callback = options;
        options = input;
      } else {
        options = Object.assign(input, options);
      }
      options.ALPNProtocols = options.ALPNProtocols || ["h2", "http/1.1"];
      if (!Array.isArray(options.ALPNProtocols) || options.ALPNProtocols.length === 0) {
        throw new Error("The `ALPNProtocols` option must be an Array with at least one entry");
      }
      options.protocol = options.protocol || "https:";
      const isHttps = options.protocol === "https:";
      options.host = options.hostname || options.host || "localhost";
      options.session = options.tlsSession;
      options.servername = options.servername || calculateServerName(options.headers && options.headers.host || options.host);
      options.port = options.port || (isHttps ? 443 : 80);
      options._defaultAgent = isHttps ? https2.globalAgent : http3.globalAgent;
      const resolveProtocol = options.resolveProtocol || defaultResolveProtocol;
      let { agent: agent2 } = options;
      if (agent2 !== void 0 && agent2 !== false && agent2.constructor.name !== "Object") {
        throw new Error("The `options.agent` can be only an object `http`, `https` or `http2` properties");
      }
      if (isHttps) {
        options.resolveSocket = true;
        let { socket, alpnProtocol, timeout } = await resolveProtocol(options);
        if (timeout) {
          if (socket) {
            socket.destroy();
          }
          const error = new Error(`Timed out resolving ALPN: ${options.timeout} ms`);
          error.code = "ETIMEDOUT";
          error.ms = options.timeout;
          throw error;
        }
        if (socket && options.createConnection) {
          socket.destroy();
          socket = void 0;
        }
        delete options.resolveSocket;
        const isHttp2 = alpnProtocol === "h2";
        if (agent2) {
          agent2 = isHttp2 ? agent2.http2 : agent2.https;
          options.agent = agent2;
        }
        if (agent2 === void 0) {
          agent2 = isHttp2 ? globalAgent : https2.globalAgent;
        }
        if (socket) {
          if (agent2 === false) {
            socket.destroy();
          } else {
            const defaultCreateConnection = (isHttp2 ? Agent : https2.Agent).prototype.createConnection;
            if (agent2.createConnection === defaultCreateConnection) {
              if (isHttp2) {
                options._reuseSocket = socket;
              } else {
                installSocket(agent2, socket, options);
              }
            } else {
              socket.destroy();
            }
          }
        }
        if (isHttp2) {
          return delayAsyncDestroy(new Http2ClientRequest(options, callback));
        }
      } else if (agent2) {
        options.agent = agent2.http;
      }
      return delayAsyncDestroy(http3.request(options, callback));
    };
    module.exports.protocolCache = cache;
    module.exports.resolveProtocol = defaultResolveProtocol;
    module.exports.createResolveProtocol = createResolveProtocol;
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/js-stream-socket.js
var require_js_stream_socket = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/js-stream-socket.js"(exports, module) {
    "use strict";
    var stream2 = __require("stream");
    var tls = __require("tls");
    var JSStreamSocket = new tls.TLSSocket(new stream2.PassThrough())._handle._parentWrap.constructor;
    module.exports = JSStreamSocket;
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/proxies/unexpected-status-code-error.js
var require_unexpected_status_code_error = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/proxies/unexpected-status-code-error.js"(exports, module) {
    "use strict";
    var UnexpectedStatusCodeError = class extends Error {
      constructor(statusCode, statusMessage = "") {
        super(`The proxy server rejected the request with status code ${statusCode} (${statusMessage || "empty status message"})`);
        this.statusCode = statusCode;
        this.statusMessage = statusMessage;
      }
    };
    module.exports = UnexpectedStatusCodeError;
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/check-type.js
var require_check_type = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/utils/check-type.js"(exports, module) {
    "use strict";
    var checkType = (name, value, types2) => {
      const valid = types2.some((type) => {
        const typeofType = typeof type;
        if (typeofType === "string") {
          return typeof value === type;
        }
        return value instanceof type;
      });
      if (!valid) {
        const names = types2.map((type) => typeof type === "string" ? type : type.name);
        throw new TypeError(`Expected '${name}' to be a type of ${names.join(" or ")}, got ${typeof value}`);
      }
    };
    module.exports = checkType;
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/proxies/initialize.js
var require_initialize = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/proxies/initialize.js"(exports, module) {
    "use strict";
    var { URL: URL4 } = __require("url");
    var checkType = require_check_type();
    module.exports = (self, proxyOptions) => {
      checkType("proxyOptions", proxyOptions, ["object"]);
      checkType("proxyOptions.headers", proxyOptions.headers, ["object", "undefined"]);
      checkType("proxyOptions.raw", proxyOptions.raw, ["boolean", "undefined"]);
      checkType("proxyOptions.url", proxyOptions.url, [URL4, "string"]);
      const url = new URL4(proxyOptions.url);
      self.proxyOptions = {
        raw: true,
        ...proxyOptions,
        headers: { ...proxyOptions.headers },
        url
      };
    };
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/proxies/get-auth-headers.js
var require_get_auth_headers = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/proxies/get-auth-headers.js"(exports, module) {
    "use strict";
    module.exports = (self) => {
      const { username, password } = self.proxyOptions.url;
      if (username || password) {
        const data = `${username}:${password}`;
        const authorization = `Basic ${Buffer.from(data).toString("base64")}`;
        return {
          "proxy-authorization": authorization,
          authorization
        };
      }
      return {};
    };
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/proxies/h1-over-h2.js
var require_h1_over_h2 = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/proxies/h1-over-h2.js"(exports, module) {
    "use strict";
    var tls = __require("tls");
    var http3 = __require("http");
    var https2 = __require("https");
    var JSStreamSocket = require_js_stream_socket();
    var { globalAgent } = require_agent();
    var UnexpectedStatusCodeError = require_unexpected_status_code_error();
    var initialize = require_initialize();
    var getAuthorizationHeaders = require_get_auth_headers();
    var createConnection = (self, options, callback) => {
      (async () => {
        try {
          const { proxyOptions } = self;
          const { url, headers, raw } = proxyOptions;
          const stream2 = await globalAgent.request(url, proxyOptions, {
            ...getAuthorizationHeaders(self),
            ...headers,
            ":method": "CONNECT",
            ":authority": `${options.host}:${options.port}`
          });
          stream2.once("error", callback);
          stream2.once("response", (headers2) => {
            const statusCode = headers2[":status"];
            if (statusCode !== 200) {
              callback(new UnexpectedStatusCodeError(statusCode, ""));
              return;
            }
            const encrypted = self instanceof https2.Agent;
            if (raw && encrypted) {
              options.socket = stream2;
              const secureStream = tls.connect(options);
              secureStream.once("close", () => {
                stream2.destroy();
              });
              callback(null, secureStream);
              return;
            }
            const socket = new JSStreamSocket(stream2);
            socket.encrypted = false;
            socket._handle.getpeername = (out) => {
              out.family = void 0;
              out.address = void 0;
              out.port = void 0;
            };
            callback(null, socket);
          });
        } catch (error) {
          callback(error);
        }
      })();
    };
    var HttpOverHttp2 = class extends http3.Agent {
      constructor(options) {
        super(options);
        initialize(this, options.proxyOptions);
      }
      createConnection(options, callback) {
        createConnection(this, options, callback);
      }
    };
    var HttpsOverHttp2 = class extends https2.Agent {
      constructor(options) {
        super(options);
        initialize(this, options.proxyOptions);
      }
      createConnection(options, callback) {
        createConnection(this, options, callback);
      }
    };
    module.exports = {
      HttpOverHttp2,
      HttpsOverHttp2
    };
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/proxies/h2-over-hx.js
var require_h2_over_hx = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/proxies/h2-over-hx.js"(exports, module) {
    "use strict";
    var { Agent } = require_agent();
    var JSStreamSocket = require_js_stream_socket();
    var UnexpectedStatusCodeError = require_unexpected_status_code_error();
    var initialize = require_initialize();
    var Http2OverHttpX = class extends Agent {
      constructor(options) {
        super(options);
        initialize(this, options.proxyOptions);
      }
      async createConnection(origin, options) {
        const authority = `${origin.hostname}:${origin.port || 443}`;
        const [stream2, statusCode, statusMessage] = await this._getProxyStream(authority);
        if (statusCode !== 200) {
          throw new UnexpectedStatusCodeError(statusCode, statusMessage);
        }
        if (this.proxyOptions.raw) {
          options.socket = stream2;
        } else {
          const socket = new JSStreamSocket(stream2);
          socket.encrypted = false;
          socket._handle.getpeername = (out) => {
            out.family = void 0;
            out.address = void 0;
            out.port = void 0;
          };
          return socket;
        }
        return super.createConnection(origin, options);
      }
    };
    module.exports = Http2OverHttpX;
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/proxies/h2-over-h2.js
var require_h2_over_h2 = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/proxies/h2-over-h2.js"(exports, module) {
    "use strict";
    var { globalAgent } = require_agent();
    var Http2OverHttpX = require_h2_over_hx();
    var getAuthorizationHeaders = require_get_auth_headers();
    var getStatusCode = (stream2) => new Promise((resolve, reject) => {
      stream2.once("error", reject);
      stream2.once("response", (headers) => {
        stream2.off("error", reject);
        resolve(headers[":status"]);
      });
    });
    var Http2OverHttp2 = class extends Http2OverHttpX {
      async _getProxyStream(authority) {
        const { proxyOptions } = this;
        const headers = {
          ...getAuthorizationHeaders(this),
          ...proxyOptions.headers,
          ":method": "CONNECT",
          ":authority": authority
        };
        const stream2 = await globalAgent.request(proxyOptions.url, proxyOptions, headers);
        const statusCode = await getStatusCode(stream2);
        return [stream2, statusCode, ""];
      }
    };
    module.exports = Http2OverHttp2;
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/proxies/h2-over-h1.js
var require_h2_over_h1 = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/proxies/h2-over-h1.js"(exports, module) {
    "use strict";
    var http3 = __require("http");
    var https2 = __require("https");
    var Http2OverHttpX = require_h2_over_hx();
    var getAuthorizationHeaders = require_get_auth_headers();
    var getStream2 = (request) => new Promise((resolve, reject) => {
      const onConnect = (response, socket, head) => {
        socket.unshift(head);
        request.off("error", reject);
        resolve([socket, response.statusCode, response.statusMessage]);
      };
      request.once("error", reject);
      request.once("connect", onConnect);
    });
    var Http2OverHttp = class extends Http2OverHttpX {
      async _getProxyStream(authority) {
        const { proxyOptions } = this;
        const { url, headers } = this.proxyOptions;
        const network = url.protocol === "https:" ? https2 : http3;
        const request = network.request({
          ...proxyOptions,
          hostname: url.hostname,
          port: url.port,
          path: authority,
          headers: {
            ...getAuthorizationHeaders(this),
            ...headers,
            host: authority
          },
          method: "CONNECT"
        }).end();
        return getStream2(request);
      }
    };
    module.exports = {
      Http2OverHttp,
      Http2OverHttps: Http2OverHttp
    };
  }
});

// node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/index.js
var require_source2 = __commonJS({
  "node_modules/.pnpm/http2-wrapper@2.2.0/node_modules/http2-wrapper/source/index.js"(exports, module) {
    "use strict";
    var http22 = __require("http2");
    var {
      Agent,
      globalAgent
    } = require_agent();
    var ClientRequest = require_client_request();
    var IncomingMessage = require_incoming_message();
    var auto = require_auto();
    var {
      HttpOverHttp2,
      HttpsOverHttp2
    } = require_h1_over_h2();
    var Http2OverHttp2 = require_h2_over_h2();
    var {
      Http2OverHttp,
      Http2OverHttps
    } = require_h2_over_h1();
    var validateHeaderName = require_validate_header_name();
    var validateHeaderValue = require_validate_header_value();
    var request = (url, options, callback) => new ClientRequest(url, options, callback);
    var get = (url, options, callback) => {
      const req = new ClientRequest(url, options, callback);
      req.end();
      return req;
    };
    module.exports = {
      ...http22,
      ClientRequest,
      IncomingMessage,
      Agent,
      globalAgent,
      request,
      get,
      auto,
      proxies: {
        HttpOverHttp2,
        HttpsOverHttp2,
        Http2OverHttp2,
        Http2OverHttp,
        Http2OverHttps
      },
      validateHeaderName,
      validateHeaderValue
    };
  }
});

// src/index.ts
import { Agent as HttpAgent } from "http";
import { Agent as HttpsAgent } from "https";

// node_modules/.pnpm/@sindresorhus+is@5.3.0/node_modules/@sindresorhus/is/dist/index.js
var typedArrayTypeNames = [
  "Int8Array",
  "Uint8Array",
  "Uint8ClampedArray",
  "Int16Array",
  "Uint16Array",
  "Int32Array",
  "Uint32Array",
  "Float32Array",
  "Float64Array",
  "BigInt64Array",
  "BigUint64Array"
];
function isTypedArrayName(name) {
  return typedArrayTypeNames.includes(name);
}
var objectTypeNames = [
  "Function",
  "Generator",
  "AsyncGenerator",
  "GeneratorFunction",
  "AsyncGeneratorFunction",
  "AsyncFunction",
  "Observable",
  "Array",
  "Buffer",
  "Blob",
  "Object",
  "RegExp",
  "Date",
  "Error",
  "Map",
  "Set",
  "WeakMap",
  "WeakSet",
  "WeakRef",
  "ArrayBuffer",
  "SharedArrayBuffer",
  "DataView",
  "Promise",
  "URL",
  "FormData",
  "URLSearchParams",
  "HTMLElement",
  "NaN",
  ...typedArrayTypeNames
];
function isObjectTypeName(name) {
  return objectTypeNames.includes(name);
}
var primitiveTypeNames = [
  "null",
  "undefined",
  "string",
  "number",
  "bigint",
  "boolean",
  "symbol"
];
function isPrimitiveTypeName(name) {
  return primitiveTypeNames.includes(name);
}
function isOfType(type) {
  return (value) => typeof value === type;
}
var { toString } = Object.prototype;
var getObjectType = (value) => {
  const objectTypeName = toString.call(value).slice(8, -1);
  if (/HTML\w+Element/.test(objectTypeName) && is.domElement(value)) {
    return "HTMLElement";
  }
  if (isObjectTypeName(objectTypeName)) {
    return objectTypeName;
  }
  return void 0;
};
var isObjectOfType = (type) => (value) => getObjectType(value) === type;
function is(value) {
  if (value === null) {
    return "null";
  }
  switch (typeof value) {
    case "undefined":
      return "undefined";
    case "string":
      return "string";
    case "number":
      return Number.isNaN(value) ? "NaN" : "number";
    case "boolean":
      return "boolean";
    case "function":
      return "Function";
    case "bigint":
      return "bigint";
    case "symbol":
      return "symbol";
    default:
  }
  if (is.observable(value)) {
    return "Observable";
  }
  if (is.array(value)) {
    return "Array";
  }
  if (is.buffer(value)) {
    return "Buffer";
  }
  const tagType = getObjectType(value);
  if (tagType) {
    return tagType;
  }
  if (value instanceof String || value instanceof Boolean || value instanceof Number) {
    throw new TypeError("Please don't use object wrappers for primitive types");
  }
  return "Object";
}
is.undefined = isOfType("undefined");
is.string = isOfType("string");
var isNumberType = isOfType("number");
is.number = (value) => isNumberType(value) && !is.nan(value);
is.bigint = isOfType("bigint");
is.function_ = isOfType("function");
is.null_ = (value) => value === null;
is.class_ = (value) => is.function_(value) && value.toString().startsWith("class ");
is.boolean = (value) => value === true || value === false;
is.symbol = isOfType("symbol");
is.numericString = (value) => is.string(value) && !is.emptyStringOrWhitespace(value) && !Number.isNaN(Number(value));
is.array = (value, assertion) => {
  if (!Array.isArray(value)) {
    return false;
  }
  if (!is.function_(assertion)) {
    return true;
  }
  return value.every((element) => assertion(element));
};
is.buffer = (value) => {
  var _a, _b;
  return ((_b = (_a = value == null ? void 0 : value.constructor) == null ? void 0 : _a.isBuffer) == null ? void 0 : _b.call(_a, value)) ?? false;
};
is.blob = (value) => isObjectOfType("Blob")(value);
is.nullOrUndefined = (value) => is.null_(value) || is.undefined(value);
is.object = (value) => !is.null_(value) && (typeof value === "object" || is.function_(value));
is.iterable = (value) => is.function_(value == null ? void 0 : value[Symbol.iterator]);
is.asyncIterable = (value) => is.function_(value == null ? void 0 : value[Symbol.asyncIterator]);
is.generator = (value) => is.iterable(value) && is.function_(value == null ? void 0 : value.next) && is.function_(value == null ? void 0 : value.throw);
is.asyncGenerator = (value) => is.asyncIterable(value) && is.function_(value.next) && is.function_(value.throw);
is.nativePromise = (value) => isObjectOfType("Promise")(value);
var hasPromiseApi = (value) => is.function_(value == null ? void 0 : value.then) && is.function_(value == null ? void 0 : value.catch);
is.promise = (value) => is.nativePromise(value) || hasPromiseApi(value);
is.generatorFunction = isObjectOfType("GeneratorFunction");
is.asyncGeneratorFunction = (value) => getObjectType(value) === "AsyncGeneratorFunction";
is.asyncFunction = (value) => getObjectType(value) === "AsyncFunction";
is.boundFunction = (value) => is.function_(value) && !value.hasOwnProperty("prototype");
is.regExp = isObjectOfType("RegExp");
is.date = isObjectOfType("Date");
is.error = isObjectOfType("Error");
is.map = (value) => isObjectOfType("Map")(value);
is.set = (value) => isObjectOfType("Set")(value);
is.weakMap = (value) => isObjectOfType("WeakMap")(value);
is.weakSet = (value) => isObjectOfType("WeakSet")(value);
is.weakRef = (value) => isObjectOfType("WeakRef")(value);
is.int8Array = isObjectOfType("Int8Array");
is.uint8Array = isObjectOfType("Uint8Array");
is.uint8ClampedArray = isObjectOfType("Uint8ClampedArray");
is.int16Array = isObjectOfType("Int16Array");
is.uint16Array = isObjectOfType("Uint16Array");
is.int32Array = isObjectOfType("Int32Array");
is.uint32Array = isObjectOfType("Uint32Array");
is.float32Array = isObjectOfType("Float32Array");
is.float64Array = isObjectOfType("Float64Array");
is.bigInt64Array = isObjectOfType("BigInt64Array");
is.bigUint64Array = isObjectOfType("BigUint64Array");
is.arrayBuffer = isObjectOfType("ArrayBuffer");
is.sharedArrayBuffer = isObjectOfType("SharedArrayBuffer");
is.dataView = isObjectOfType("DataView");
is.enumCase = (value, targetEnum) => Object.values(targetEnum).includes(value);
is.directInstanceOf = (instance, class_) => Object.getPrototypeOf(instance) === class_.prototype;
is.urlInstance = (value) => isObjectOfType("URL")(value);
is.urlString = (value) => {
  if (!is.string(value)) {
    return false;
  }
  try {
    new URL(value);
    return true;
  } catch {
    return false;
  }
};
is.truthy = (value) => Boolean(value);
is.falsy = (value) => !value;
is.nan = (value) => Number.isNaN(value);
is.primitive = (value) => is.null_(value) || isPrimitiveTypeName(typeof value);
is.integer = (value) => Number.isInteger(value);
is.safeInteger = (value) => Number.isSafeInteger(value);
is.plainObject = (value) => {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const prototype = Object.getPrototypeOf(value);
  return (prototype === null || prototype === Object.prototype || Object.getPrototypeOf(prototype) === null) && !(Symbol.toStringTag in value) && !(Symbol.iterator in value);
};
is.typedArray = (value) => isTypedArrayName(getObjectType(value));
var isValidLength = (value) => is.safeInteger(value) && value >= 0;
is.arrayLike = (value) => !is.nullOrUndefined(value) && !is.function_(value) && isValidLength(value.length);
is.inRange = (value, range) => {
  if (is.number(range)) {
    return value >= Math.min(0, range) && value <= Math.max(range, 0);
  }
  if (is.array(range) && range.length === 2) {
    return value >= Math.min(...range) && value <= Math.max(...range);
  }
  throw new TypeError(`Invalid range: ${JSON.stringify(range)}`);
};
var NODE_TYPE_ELEMENT = 1;
var DOM_PROPERTIES_TO_CHECK = [
  "innerHTML",
  "ownerDocument",
  "style",
  "attributes",
  "nodeValue"
];
is.domElement = (value) => is.object(value) && value.nodeType === NODE_TYPE_ELEMENT && is.string(value.nodeName) && !is.plainObject(value) && DOM_PROPERTIES_TO_CHECK.every((property) => property in value);
is.observable = (value) => {
  var _a, _b;
  if (!value) {
    return false;
  }
  if (value === ((_a = value[Symbol.observable]) == null ? void 0 : _a.call(value))) {
    return true;
  }
  if (value === ((_b = value["@@observable"]) == null ? void 0 : _b.call(value))) {
    return true;
  }
  return false;
};
is.nodeStream = (value) => is.object(value) && is.function_(value.pipe) && !is.observable(value);
is.infinite = (value) => value === Number.POSITIVE_INFINITY || value === Number.NEGATIVE_INFINITY;
var isAbsoluteMod2 = (remainder) => (value) => is.integer(value) && Math.abs(value % 2) === remainder;
is.evenInteger = isAbsoluteMod2(0);
is.oddInteger = isAbsoluteMod2(1);
is.emptyArray = (value) => is.array(value) && value.length === 0;
is.nonEmptyArray = (value) => is.array(value) && value.length > 0;
is.emptyString = (value) => is.string(value) && value.length === 0;
var isWhiteSpaceString = (value) => is.string(value) && !/\S/.test(value);
is.emptyStringOrWhitespace = (value) => is.emptyString(value) || isWhiteSpaceString(value);
is.nonEmptyString = (value) => is.string(value) && value.length > 0;
is.nonEmptyStringAndNotWhitespace = (value) => is.string(value) && !is.emptyStringOrWhitespace(value);
is.emptyObject = (value) => is.object(value) && !is.map(value) && !is.set(value) && Object.keys(value).length === 0;
is.nonEmptyObject = (value) => is.object(value) && !is.map(value) && !is.set(value) && Object.keys(value).length > 0;
is.emptySet = (value) => is.set(value) && value.size === 0;
is.nonEmptySet = (value) => is.set(value) && value.size > 0;
is.emptyMap = (value) => is.map(value) && value.size === 0;
is.nonEmptyMap = (value) => is.map(value) && value.size > 0;
is.propertyKey = (value) => is.any([is.string, is.number, is.symbol], value);
is.formData = (value) => isObjectOfType("FormData")(value);
is.urlSearchParams = (value) => isObjectOfType("URLSearchParams")(value);
var predicateOnArray = (method, predicate, values) => {
  if (!is.function_(predicate)) {
    throw new TypeError(`Invalid predicate: ${JSON.stringify(predicate)}`);
  }
  if (values.length === 0) {
    throw new TypeError("Invalid number of values");
  }
  return method.call(values, predicate);
};
is.any = (predicate, ...values) => {
  const predicates = is.array(predicate) ? predicate : [predicate];
  return predicates.some((singlePredicate) => predicateOnArray(Array.prototype.some, singlePredicate, values));
};
is.all = (predicate, ...values) => predicateOnArray(Array.prototype.every, predicate, values);
var assertType = (condition, description, value, options = {}) => {
  if (!condition) {
    const { multipleValues } = options;
    const valuesMessage = multipleValues ? `received values of types ${[
      ...new Set(value.map((singleValue) => `\`${is(singleValue)}\``))
    ].join(", ")}` : `received value of type \`${is(value)}\``;
    throw new TypeError(`Expected value which is \`${description}\`, ${valuesMessage}.`);
  }
};
var assert = {
  // Unknowns.
  undefined: (value) => assertType(is.undefined(value), "undefined", value),
  string: (value) => assertType(is.string(value), "string", value),
  number: (value) => assertType(is.number(value), "number", value),
  bigint: (value) => assertType(is.bigint(value), "bigint", value),
  // eslint-disable-next-line @typescript-eslint/ban-types
  function_: (value) => assertType(is.function_(value), "Function", value),
  null_: (value) => assertType(is.null_(value), "null", value),
  class_: (value) => assertType(is.class_(value), "Class", value),
  boolean: (value) => assertType(is.boolean(value), "boolean", value),
  symbol: (value) => assertType(is.symbol(value), "symbol", value),
  numericString: (value) => assertType(is.numericString(value), "string with a number", value),
  array: (value, assertion) => {
    const assert2 = assertType;
    assert2(is.array(value), "Array", value);
    if (assertion) {
      value.forEach(assertion);
    }
  },
  buffer: (value) => assertType(is.buffer(value), "Buffer", value),
  blob: (value) => assertType(is.blob(value), "Blob", value),
  nullOrUndefined: (value) => assertType(is.nullOrUndefined(value), "null or undefined", value),
  object: (value) => assertType(is.object(value), "Object", value),
  iterable: (value) => assertType(is.iterable(value), "Iterable", value),
  asyncIterable: (value) => assertType(is.asyncIterable(value), "AsyncIterable", value),
  generator: (value) => assertType(is.generator(value), "Generator", value),
  asyncGenerator: (value) => assertType(is.asyncGenerator(value), "AsyncGenerator", value),
  nativePromise: (value) => assertType(is.nativePromise(value), "native Promise", value),
  promise: (value) => assertType(is.promise(value), "Promise", value),
  generatorFunction: (value) => assertType(is.generatorFunction(value), "GeneratorFunction", value),
  asyncGeneratorFunction: (value) => assertType(is.asyncGeneratorFunction(value), "AsyncGeneratorFunction", value),
  // eslint-disable-next-line @typescript-eslint/ban-types
  asyncFunction: (value) => assertType(is.asyncFunction(value), "AsyncFunction", value),
  // eslint-disable-next-line @typescript-eslint/ban-types
  boundFunction: (value) => assertType(is.boundFunction(value), "Function", value),
  regExp: (value) => assertType(is.regExp(value), "RegExp", value),
  date: (value) => assertType(is.date(value), "Date", value),
  error: (value) => assertType(is.error(value), "Error", value),
  map: (value) => assertType(is.map(value), "Map", value),
  set: (value) => assertType(is.set(value), "Set", value),
  weakMap: (value) => assertType(is.weakMap(value), "WeakMap", value),
  weakSet: (value) => assertType(is.weakSet(value), "WeakSet", value),
  weakRef: (value) => assertType(is.weakRef(value), "WeakRef", value),
  int8Array: (value) => assertType(is.int8Array(value), "Int8Array", value),
  uint8Array: (value) => assertType(is.uint8Array(value), "Uint8Array", value),
  uint8ClampedArray: (value) => assertType(is.uint8ClampedArray(value), "Uint8ClampedArray", value),
  int16Array: (value) => assertType(is.int16Array(value), "Int16Array", value),
  uint16Array: (value) => assertType(is.uint16Array(value), "Uint16Array", value),
  int32Array: (value) => assertType(is.int32Array(value), "Int32Array", value),
  uint32Array: (value) => assertType(is.uint32Array(value), "Uint32Array", value),
  float32Array: (value) => assertType(is.float32Array(value), "Float32Array", value),
  float64Array: (value) => assertType(is.float64Array(value), "Float64Array", value),
  bigInt64Array: (value) => assertType(is.bigInt64Array(value), "BigInt64Array", value),
  bigUint64Array: (value) => assertType(is.bigUint64Array(value), "BigUint64Array", value),
  arrayBuffer: (value) => assertType(is.arrayBuffer(value), "ArrayBuffer", value),
  sharedArrayBuffer: (value) => assertType(is.sharedArrayBuffer(value), "SharedArrayBuffer", value),
  dataView: (value) => assertType(is.dataView(value), "DataView", value),
  enumCase: (value, targetEnum) => assertType(is.enumCase(value, targetEnum), "EnumCase", value),
  urlInstance: (value) => assertType(is.urlInstance(value), "URL", value),
  urlString: (value) => assertType(is.urlString(value), "string with a URL", value),
  truthy: (value) => assertType(is.truthy(value), "truthy", value),
  falsy: (value) => assertType(is.falsy(value), "falsy", value),
  nan: (value) => assertType(is.nan(value), "NaN", value),
  primitive: (value) => assertType(is.primitive(value), "primitive", value),
  integer: (value) => assertType(is.integer(value), "integer", value),
  safeInteger: (value) => assertType(is.safeInteger(value), "integer", value),
  plainObject: (value) => assertType(is.plainObject(value), "plain object", value),
  typedArray: (value) => assertType(is.typedArray(value), "TypedArray", value),
  arrayLike: (value) => assertType(is.arrayLike(value), "array-like", value),
  domElement: (value) => assertType(is.domElement(value), "HTMLElement", value),
  observable: (value) => assertType(is.observable(value), "Observable", value),
  nodeStream: (value) => assertType(is.nodeStream(value), "Node.js Stream", value),
  infinite: (value) => assertType(is.infinite(value), "infinite number", value),
  emptyArray: (value) => assertType(is.emptyArray(value), "empty array", value),
  nonEmptyArray: (value) => assertType(is.nonEmptyArray(value), "non-empty array", value),
  emptyString: (value) => assertType(is.emptyString(value), "empty string", value),
  emptyStringOrWhitespace: (value) => assertType(is.emptyStringOrWhitespace(value), "empty string or whitespace", value),
  nonEmptyString: (value) => assertType(is.nonEmptyString(value), "non-empty string", value),
  nonEmptyStringAndNotWhitespace: (value) => assertType(is.nonEmptyStringAndNotWhitespace(value), "non-empty string and not whitespace", value),
  emptyObject: (value) => assertType(is.emptyObject(value), "empty object", value),
  nonEmptyObject: (value) => assertType(is.nonEmptyObject(value), "non-empty object", value),
  emptySet: (value) => assertType(is.emptySet(value), "empty set", value),
  nonEmptySet: (value) => assertType(is.nonEmptySet(value), "non-empty set", value),
  emptyMap: (value) => assertType(is.emptyMap(value), "empty map", value),
  nonEmptyMap: (value) => assertType(is.nonEmptyMap(value), "non-empty map", value),
  propertyKey: (value) => assertType(is.propertyKey(value), "PropertyKey", value),
  formData: (value) => assertType(is.formData(value), "FormData", value),
  urlSearchParams: (value) => assertType(is.urlSearchParams(value), "URLSearchParams", value),
  // Numbers.
  evenInteger: (value) => assertType(is.evenInteger(value), "even integer", value),
  oddInteger: (value) => assertType(is.oddInteger(value), "odd integer", value),
  // Two arguments.
  directInstanceOf: (instance, class_) => assertType(is.directInstanceOf(instance, class_), "T", instance),
  inRange: (value, range) => assertType(is.inRange(value, range), "in range", value),
  // Variadic functions.
  any: (predicate, ...values) => assertType(is.any(predicate, ...values), "predicate returns truthy for any value", values, { multipleValues: true }),
  all: (predicate, ...values) => assertType(is.all(predicate, ...values), "predicate returns truthy for all values", values, { multipleValues: true })
};
Object.defineProperties(is, {
  class: {
    value: is.class_
  },
  function: {
    value: is.function_
  },
  null: {
    value: is.null_
  }
});
Object.defineProperties(assert, {
  class: {
    value: assert.class_
  },
  function: {
    value: assert.function_
  },
  null: {
    value: assert.null_
  }
});
var dist_default = is;

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/as-promise/index.js
import { EventEmitter as EventEmitter2 } from "node:events";

// node_modules/.pnpm/p-cancelable@3.0.0/node_modules/p-cancelable/index.js
var CancelError = class extends Error {
  constructor(reason) {
    super(reason || "Promise was canceled");
    this.name = "CancelError";
  }
  get isCanceled() {
    return true;
  }
};
var PCancelable = class {
  static fn(userFunction) {
    return (...arguments_) => {
      return new PCancelable((resolve, reject, onCancel) => {
        arguments_.push(onCancel);
        userFunction(...arguments_).then(resolve, reject);
      });
    };
  }
  constructor(executor) {
    this._cancelHandlers = [];
    this._isPending = true;
    this._isCanceled = false;
    this._rejectOnCancel = true;
    this._promise = new Promise((resolve, reject) => {
      this._reject = reject;
      const onResolve = (value) => {
        if (!this._isCanceled || !onCancel.shouldReject) {
          this._isPending = false;
          resolve(value);
        }
      };
      const onReject = (error) => {
        this._isPending = false;
        reject(error);
      };
      const onCancel = (handler) => {
        if (!this._isPending) {
          throw new Error("The `onCancel` handler was attached after the promise settled.");
        }
        this._cancelHandlers.push(handler);
      };
      Object.defineProperties(onCancel, {
        shouldReject: {
          get: () => this._rejectOnCancel,
          set: (boolean) => {
            this._rejectOnCancel = boolean;
          }
        }
      });
      executor(onResolve, onReject, onCancel);
    });
  }
  then(onFulfilled, onRejected) {
    return this._promise.then(onFulfilled, onRejected);
  }
  catch(onRejected) {
    return this._promise.catch(onRejected);
  }
  finally(onFinally) {
    return this._promise.finally(onFinally);
  }
  cancel(reason) {
    if (!this._isPending || this._isCanceled) {
      return;
    }
    this._isCanceled = true;
    if (this._cancelHandlers.length > 0) {
      try {
        for (const handler of this._cancelHandlers) {
          handler();
        }
      } catch (error) {
        this._reject(error);
        return;
      }
    }
    if (this._rejectOnCancel) {
      this._reject(new CancelError(reason));
    }
  }
  get isCanceled() {
    return this._isCanceled;
  }
};
Object.setPrototypeOf(PCancelable.prototype, Promise.prototype);

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/errors.js
function isRequest(x) {
  return dist_default.object(x) && "_onResponse" in x;
}
var RequestError = class extends Error {
  constructor(message, error, self) {
    var _a;
    super(message);
    Object.defineProperty(this, "input", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "code", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "stack", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "response", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "request", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "timings", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Error.captureStackTrace(this, this.constructor);
    this.name = "RequestError";
    this.code = error.code ?? "ERR_GOT_REQUEST_ERROR";
    this.input = error.input;
    if (isRequest(self)) {
      Object.defineProperty(this, "request", {
        enumerable: false,
        value: self
      });
      Object.defineProperty(this, "response", {
        enumerable: false,
        value: self.response
      });
      this.options = self.options;
    } else {
      this.options = self;
    }
    this.timings = (_a = this.request) == null ? void 0 : _a.timings;
    if (dist_default.string(error.stack) && dist_default.string(this.stack)) {
      const indexOfMessage = this.stack.indexOf(this.message) + this.message.length;
      const thisStackTrace = this.stack.slice(indexOfMessage).split("\n").reverse();
      const errorStackTrace = error.stack.slice(error.stack.indexOf(error.message) + error.message.length).split("\n").reverse();
      while (errorStackTrace.length > 0 && errorStackTrace[0] === thisStackTrace[0]) {
        thisStackTrace.shift();
      }
      this.stack = `${this.stack.slice(0, indexOfMessage)}${thisStackTrace.reverse().join("\n")}${errorStackTrace.reverse().join("\n")}`;
    }
  }
};
var MaxRedirectsError = class extends RequestError {
  constructor(request) {
    super(`Redirected ${request.options.maxRedirects} times. Aborting.`, {}, request);
    this.name = "MaxRedirectsError";
    this.code = "ERR_TOO_MANY_REDIRECTS";
  }
};
var HTTPError = class extends RequestError {
  constructor(response) {
    super(`Response code ${response.statusCode} (${response.statusMessage})`, {}, response.request);
    this.name = "HTTPError";
    this.code = "ERR_NON_2XX_3XX_RESPONSE";
  }
};
var CacheError = class extends RequestError {
  constructor(error, request) {
    super(error.message, error, request);
    this.name = "CacheError";
    this.code = this.code === "ERR_GOT_REQUEST_ERROR" ? "ERR_CACHE_ACCESS" : this.code;
  }
};
var UploadError = class extends RequestError {
  constructor(error, request) {
    super(error.message, error, request);
    this.name = "UploadError";
    this.code = this.code === "ERR_GOT_REQUEST_ERROR" ? "ERR_UPLOAD" : this.code;
  }
};
var TimeoutError = class extends RequestError {
  constructor(error, timings, request) {
    super(error.message, error, request);
    Object.defineProperty(this, "timings", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "event", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    this.name = "TimeoutError";
    this.event = error.event;
    this.timings = timings;
  }
};
var ReadError = class extends RequestError {
  constructor(error, request) {
    super(error.message, error, request);
    this.name = "ReadError";
    this.code = this.code === "ERR_GOT_REQUEST_ERROR" ? "ERR_READING_RESPONSE_STREAM" : this.code;
  }
};
var RetryError = class extends RequestError {
  constructor(request) {
    super("Retrying", {}, request);
    this.name = "RetryError";
    this.code = "ERR_RETRYING";
  }
};
var AbortError = class extends RequestError {
  constructor(request) {
    super("This operation was aborted.", {}, request);
    this.code = "ERR_ABORTED";
    this.name = "AbortError";
  }
};

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/index.js
import process3 from "node:process";
import { Buffer as Buffer3 } from "node:buffer";
import { Duplex } from "node:stream";
import { URL as URL3, URLSearchParams as URLSearchParams2 } from "node:url";
import http2, { ServerResponse } from "node:http";

// node_modules/.pnpm/@szmarczak+http-timer@5.0.1/node_modules/@szmarczak/http-timer/dist/source/index.js
var import_defer_to_connect = __toESM(require_source(), 1);
import { errorMonitor } from "events";
import { types } from "util";
var timer = (request) => {
  if (request.timings) {
    return request.timings;
  }
  const timings = {
    start: Date.now(),
    socket: void 0,
    lookup: void 0,
    connect: void 0,
    secureConnect: void 0,
    upload: void 0,
    response: void 0,
    end: void 0,
    error: void 0,
    abort: void 0,
    phases: {
      wait: void 0,
      dns: void 0,
      tcp: void 0,
      tls: void 0,
      request: void 0,
      firstByte: void 0,
      download: void 0,
      total: void 0
    }
  };
  request.timings = timings;
  const handleError = (origin) => {
    origin.once(errorMonitor, () => {
      timings.error = Date.now();
      timings.phases.total = timings.error - timings.start;
    });
  };
  handleError(request);
  const onAbort = () => {
    timings.abort = Date.now();
    timings.phases.total = timings.abort - timings.start;
  };
  request.prependOnceListener("abort", onAbort);
  const onSocket = (socket) => {
    timings.socket = Date.now();
    timings.phases.wait = timings.socket - timings.start;
    if (types.isProxy(socket)) {
      return;
    }
    const lookupListener = () => {
      timings.lookup = Date.now();
      timings.phases.dns = timings.lookup - timings.socket;
    };
    socket.prependOnceListener("lookup", lookupListener);
    (0, import_defer_to_connect.default)(socket, {
      connect: () => {
        timings.connect = Date.now();
        if (timings.lookup === void 0) {
          socket.removeListener("lookup", lookupListener);
          timings.lookup = timings.connect;
          timings.phases.dns = timings.lookup - timings.socket;
        }
        timings.phases.tcp = timings.connect - timings.lookup;
      },
      secureConnect: () => {
        timings.secureConnect = Date.now();
        timings.phases.tls = timings.secureConnect - timings.connect;
      }
    });
  };
  if (request.socket) {
    onSocket(request.socket);
  } else {
    request.prependOnceListener("socket", onSocket);
  }
  const onUpload = () => {
    timings.upload = Date.now();
    timings.phases.request = timings.upload - (timings.secureConnect ?? timings.connect);
  };
  if (request.writableFinished) {
    onUpload();
  } else {
    request.prependOnceListener("finish", onUpload);
  }
  request.prependOnceListener("response", (response) => {
    timings.response = Date.now();
    timings.phases.firstByte = timings.response - timings.upload;
    response.timings = timings;
    handleError(response);
    response.prependOnceListener("end", () => {
      request.off("abort", onAbort);
      response.off("aborted", onAbort);
      if (timings.phases.total) {
        return;
      }
      timings.end = Date.now();
      timings.phases.download = timings.end - timings.response;
      timings.phases.total = timings.end - timings.start;
    });
    response.prependOnceListener("aborted", onAbort);
  });
  return timings;
};
var source_default = timer;

// node_modules/.pnpm/cacheable-request@10.2.7/node_modules/cacheable-request/dist/index.js
import EventEmitter from "node:events";
import urlLib from "node:url";
import crypto from "node:crypto";
import stream, { PassThrough as PassThroughStream } from "node:stream";

// node_modules/.pnpm/normalize-url@8.0.0/node_modules/normalize-url/index.js
var DATA_URL_DEFAULT_MIME_TYPE = "text/plain";
var DATA_URL_DEFAULT_CHARSET = "us-ascii";
var testParameter = (name, filters) => filters.some((filter) => filter instanceof RegExp ? filter.test(name) : filter === name);
var supportedProtocols = /* @__PURE__ */ new Set([
  "https:",
  "http:",
  "file:"
]);
var hasCustomProtocol = (urlString) => {
  try {
    const { protocol } = new URL(urlString);
    return protocol.endsWith(":") && !supportedProtocols.has(protocol);
  } catch {
    return false;
  }
};
var normalizeDataURL = (urlString, { stripHash }) => {
  var _a;
  const match = /^data:(?<type>[^,]*?),(?<data>[^#]*?)(?:#(?<hash>.*))?$/.exec(urlString);
  if (!match) {
    throw new Error(`Invalid URL: ${urlString}`);
  }
  let { type, data, hash } = match.groups;
  const mediaType = type.split(";");
  hash = stripHash ? "" : hash;
  let isBase64 = false;
  if (mediaType[mediaType.length - 1] === "base64") {
    mediaType.pop();
    isBase64 = true;
  }
  const mimeType = ((_a = mediaType.shift()) == null ? void 0 : _a.toLowerCase()) ?? "";
  const attributes = mediaType.map((attribute) => {
    let [key, value = ""] = attribute.split("=").map((string) => string.trim());
    if (key === "charset") {
      value = value.toLowerCase();
      if (value === DATA_URL_DEFAULT_CHARSET) {
        return "";
      }
    }
    return `${key}${value ? `=${value}` : ""}`;
  }).filter(Boolean);
  const normalizedMediaType = [
    ...attributes
  ];
  if (isBase64) {
    normalizedMediaType.push("base64");
  }
  if (normalizedMediaType.length > 0 || mimeType && mimeType !== DATA_URL_DEFAULT_MIME_TYPE) {
    normalizedMediaType.unshift(mimeType);
  }
  return `data:${normalizedMediaType.join(";")},${isBase64 ? data.trim() : data}${hash ? `#${hash}` : ""}`;
};
function normalizeUrl(urlString, options) {
  options = {
    defaultProtocol: "http",
    normalizeProtocol: true,
    forceHttp: false,
    forceHttps: false,
    stripAuthentication: true,
    stripHash: false,
    stripTextFragment: true,
    stripWWW: true,
    removeQueryParameters: [/^utm_\w+/i],
    removeTrailingSlash: true,
    removeSingleSlash: true,
    removeDirectoryIndex: false,
    removeExplicitPort: false,
    sortQueryParameters: true,
    ...options
  };
  if (typeof options.defaultProtocol === "string" && !options.defaultProtocol.endsWith(":")) {
    options.defaultProtocol = `${options.defaultProtocol}:`;
  }
  urlString = urlString.trim();
  if (/^data:/i.test(urlString)) {
    return normalizeDataURL(urlString, options);
  }
  if (hasCustomProtocol(urlString)) {
    return urlString;
  }
  const hasRelativeProtocol = urlString.startsWith("//");
  const isRelativeUrl = !hasRelativeProtocol && /^\.*\//.test(urlString);
  if (!isRelativeUrl) {
    urlString = urlString.replace(/^(?!(?:\w+:)?\/\/)|^\/\//, options.defaultProtocol);
  }
  const urlObject = new URL(urlString);
  if (options.forceHttp && options.forceHttps) {
    throw new Error("The `forceHttp` and `forceHttps` options cannot be used together");
  }
  if (options.forceHttp && urlObject.protocol === "https:") {
    urlObject.protocol = "http:";
  }
  if (options.forceHttps && urlObject.protocol === "http:") {
    urlObject.protocol = "https:";
  }
  if (options.stripAuthentication) {
    urlObject.username = "";
    urlObject.password = "";
  }
  if (options.stripHash) {
    urlObject.hash = "";
  } else if (options.stripTextFragment) {
    urlObject.hash = urlObject.hash.replace(/#?:~:text.*?$/i, "");
  }
  if (urlObject.pathname) {
    const protocolRegex = /\b[a-z][a-z\d+\-.]{1,50}:\/\//g;
    let lastIndex = 0;
    let result = "";
    for (; ; ) {
      const match = protocolRegex.exec(urlObject.pathname);
      if (!match) {
        break;
      }
      const protocol = match[0];
      const protocolAtIndex = match.index;
      const intermediate = urlObject.pathname.slice(lastIndex, protocolAtIndex);
      result += intermediate.replace(/\/{2,}/g, "/");
      result += protocol;
      lastIndex = protocolAtIndex + protocol.length;
    }
    const remnant = urlObject.pathname.slice(lastIndex, urlObject.pathname.length);
    result += remnant.replace(/\/{2,}/g, "/");
    urlObject.pathname = result;
  }
  if (urlObject.pathname) {
    try {
      urlObject.pathname = decodeURI(urlObject.pathname);
    } catch {
    }
  }
  if (options.removeDirectoryIndex === true) {
    options.removeDirectoryIndex = [/^index\.[a-z]+$/];
  }
  if (Array.isArray(options.removeDirectoryIndex) && options.removeDirectoryIndex.length > 0) {
    let pathComponents = urlObject.pathname.split("/");
    const lastComponent = pathComponents[pathComponents.length - 1];
    if (testParameter(lastComponent, options.removeDirectoryIndex)) {
      pathComponents = pathComponents.slice(0, -1);
      urlObject.pathname = pathComponents.slice(1).join("/") + "/";
    }
  }
  if (urlObject.hostname) {
    urlObject.hostname = urlObject.hostname.replace(/\.$/, "");
    if (options.stripWWW && /^www\.(?!www\.)[a-z\-\d]{1,63}\.[a-z.\-\d]{2,63}$/.test(urlObject.hostname)) {
      urlObject.hostname = urlObject.hostname.replace(/^www\./, "");
    }
  }
  if (Array.isArray(options.removeQueryParameters)) {
    for (const key of [...urlObject.searchParams.keys()]) {
      if (testParameter(key, options.removeQueryParameters)) {
        urlObject.searchParams.delete(key);
      }
    }
  }
  if (!Array.isArray(options.keepQueryParameters) && options.removeQueryParameters === true) {
    urlObject.search = "";
  }
  if (Array.isArray(options.keepQueryParameters) && options.keepQueryParameters.length > 0) {
    for (const key of [...urlObject.searchParams.keys()]) {
      if (!testParameter(key, options.keepQueryParameters)) {
        urlObject.searchParams.delete(key);
      }
    }
  }
  if (options.sortQueryParameters) {
    urlObject.searchParams.sort();
    try {
      urlObject.search = decodeURIComponent(urlObject.search);
    } catch {
    }
  }
  if (options.removeTrailingSlash) {
    urlObject.pathname = urlObject.pathname.replace(/\/$/, "");
  }
  if (options.removeExplicitPort && urlObject.port) {
    urlObject.port = "";
  }
  const oldUrlString = urlString;
  urlString = urlObject.toString();
  if (!options.removeSingleSlash && urlObject.pathname === "/" && !oldUrlString.endsWith("/") && urlObject.hash === "") {
    urlString = urlString.replace(/\/$/, "");
  }
  if ((options.removeTrailingSlash || urlObject.pathname === "/") && urlObject.hash === "" && options.removeSingleSlash) {
    urlString = urlString.replace(/\/$/, "");
  }
  if (hasRelativeProtocol && !options.normalizeProtocol) {
    urlString = urlString.replace(/^http:\/\//, "//");
  }
  if (options.stripProtocol) {
    urlString = urlString.replace(/^(?:https?:)?\/\//, "");
  }
  return urlString;
}

// node_modules/.pnpm/cacheable-request@10.2.7/node_modules/cacheable-request/dist/index.js
var import_get_stream = __toESM(require_get_stream(), 1);
var import_http_cache_semantics = __toESM(require_http_cache_semantics(), 1);

// node_modules/.pnpm/responselike@3.0.0/node_modules/responselike/index.js
import { Readable as ReadableStream } from "node:stream";

// node_modules/.pnpm/lowercase-keys@3.0.0/node_modules/lowercase-keys/index.js
function lowercaseKeys(object) {
  return Object.fromEntries(Object.entries(object).map(([key, value]) => [key.toLowerCase(), value]));
}

// node_modules/.pnpm/responselike@3.0.0/node_modules/responselike/index.js
var Response = class extends ReadableStream {
  statusCode;
  headers;
  body;
  url;
  constructor({ statusCode, headers, body, url }) {
    if (typeof statusCode !== "number") {
      throw new TypeError("Argument `statusCode` should be a number");
    }
    if (typeof headers !== "object") {
      throw new TypeError("Argument `headers` should be an object");
    }
    if (!(body instanceof Uint8Array)) {
      throw new TypeError("Argument `body` should be a buffer");
    }
    if (typeof url !== "string") {
      throw new TypeError("Argument `url` should be a string");
    }
    super({
      read() {
        this.push(body);
        this.push(null);
      }
    });
    this.statusCode = statusCode;
    this.headers = lowercaseKeys(headers);
    this.body = body;
    this.url = url;
  }
};

// node_modules/.pnpm/cacheable-request@10.2.7/node_modules/cacheable-request/dist/index.js
var import_keyv = __toESM(require_src(), 1);

// node_modules/.pnpm/mimic-response@4.0.0/node_modules/mimic-response/index.js
var knownProperties = [
  "aborted",
  "complete",
  "headers",
  "httpVersion",
  "httpVersionMinor",
  "httpVersionMajor",
  "method",
  "rawHeaders",
  "rawTrailers",
  "setTimeout",
  "socket",
  "statusCode",
  "statusMessage",
  "trailers",
  "url"
];
function mimicResponse(fromStream, toStream) {
  if (toStream._readableState.autoDestroy) {
    throw new Error("The second stream must have the `autoDestroy` option set to `false`");
  }
  const fromProperties = /* @__PURE__ */ new Set([...Object.keys(fromStream), ...knownProperties]);
  const properties = {};
  for (const property of fromProperties) {
    if (property in toStream) {
      continue;
    }
    properties[property] = {
      get() {
        const value = fromStream[property];
        const isFunction2 = typeof value === "function";
        return isFunction2 ? value.bind(fromStream) : value;
      },
      set(value) {
        fromStream[property] = value;
      },
      enumerable: true,
      configurable: false
    };
  }
  Object.defineProperties(toStream, properties);
  fromStream.once("aborted", () => {
    toStream.destroy();
    toStream.emit("aborted");
  });
  fromStream.once("close", () => {
    if (fromStream.complete) {
      if (toStream.readable) {
        toStream.once("end", () => {
          toStream.emit("close");
        });
      } else {
        toStream.emit("close");
      }
    } else {
      toStream.emit("close");
    }
  });
  return toStream;
}

// node_modules/.pnpm/cacheable-request@10.2.7/node_modules/cacheable-request/dist/types.js
var RequestError2 = class extends Error {
  constructor(error) {
    super(error.message);
    Object.assign(this, error);
  }
};
var CacheError2 = class extends Error {
  constructor(error) {
    super(error.message);
    Object.assign(this, error);
  }
};

// node_modules/.pnpm/cacheable-request@10.2.7/node_modules/cacheable-request/dist/index.js
var CacheableRequest = class {
  constructor(cacheRequest, cacheAdapter) {
    this.hooks = /* @__PURE__ */ new Map();
    this.request = () => (options, cb) => {
      let url;
      if (typeof options === "string") {
        url = normalizeUrlObject(urlLib.parse(options));
        options = {};
      } else if (options instanceof urlLib.URL) {
        url = normalizeUrlObject(urlLib.parse(options.toString()));
        options = {};
      } else {
        const [pathname, ...searchParts] = (options.path ?? "").split("?");
        const search = searchParts.length > 0 ? `?${searchParts.join("?")}` : "";
        url = normalizeUrlObject({ ...options, pathname, search });
      }
      options = {
        headers: {},
        method: "GET",
        cache: true,
        strictTtl: false,
        automaticFailover: false,
        ...options,
        ...urlObjectToRequestOptions(url)
      };
      options.headers = Object.fromEntries(entries(options.headers).map(([key2, value]) => [key2.toLowerCase(), value]));
      const ee = new EventEmitter();
      const normalizedUrlString = normalizeUrl(urlLib.format(url), {
        stripWWW: false,
        removeTrailingSlash: false,
        stripAuthentication: false
      });
      let key = `${options.method}:${normalizedUrlString}`;
      if (options.body && options.method !== void 0 && ["POST", "PATCH", "PUT"].includes(options.method)) {
        if (options.body instanceof stream.Readable) {
          options.cache = false;
        } else {
          key += `:${crypto.createHash("md5").update(options.body).digest("hex")}`;
        }
      }
      let revalidate = false;
      let madeRequest = false;
      const makeRequest = (options_) => {
        madeRequest = true;
        let requestErrored = false;
        let requestErrorCallback = () => {
        };
        const requestErrorPromise = new Promise((resolve) => {
          requestErrorCallback = () => {
            if (!requestErrored) {
              requestErrored = true;
              resolve();
            }
          };
        });
        const handler = async (response) => {
          if (revalidate) {
            response.status = response.statusCode;
            const revalidatedPolicy = import_http_cache_semantics.default.fromObject(revalidate.cachePolicy).revalidatedPolicy(options_, response);
            if (!revalidatedPolicy.modified) {
              response.resume();
              await new Promise((resolve) => {
                response.once("end", resolve);
              });
              const headers = convertHeaders(revalidatedPolicy.policy.responseHeaders());
              response = new Response({ statusCode: revalidate.statusCode, headers, body: revalidate.body, url: revalidate.url });
              response.cachePolicy = revalidatedPolicy.policy;
              response.fromCache = true;
            }
          }
          if (!response.fromCache) {
            response.cachePolicy = new import_http_cache_semantics.default(options_, response, options_);
            response.fromCache = false;
          }
          let clonedResponse;
          if (options_.cache && response.cachePolicy.storable()) {
            clonedResponse = cloneResponse(response);
            (async () => {
              try {
                const bodyPromise = import_get_stream.default.buffer(response);
                await Promise.race([
                  requestErrorPromise,
                  new Promise((resolve) => response.once("end", resolve)),
                  new Promise((resolve) => response.once("close", resolve))
                  // eslint-disable-line no-promise-executor-return
                ]);
                const body = await bodyPromise;
                let value = {
                  url: response.url,
                  statusCode: response.fromCache ? revalidate.statusCode : response.statusCode,
                  body,
                  cachePolicy: response.cachePolicy.toObject()
                };
                let ttl2 = options_.strictTtl ? response.cachePolicy.timeToLive() : void 0;
                if (options_.maxTtl) {
                  ttl2 = ttl2 ? Math.min(ttl2, options_.maxTtl) : options_.maxTtl;
                }
                if (this.hooks.size > 0) {
                  for (const key_ of this.hooks.keys()) {
                    value = await this.runHook(key_, value, response);
                  }
                }
                await this.cache.set(key, value, ttl2);
              } catch (error) {
                ee.emit("error", new CacheError2(error));
              }
            })();
          } else if (options_.cache && revalidate) {
            (async () => {
              try {
                await this.cache.delete(key);
              } catch (error) {
                ee.emit("error", new CacheError2(error));
              }
            })();
          }
          ee.emit("response", clonedResponse ?? response);
          if (typeof cb === "function") {
            cb(clonedResponse ?? response);
          }
        };
        try {
          const request_ = this.cacheRequest(options_, handler);
          request_.once("error", requestErrorCallback);
          request_.once("abort", requestErrorCallback);
          request_.once("destroy", requestErrorCallback);
          ee.emit("request", request_);
        } catch (error) {
          ee.emit("error", new RequestError2(error));
        }
      };
      (async () => {
        const get = async (options_) => {
          await Promise.resolve();
          const cacheEntry = options_.cache ? await this.cache.get(key) : void 0;
          if (typeof cacheEntry === "undefined" && !options_.forceRefresh) {
            makeRequest(options_);
            return;
          }
          const policy = import_http_cache_semantics.default.fromObject(cacheEntry.cachePolicy);
          if (policy.satisfiesWithoutRevalidation(options_) && !options_.forceRefresh) {
            const headers = convertHeaders(policy.responseHeaders());
            const response = new Response({ statusCode: cacheEntry.statusCode, headers, body: cacheEntry.body, url: cacheEntry.url });
            response.cachePolicy = policy;
            response.fromCache = true;
            ee.emit("response", response);
            if (typeof cb === "function") {
              cb(response);
            }
          } else if (policy.satisfiesWithoutRevalidation(options_) && Date.now() >= policy.timeToLive() && options_.forceRefresh) {
            await this.cache.delete(key);
            options_.headers = policy.revalidationHeaders(options_);
            makeRequest(options_);
          } else {
            revalidate = cacheEntry;
            options_.headers = policy.revalidationHeaders(options_);
            makeRequest(options_);
          }
        };
        const errorHandler = (error) => ee.emit("error", new CacheError2(error));
        if (this.cache instanceof import_keyv.default) {
          const cachek = this.cache;
          cachek.once("error", errorHandler);
          ee.on("error", () => cachek.removeListener("error", errorHandler));
          ee.on("response", () => cachek.removeListener("error", errorHandler));
        }
        try {
          await get(options);
        } catch (error) {
          if (options.automaticFailover && !madeRequest) {
            makeRequest(options);
          }
          ee.emit("error", new CacheError2(error));
        }
      })();
      return ee;
    };
    this.addHook = (name, fn) => {
      if (!this.hooks.has(name)) {
        this.hooks.set(name, fn);
      }
    };
    this.removeHook = (name) => this.hooks.delete(name);
    this.getHook = (name) => this.hooks.get(name);
    this.runHook = async (name, ...args) => {
      var _a;
      return (_a = this.hooks.get(name)) == null ? void 0 : _a(...args);
    };
    if (cacheAdapter instanceof import_keyv.default) {
      this.cache = cacheAdapter;
    } else if (typeof cacheAdapter === "string") {
      this.cache = new import_keyv.default({
        uri: cacheAdapter,
        namespace: "cacheable-request"
      });
    } else {
      this.cache = new import_keyv.default({
        store: cacheAdapter,
        namespace: "cacheable-request"
      });
    }
    this.request = this.request.bind(this);
    this.cacheRequest = cacheRequest;
  }
};
var entries = Object.entries;
var cloneResponse = (response) => {
  const clone = new PassThroughStream({ autoDestroy: false });
  mimicResponse(response, clone);
  return response.pipe(clone);
};
var urlObjectToRequestOptions = (url) => {
  const options = { ...url };
  options.path = `${url.pathname || "/"}${url.search || ""}`;
  delete options.pathname;
  delete options.search;
  return options;
};
var normalizeUrlObject = (url) => (
  // If url was parsed by url.parse or new URL:
  // - hostname will be set
  // - host will be hostname[:port]
  // - port will be set if it was explicit in the parsed string
  // Otherwise, url was from request options:
  // - hostname or host may be set
  // - host shall not have port encoded
  {
    protocol: url.protocol,
    auth: url.auth,
    hostname: url.hostname || url.host || "localhost",
    port: url.port,
    pathname: url.pathname,
    search: url.search
  }
);
var convertHeaders = (headers) => {
  const result = [];
  for (const name of Object.keys(headers)) {
    result[name.toLowerCase()] = headers[name];
  }
  return result;
};
var dist_default2 = CacheableRequest;

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/index.js
var import_decompress_response = __toESM(require_decompress_response(), 1);
var import_get_stream2 = __toESM(require_get_stream(), 1);

// node_modules/.pnpm/form-data-encoder@2.1.4/node_modules/form-data-encoder/lib/util/isFunction.js
var isFunction = (value) => typeof value === "function";

// node_modules/.pnpm/form-data-encoder@2.1.4/node_modules/form-data-encoder/lib/util/getStreamIterator.js
var isAsyncIterable = (value) => isFunction(value[Symbol.asyncIterator]);
async function* readStream(readable) {
  const reader = readable.getReader();
  while (true) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }
    yield value;
  }
}
var getStreamIterator = (source) => {
  if (isAsyncIterable(source)) {
    return source;
  }
  if (isFunction(source.getReader)) {
    return readStream(source);
  }
  throw new TypeError("Unsupported data source: Expected either ReadableStream or async iterable.");
};

// node_modules/.pnpm/form-data-encoder@2.1.4/node_modules/form-data-encoder/lib/util/createBoundary.js
var alphabet = "abcdefghijklmnopqrstuvwxyz0123456789";
function createBoundary() {
  let size = 16;
  let res = "";
  while (size--) {
    res += alphabet[Math.random() * alphabet.length << 0];
  }
  return res;
}

// node_modules/.pnpm/form-data-encoder@2.1.4/node_modules/form-data-encoder/lib/util/normalizeValue.js
var normalizeValue = (value) => String(value).replace(/\r|\n/g, (match, i, str) => {
  if (match === "\r" && str[i + 1] !== "\n" || match === "\n" && str[i - 1] !== "\r") {
    return "\r\n";
  }
  return match;
});

// node_modules/.pnpm/form-data-encoder@2.1.4/node_modules/form-data-encoder/lib/util/isPlainObject.js
var getType = (value) => Object.prototype.toString.call(value).slice(8, -1).toLowerCase();
function isPlainObject(value) {
  if (getType(value) !== "object") {
    return false;
  }
  const pp = Object.getPrototypeOf(value);
  if (pp === null || pp === void 0) {
    return true;
  }
  const Ctor = pp.constructor && pp.constructor.toString();
  return Ctor === Object.toString();
}

// node_modules/.pnpm/form-data-encoder@2.1.4/node_modules/form-data-encoder/lib/util/proxyHeaders.js
function getProperty(target, prop) {
  if (typeof prop === "string") {
    for (const [name, value] of Object.entries(target)) {
      if (prop.toLowerCase() === name.toLowerCase()) {
        return value;
      }
    }
  }
  return void 0;
}
var proxyHeaders = (object) => new Proxy(object, {
  get: (target, prop) => getProperty(target, prop),
  has: (target, prop) => getProperty(target, prop) !== void 0
});

// node_modules/.pnpm/form-data-encoder@2.1.4/node_modules/form-data-encoder/lib/util/isFormData.js
var isFormData = (value) => Boolean(value && isFunction(value.constructor) && value[Symbol.toStringTag] === "FormData" && isFunction(value.append) && isFunction(value.getAll) && isFunction(value.entries) && isFunction(value[Symbol.iterator]));

// node_modules/.pnpm/form-data-encoder@2.1.4/node_modules/form-data-encoder/lib/util/escapeName.js
var escapeName = (name) => String(name).replace(/\r/g, "%0D").replace(/\n/g, "%0A").replace(/"/g, "%22");

// node_modules/.pnpm/form-data-encoder@2.1.4/node_modules/form-data-encoder/lib/util/isFile.js
var isFile = (value) => Boolean(value && typeof value === "object" && isFunction(value.constructor) && value[Symbol.toStringTag] === "File" && isFunction(value.stream) && value.name != null);

// node_modules/.pnpm/form-data-encoder@2.1.4/node_modules/form-data-encoder/lib/FormDataEncoder.js
var __classPrivateFieldSet = function(receiver, state, value, kind, f) {
  if (kind === "m")
    throw new TypeError("Private method is not writable");
  if (kind === "a" && !f)
    throw new TypeError("Private accessor was defined without a setter");
  if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver))
    throw new TypeError("Cannot write private member to an object whose class did not declare it");
  return kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value), value;
};
var __classPrivateFieldGet = function(receiver, state, kind, f) {
  if (kind === "a" && !f)
    throw new TypeError("Private accessor was defined without a getter");
  if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver))
    throw new TypeError("Cannot read private member from an object whose class did not declare it");
  return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var _FormDataEncoder_instances;
var _FormDataEncoder_CRLF;
var _FormDataEncoder_CRLF_BYTES;
var _FormDataEncoder_CRLF_BYTES_LENGTH;
var _FormDataEncoder_DASHES;
var _FormDataEncoder_encoder;
var _FormDataEncoder_footer;
var _FormDataEncoder_form;
var _FormDataEncoder_options;
var _FormDataEncoder_getFieldHeader;
var _FormDataEncoder_getContentLength;
var defaultOptions = {
  enableAdditionalHeaders: false
};
var readonlyProp = { writable: false, configurable: false };
var FormDataEncoder = class {
  constructor(form, boundaryOrOptions, options) {
    _FormDataEncoder_instances.add(this);
    _FormDataEncoder_CRLF.set(this, "\r\n");
    _FormDataEncoder_CRLF_BYTES.set(this, void 0);
    _FormDataEncoder_CRLF_BYTES_LENGTH.set(this, void 0);
    _FormDataEncoder_DASHES.set(this, "-".repeat(2));
    _FormDataEncoder_encoder.set(this, new TextEncoder());
    _FormDataEncoder_footer.set(this, void 0);
    _FormDataEncoder_form.set(this, void 0);
    _FormDataEncoder_options.set(this, void 0);
    if (!isFormData(form)) {
      throw new TypeError("Expected first argument to be a FormData instance.");
    }
    let boundary;
    if (isPlainObject(boundaryOrOptions)) {
      options = boundaryOrOptions;
    } else {
      boundary = boundaryOrOptions;
    }
    if (!boundary) {
      boundary = createBoundary();
    }
    if (typeof boundary !== "string") {
      throw new TypeError("Expected boundary argument to be a string.");
    }
    if (options && !isPlainObject(options)) {
      throw new TypeError("Expected options argument to be an object.");
    }
    __classPrivateFieldSet(this, _FormDataEncoder_form, Array.from(form.entries()), "f");
    __classPrivateFieldSet(this, _FormDataEncoder_options, { ...defaultOptions, ...options }, "f");
    __classPrivateFieldSet(this, _FormDataEncoder_CRLF_BYTES, __classPrivateFieldGet(this, _FormDataEncoder_encoder, "f").encode(__classPrivateFieldGet(this, _FormDataEncoder_CRLF, "f")), "f");
    __classPrivateFieldSet(this, _FormDataEncoder_CRLF_BYTES_LENGTH, __classPrivateFieldGet(this, _FormDataEncoder_CRLF_BYTES, "f").byteLength, "f");
    this.boundary = `form-data-boundary-${boundary}`;
    this.contentType = `multipart/form-data; boundary=${this.boundary}`;
    __classPrivateFieldSet(this, _FormDataEncoder_footer, __classPrivateFieldGet(this, _FormDataEncoder_encoder, "f").encode(`${__classPrivateFieldGet(this, _FormDataEncoder_DASHES, "f")}${this.boundary}${__classPrivateFieldGet(this, _FormDataEncoder_DASHES, "f")}${__classPrivateFieldGet(this, _FormDataEncoder_CRLF, "f").repeat(2)}`), "f");
    const headers = {
      "Content-Type": this.contentType
    };
    const contentLength = __classPrivateFieldGet(this, _FormDataEncoder_instances, "m", _FormDataEncoder_getContentLength).call(this);
    if (contentLength) {
      this.contentLength = contentLength;
      headers["Content-Length"] = contentLength;
    }
    this.headers = proxyHeaders(Object.freeze(headers));
    Object.defineProperties(this, {
      boundary: readonlyProp,
      contentType: readonlyProp,
      contentLength: readonlyProp,
      headers: readonlyProp
    });
  }
  getContentLength() {
    return this.contentLength == null ? void 0 : Number(this.contentLength);
  }
  *values() {
    for (const [name, raw] of __classPrivateFieldGet(this, _FormDataEncoder_form, "f")) {
      const value = isFile(raw) ? raw : __classPrivateFieldGet(this, _FormDataEncoder_encoder, "f").encode(normalizeValue(raw));
      yield __classPrivateFieldGet(this, _FormDataEncoder_instances, "m", _FormDataEncoder_getFieldHeader).call(this, name, value);
      yield value;
      yield __classPrivateFieldGet(this, _FormDataEncoder_CRLF_BYTES, "f");
    }
    yield __classPrivateFieldGet(this, _FormDataEncoder_footer, "f");
  }
  async *encode() {
    for (const part of this.values()) {
      if (isFile(part)) {
        yield* getStreamIterator(part.stream());
      } else {
        yield part;
      }
    }
  }
  [(_FormDataEncoder_CRLF = /* @__PURE__ */ new WeakMap(), _FormDataEncoder_CRLF_BYTES = /* @__PURE__ */ new WeakMap(), _FormDataEncoder_CRLF_BYTES_LENGTH = /* @__PURE__ */ new WeakMap(), _FormDataEncoder_DASHES = /* @__PURE__ */ new WeakMap(), _FormDataEncoder_encoder = /* @__PURE__ */ new WeakMap(), _FormDataEncoder_footer = /* @__PURE__ */ new WeakMap(), _FormDataEncoder_form = /* @__PURE__ */ new WeakMap(), _FormDataEncoder_options = /* @__PURE__ */ new WeakMap(), _FormDataEncoder_instances = /* @__PURE__ */ new WeakSet(), _FormDataEncoder_getFieldHeader = function _FormDataEncoder_getFieldHeader2(name, value) {
    let header = "";
    header += `${__classPrivateFieldGet(this, _FormDataEncoder_DASHES, "f")}${this.boundary}${__classPrivateFieldGet(this, _FormDataEncoder_CRLF, "f")}`;
    header += `Content-Disposition: form-data; name="${escapeName(name)}"`;
    if (isFile(value)) {
      header += `; filename="${escapeName(value.name)}"${__classPrivateFieldGet(this, _FormDataEncoder_CRLF, "f")}`;
      header += `Content-Type: ${value.type || "application/octet-stream"}`;
    }
    const size = isFile(value) ? value.size : value.byteLength;
    if (__classPrivateFieldGet(this, _FormDataEncoder_options, "f").enableAdditionalHeaders === true && size != null && !isNaN(size)) {
      header += `${__classPrivateFieldGet(this, _FormDataEncoder_CRLF, "f")}Content-Length: ${isFile(value) ? value.size : value.byteLength}`;
    }
    return __classPrivateFieldGet(this, _FormDataEncoder_encoder, "f").encode(`${header}${__classPrivateFieldGet(this, _FormDataEncoder_CRLF, "f").repeat(2)}`);
  }, _FormDataEncoder_getContentLength = function _FormDataEncoder_getContentLength2() {
    let length = 0;
    for (const [name, raw] of __classPrivateFieldGet(this, _FormDataEncoder_form, "f")) {
      const value = isFile(raw) ? raw : __classPrivateFieldGet(this, _FormDataEncoder_encoder, "f").encode(normalizeValue(raw));
      const size = isFile(value) ? value.size : value.byteLength;
      if (size == null || isNaN(size)) {
        return void 0;
      }
      length += __classPrivateFieldGet(this, _FormDataEncoder_instances, "m", _FormDataEncoder_getFieldHeader).call(this, name, value).byteLength;
      length += size;
      length += __classPrivateFieldGet(this, _FormDataEncoder_CRLF_BYTES_LENGTH, "f");
    }
    return String(length + __classPrivateFieldGet(this, _FormDataEncoder_footer, "f").byteLength);
  }, Symbol.iterator)]() {
    return this.values();
  }
  [Symbol.asyncIterator]() {
    return this.encode();
  }
};

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/utils/get-body-size.js
import { Buffer as Buffer2 } from "node:buffer";
import { promisify } from "node:util";

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/utils/is-form-data.js
function isFormData2(body) {
  return dist_default.nodeStream(body) && dist_default.function_(body.getBoundary);
}

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/utils/get-body-size.js
async function getBodySize(body, headers) {
  if (headers && "content-length" in headers) {
    return Number(headers["content-length"]);
  }
  if (!body) {
    return 0;
  }
  if (dist_default.string(body)) {
    return Buffer2.byteLength(body);
  }
  if (dist_default.buffer(body)) {
    return body.length;
  }
  if (isFormData2(body)) {
    return promisify(body.getLength.bind(body))();
  }
  return void 0;
}

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/utils/proxy-events.js
function proxyEvents(from, to, events) {
  const eventFunctions = {};
  for (const event of events) {
    const eventFunction = (...args) => {
      to.emit(event, ...args);
    };
    eventFunctions[event] = eventFunction;
    from.on(event, eventFunction);
  }
  return () => {
    for (const [event, eventFunction] of Object.entries(eventFunctions)) {
      from.off(event, eventFunction);
    }
  };
}

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/timed-out.js
import net from "node:net";

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/utils/unhandle.js
function unhandle() {
  const handlers = [];
  return {
    once(origin, event, fn) {
      origin.once(event, fn);
      handlers.push({ origin, event, fn });
    },
    unhandleAll() {
      for (const handler of handlers) {
        const { origin, event, fn } = handler;
        origin.removeListener(event, fn);
      }
      handlers.length = 0;
    }
  };
}

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/timed-out.js
var reentry = Symbol("reentry");
var noop = () => {
};
var TimeoutError2 = class extends Error {
  constructor(threshold, event) {
    super(`Timeout awaiting '${event}' for ${threshold}ms`);
    Object.defineProperty(this, "event", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: event
    });
    Object.defineProperty(this, "code", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    this.name = "TimeoutError";
    this.code = "ETIMEDOUT";
  }
};
function timedOut(request, delays, options) {
  if (reentry in request) {
    return noop;
  }
  request[reentry] = true;
  const cancelers = [];
  const { once, unhandleAll } = unhandle();
  const addTimeout = (delay2, callback, event) => {
    var _a;
    const timeout = setTimeout(callback, delay2, delay2, event);
    (_a = timeout.unref) == null ? void 0 : _a.call(timeout);
    const cancel = () => {
      clearTimeout(timeout);
    };
    cancelers.push(cancel);
    return cancel;
  };
  const { host, hostname } = options;
  const timeoutHandler = (delay2, event) => {
    request.destroy(new TimeoutError2(delay2, event));
  };
  const cancelTimeouts = () => {
    for (const cancel of cancelers) {
      cancel();
    }
    unhandleAll();
  };
  request.once("error", (error) => {
    cancelTimeouts();
    if (request.listenerCount("error") === 0) {
      throw error;
    }
  });
  if (typeof delays.request !== "undefined") {
    const cancelTimeout = addTimeout(delays.request, timeoutHandler, "request");
    once(request, "response", (response) => {
      once(response, "end", cancelTimeout);
    });
  }
  if (typeof delays.socket !== "undefined") {
    const { socket } = delays;
    const socketTimeoutHandler = () => {
      timeoutHandler(socket, "socket");
    };
    request.setTimeout(socket, socketTimeoutHandler);
    cancelers.push(() => {
      request.removeListener("timeout", socketTimeoutHandler);
    });
  }
  const hasLookup = typeof delays.lookup !== "undefined";
  const hasConnect = typeof delays.connect !== "undefined";
  const hasSecureConnect = typeof delays.secureConnect !== "undefined";
  const hasSend = typeof delays.send !== "undefined";
  if (hasLookup || hasConnect || hasSecureConnect || hasSend) {
    once(request, "socket", (socket) => {
      const { socketPath } = request;
      if (socket.connecting) {
        const hasPath = Boolean(socketPath ?? net.isIP(hostname ?? host ?? "") !== 0);
        if (hasLookup && !hasPath && typeof socket.address().address === "undefined") {
          const cancelTimeout = addTimeout(delays.lookup, timeoutHandler, "lookup");
          once(socket, "lookup", cancelTimeout);
        }
        if (hasConnect) {
          const timeConnect = () => addTimeout(delays.connect, timeoutHandler, "connect");
          if (hasPath) {
            once(socket, "connect", timeConnect());
          } else {
            once(socket, "lookup", (error) => {
              if (error === null) {
                once(socket, "connect", timeConnect());
              }
            });
          }
        }
        if (hasSecureConnect && options.protocol === "https:") {
          once(socket, "connect", () => {
            const cancelTimeout = addTimeout(delays.secureConnect, timeoutHandler, "secureConnect");
            once(socket, "secureConnect", cancelTimeout);
          });
        }
      }
      if (hasSend) {
        const timeRequest = () => addTimeout(delays.send, timeoutHandler, "send");
        if (socket.connecting) {
          once(socket, "connect", () => {
            once(request, "upload-complete", timeRequest());
          });
        } else {
          once(request, "upload-complete", timeRequest());
        }
      }
    });
  }
  if (typeof delays.response !== "undefined") {
    once(request, "upload-complete", () => {
      const cancelTimeout = addTimeout(delays.response, timeoutHandler, "response");
      once(request, "response", cancelTimeout);
    });
  }
  if (typeof delays.read !== "undefined") {
    once(request, "response", (response) => {
      const cancelTimeout = addTimeout(delays.read, timeoutHandler, "read");
      once(response, "end", cancelTimeout);
    });
  }
  return cancelTimeouts;
}

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/utils/url-to-options.js
function urlToOptions(url) {
  url = url;
  const options = {
    protocol: url.protocol,
    hostname: dist_default.string(url.hostname) && url.hostname.startsWith("[") ? url.hostname.slice(1, -1) : url.hostname,
    host: url.host,
    hash: url.hash,
    search: url.search,
    pathname: url.pathname,
    href: url.href,
    path: `${url.pathname || ""}${url.search || ""}`
  };
  if (dist_default.string(url.port) && url.port.length > 0) {
    options.port = Number(url.port);
  }
  if (url.username || url.password) {
    options.auth = `${url.username || ""}:${url.password || ""}`;
  }
  return options;
}

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/utils/weakable-map.js
var WeakableMap = class {
  constructor() {
    Object.defineProperty(this, "weakMap", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "map", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    this.weakMap = /* @__PURE__ */ new WeakMap();
    this.map = /* @__PURE__ */ new Map();
  }
  set(key, value) {
    if (typeof key === "object") {
      this.weakMap.set(key, value);
    } else {
      this.map.set(key, value);
    }
  }
  get(key) {
    if (typeof key === "object") {
      return this.weakMap.get(key);
    }
    return this.map.get(key);
  }
  has(key) {
    if (typeof key === "object") {
      return this.weakMap.has(key);
    }
    return this.map.has(key);
  }
};

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/calculate-retry-delay.js
var calculateRetryDelay = ({ attemptCount, retryOptions, error, retryAfter, computedValue }) => {
  if (error.name === "RetryError") {
    return 1;
  }
  if (attemptCount > retryOptions.limit) {
    return 0;
  }
  const hasMethod = retryOptions.methods.includes(error.options.method);
  const hasErrorCode = retryOptions.errorCodes.includes(error.code);
  const hasStatusCode = error.response && retryOptions.statusCodes.includes(error.response.statusCode);
  if (!hasMethod || !hasErrorCode && !hasStatusCode) {
    return 0;
  }
  if (error.response) {
    if (retryAfter) {
      if (retryAfter > computedValue) {
        return 0;
      }
      return retryAfter;
    }
    if (error.response.statusCode === 413) {
      return 0;
    }
  }
  const noise = Math.random() * retryOptions.noise;
  return Math.min(2 ** (attemptCount - 1) * 1e3, retryOptions.backoffLimit) + noise;
};
var calculate_retry_delay_default = calculateRetryDelay;

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/options.js
import process2 from "node:process";
import { promisify as promisify3, inspect } from "node:util";
import { URL as URL2, URLSearchParams } from "node:url";
import { checkServerIdentity } from "node:tls";
import http from "node:http";
import https from "node:https";

// node_modules/.pnpm/cacheable-lookup@7.0.0/node_modules/cacheable-lookup/source/index.js
import {
  V4MAPPED,
  ADDRCONFIG,
  ALL,
  promises as dnsPromises,
  lookup as dnsLookup
} from "node:dns";
import { promisify as promisify2 } from "node:util";
import os from "node:os";
var { Resolver: AsyncResolver } = dnsPromises;
var kCacheableLookupCreateConnection = Symbol("cacheableLookupCreateConnection");
var kCacheableLookupInstance = Symbol("cacheableLookupInstance");
var kExpires = Symbol("expires");
var supportsALL = typeof ALL === "number";
var verifyAgent = (agent2) => {
  if (!(agent2 && typeof agent2.createConnection === "function")) {
    throw new Error("Expected an Agent instance as the first argument");
  }
};
var map4to6 = (entries2) => {
  for (const entry of entries2) {
    if (entry.family === 6) {
      continue;
    }
    entry.address = `::ffff:${entry.address}`;
    entry.family = 6;
  }
};
var getIfaceInfo = () => {
  let has4 = false;
  let has6 = false;
  for (const device of Object.values(os.networkInterfaces())) {
    for (const iface of device) {
      if (iface.internal) {
        continue;
      }
      if (iface.family === "IPv6") {
        has6 = true;
      } else {
        has4 = true;
      }
      if (has4 && has6) {
        return { has4, has6 };
      }
    }
  }
  return { has4, has6 };
};
var isIterable = (map) => {
  return Symbol.iterator in map;
};
var ignoreNoResultErrors = (dnsPromise) => {
  return dnsPromise.catch((error) => {
    if (error.code === "ENODATA" || error.code === "ENOTFOUND" || error.code === "ENOENT") {
      return [];
    }
    throw error;
  });
};
var ttl = { ttl: true };
var all = { all: true };
var all4 = { all: true, family: 4 };
var all6 = { all: true, family: 6 };
var CacheableLookup = class {
  constructor({
    cache = /* @__PURE__ */ new Map(),
    maxTtl = Infinity,
    fallbackDuration = 3600,
    errorTtl = 0.15,
    resolver = new AsyncResolver(),
    lookup = dnsLookup
  } = {}) {
    this.maxTtl = maxTtl;
    this.errorTtl = errorTtl;
    this._cache = cache;
    this._resolver = resolver;
    this._dnsLookup = lookup && promisify2(lookup);
    this.stats = {
      cache: 0,
      query: 0
    };
    if (this._resolver instanceof AsyncResolver) {
      this._resolve4 = this._resolver.resolve4.bind(this._resolver);
      this._resolve6 = this._resolver.resolve6.bind(this._resolver);
    } else {
      this._resolve4 = promisify2(this._resolver.resolve4.bind(this._resolver));
      this._resolve6 = promisify2(this._resolver.resolve6.bind(this._resolver));
    }
    this._iface = getIfaceInfo();
    this._pending = {};
    this._nextRemovalTime = false;
    this._hostnamesToFallback = /* @__PURE__ */ new Set();
    this.fallbackDuration = fallbackDuration;
    if (fallbackDuration > 0) {
      const interval = setInterval(() => {
        this._hostnamesToFallback.clear();
      }, fallbackDuration * 1e3);
      if (interval.unref) {
        interval.unref();
      }
      this._fallbackInterval = interval;
    }
    this.lookup = this.lookup.bind(this);
    this.lookupAsync = this.lookupAsync.bind(this);
  }
  set servers(servers) {
    this.clear();
    this._resolver.setServers(servers);
  }
  get servers() {
    return this._resolver.getServers();
  }
  lookup(hostname, options, callback) {
    if (typeof options === "function") {
      callback = options;
      options = {};
    } else if (typeof options === "number") {
      options = {
        family: options
      };
    }
    if (!callback) {
      throw new Error("Callback must be a function.");
    }
    this.lookupAsync(hostname, options).then((result) => {
      if (options.all) {
        callback(null, result);
      } else {
        callback(null, result.address, result.family, result.expires, result.ttl, result.source);
      }
    }, callback);
  }
  async lookupAsync(hostname, options = {}) {
    if (typeof options === "number") {
      options = {
        family: options
      };
    }
    let cached = await this.query(hostname);
    if (options.family === 6) {
      const filtered = cached.filter((entry) => entry.family === 6);
      if (options.hints & V4MAPPED) {
        if (supportsALL && options.hints & ALL || filtered.length === 0) {
          map4to6(cached);
        } else {
          cached = filtered;
        }
      } else {
        cached = filtered;
      }
    } else if (options.family === 4) {
      cached = cached.filter((entry) => entry.family === 4);
    }
    if (options.hints & ADDRCONFIG) {
      const { _iface } = this;
      cached = cached.filter((entry) => entry.family === 6 ? _iface.has6 : _iface.has4);
    }
    if (cached.length === 0) {
      const error = new Error(`cacheableLookup ENOTFOUND ${hostname}`);
      error.code = "ENOTFOUND";
      error.hostname = hostname;
      throw error;
    }
    if (options.all) {
      return cached;
    }
    return cached[0];
  }
  async query(hostname) {
    let source = "cache";
    let cached = await this._cache.get(hostname);
    if (cached) {
      this.stats.cache++;
    }
    if (!cached) {
      const pending = this._pending[hostname];
      if (pending) {
        this.stats.cache++;
        cached = await pending;
      } else {
        source = "query";
        const newPromise = this.queryAndCache(hostname);
        this._pending[hostname] = newPromise;
        this.stats.query++;
        try {
          cached = await newPromise;
        } finally {
          delete this._pending[hostname];
        }
      }
    }
    cached = cached.map((entry) => {
      return { ...entry, source };
    });
    return cached;
  }
  async _resolve(hostname) {
    const [A, AAAA] = await Promise.all([
      ignoreNoResultErrors(this._resolve4(hostname, ttl)),
      ignoreNoResultErrors(this._resolve6(hostname, ttl))
    ]);
    let aTtl = 0;
    let aaaaTtl = 0;
    let cacheTtl = 0;
    const now = Date.now();
    for (const entry of A) {
      entry.family = 4;
      entry.expires = now + entry.ttl * 1e3;
      aTtl = Math.max(aTtl, entry.ttl);
    }
    for (const entry of AAAA) {
      entry.family = 6;
      entry.expires = now + entry.ttl * 1e3;
      aaaaTtl = Math.max(aaaaTtl, entry.ttl);
    }
    if (A.length > 0) {
      if (AAAA.length > 0) {
        cacheTtl = Math.min(aTtl, aaaaTtl);
      } else {
        cacheTtl = aTtl;
      }
    } else {
      cacheTtl = aaaaTtl;
    }
    return {
      entries: [
        ...A,
        ...AAAA
      ],
      cacheTtl
    };
  }
  async _lookup(hostname) {
    try {
      const [A, AAAA] = await Promise.all([
        // Passing {all: true} doesn't return all IPv4 and IPv6 entries.
        // See https://github.com/szmarczak/cacheable-lookup/issues/42
        ignoreNoResultErrors(this._dnsLookup(hostname, all4)),
        ignoreNoResultErrors(this._dnsLookup(hostname, all6))
      ]);
      return {
        entries: [
          ...A,
          ...AAAA
        ],
        cacheTtl: 0
      };
    } catch {
      return {
        entries: [],
        cacheTtl: 0
      };
    }
  }
  async _set(hostname, data, cacheTtl) {
    if (this.maxTtl > 0 && cacheTtl > 0) {
      cacheTtl = Math.min(cacheTtl, this.maxTtl) * 1e3;
      data[kExpires] = Date.now() + cacheTtl;
      try {
        await this._cache.set(hostname, data, cacheTtl);
      } catch (error) {
        this.lookupAsync = async () => {
          const cacheError = new Error("Cache Error. Please recreate the CacheableLookup instance.");
          cacheError.cause = error;
          throw cacheError;
        };
      }
      if (isIterable(this._cache)) {
        this._tick(cacheTtl);
      }
    }
  }
  async queryAndCache(hostname) {
    if (this._hostnamesToFallback.has(hostname)) {
      return this._dnsLookup(hostname, all);
    }
    let query = await this._resolve(hostname);
    if (query.entries.length === 0 && this._dnsLookup) {
      query = await this._lookup(hostname);
      if (query.entries.length !== 0 && this.fallbackDuration > 0) {
        this._hostnamesToFallback.add(hostname);
      }
    }
    const cacheTtl = query.entries.length === 0 ? this.errorTtl : query.cacheTtl;
    await this._set(hostname, query.entries, cacheTtl);
    return query.entries;
  }
  _tick(ms) {
    const nextRemovalTime = this._nextRemovalTime;
    if (!nextRemovalTime || ms < nextRemovalTime) {
      clearTimeout(this._removalTimeout);
      this._nextRemovalTime = ms;
      this._removalTimeout = setTimeout(() => {
        this._nextRemovalTime = false;
        let nextExpiry = Infinity;
        const now = Date.now();
        for (const [hostname, entries2] of this._cache) {
          const expires = entries2[kExpires];
          if (now >= expires) {
            this._cache.delete(hostname);
          } else if (expires < nextExpiry) {
            nextExpiry = expires;
          }
        }
        if (nextExpiry !== Infinity) {
          this._tick(nextExpiry - now);
        }
      }, ms);
      if (this._removalTimeout.unref) {
        this._removalTimeout.unref();
      }
    }
  }
  install(agent2) {
    verifyAgent(agent2);
    if (kCacheableLookupCreateConnection in agent2) {
      throw new Error("CacheableLookup has been already installed");
    }
    agent2[kCacheableLookupCreateConnection] = agent2.createConnection;
    agent2[kCacheableLookupInstance] = this;
    agent2.createConnection = (options, callback) => {
      if (!("lookup" in options)) {
        options.lookup = this.lookup;
      }
      return agent2[kCacheableLookupCreateConnection](options, callback);
    };
  }
  uninstall(agent2) {
    verifyAgent(agent2);
    if (agent2[kCacheableLookupCreateConnection]) {
      if (agent2[kCacheableLookupInstance] !== this) {
        throw new Error("The agent is not owned by this CacheableLookup instance");
      }
      agent2.createConnection = agent2[kCacheableLookupCreateConnection];
      delete agent2[kCacheableLookupCreateConnection];
      delete agent2[kCacheableLookupInstance];
    }
  }
  updateInterfaceInfo() {
    const { _iface } = this;
    this._iface = getIfaceInfo();
    if (_iface.has4 && !this._iface.has4 || _iface.has6 && !this._iface.has6) {
      this._cache.clear();
    }
  }
  clear(hostname) {
    if (hostname) {
      this._cache.delete(hostname);
      return;
    }
    this._cache.clear();
  }
};

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/options.js
var import_http2_wrapper = __toESM(require_source2(), 1);

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/parse-link-header.js
function parseLinkHeader(link) {
  const parsed = [];
  const items = link.split(",");
  for (const item of items) {
    const [rawUriReference, ...rawLinkParameters] = item.split(";");
    const trimmedUriReference = rawUriReference.trim();
    if (trimmedUriReference[0] !== "<" || trimmedUriReference[trimmedUriReference.length - 1] !== ">") {
      throw new Error(`Invalid format of the Link header reference: ${trimmedUriReference}`);
    }
    const reference = trimmedUriReference.slice(1, -1);
    const parameters = {};
    if (rawLinkParameters.length === 0) {
      throw new Error(`Unexpected end of Link header parameters: ${rawLinkParameters.join(";")}`);
    }
    for (const rawParameter of rawLinkParameters) {
      const trimmedRawParameter = rawParameter.trim();
      const center = trimmedRawParameter.indexOf("=");
      if (center === -1) {
        throw new Error(`Failed to parse Link header: ${link}`);
      }
      const name = trimmedRawParameter.slice(0, center).trim();
      const value = trimmedRawParameter.slice(center + 1).trim();
      parameters[name] = value;
    }
    parsed.push({
      reference,
      parameters
    });
  }
  return parsed;
}

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/options.js
var [major, minor] = process2.versions.node.split(".").map(Number);
function validateSearchParameters(searchParameters) {
  for (const key in searchParameters) {
    const value = searchParameters[key];
    assert.any([dist_default.string, dist_default.number, dist_default.boolean, dist_default.null_, dist_default.undefined], value);
  }
}
var globalCache = /* @__PURE__ */ new Map();
var globalDnsCache;
var getGlobalDnsCache = () => {
  if (globalDnsCache) {
    return globalDnsCache;
  }
  globalDnsCache = new CacheableLookup();
  return globalDnsCache;
};
var defaultInternals = {
  request: void 0,
  agent: {
    http: void 0,
    https: void 0,
    http2: void 0
  },
  h2session: void 0,
  decompress: true,
  timeout: {
    connect: void 0,
    lookup: void 0,
    read: void 0,
    request: void 0,
    response: void 0,
    secureConnect: void 0,
    send: void 0,
    socket: void 0
  },
  prefixUrl: "",
  body: void 0,
  form: void 0,
  json: void 0,
  cookieJar: void 0,
  ignoreInvalidCookies: false,
  searchParams: void 0,
  dnsLookup: void 0,
  dnsCache: void 0,
  context: {},
  hooks: {
    init: [],
    beforeRequest: [],
    beforeError: [],
    beforeRedirect: [],
    beforeRetry: [],
    afterResponse: []
  },
  followRedirect: true,
  maxRedirects: 10,
  cache: void 0,
  throwHttpErrors: true,
  username: "",
  password: "",
  http2: false,
  allowGetBody: false,
  headers: {
    "user-agent": "got (https://github.com/sindresorhus/got)"
  },
  methodRewriting: false,
  dnsLookupIpVersion: void 0,
  parseJson: JSON.parse,
  stringifyJson: JSON.stringify,
  retry: {
    limit: 2,
    methods: [
      "GET",
      "PUT",
      "HEAD",
      "DELETE",
      "OPTIONS",
      "TRACE"
    ],
    statusCodes: [
      408,
      413,
      429,
      500,
      502,
      503,
      504,
      521,
      522,
      524
    ],
    errorCodes: [
      "ETIMEDOUT",
      "ECONNRESET",
      "EADDRINUSE",
      "ECONNREFUSED",
      "EPIPE",
      "ENOTFOUND",
      "ENETUNREACH",
      "EAI_AGAIN"
    ],
    maxRetryAfter: void 0,
    calculateDelay: ({ computedValue }) => computedValue,
    backoffLimit: Number.POSITIVE_INFINITY,
    noise: 100
  },
  localAddress: void 0,
  method: "GET",
  createConnection: void 0,
  cacheOptions: {
    shared: void 0,
    cacheHeuristic: void 0,
    immutableMinTimeToLive: void 0,
    ignoreCargoCult: void 0
  },
  https: {
    alpnProtocols: void 0,
    rejectUnauthorized: void 0,
    checkServerIdentity: void 0,
    certificateAuthority: void 0,
    key: void 0,
    certificate: void 0,
    passphrase: void 0,
    pfx: void 0,
    ciphers: void 0,
    honorCipherOrder: void 0,
    minVersion: void 0,
    maxVersion: void 0,
    signatureAlgorithms: void 0,
    tlsSessionLifetime: void 0,
    dhparam: void 0,
    ecdhCurve: void 0,
    certificateRevocationLists: void 0
  },
  encoding: void 0,
  resolveBodyOnly: false,
  isStream: false,
  responseType: "text",
  url: void 0,
  pagination: {
    transform(response) {
      if (response.request.options.responseType === "json") {
        return response.body;
      }
      return JSON.parse(response.body);
    },
    paginate({ response }) {
      const rawLinkHeader = response.headers.link;
      if (typeof rawLinkHeader !== "string" || rawLinkHeader.trim() === "") {
        return false;
      }
      const parsed = parseLinkHeader(rawLinkHeader);
      const next = parsed.find((entry) => entry.parameters.rel === "next" || entry.parameters.rel === '"next"');
      if (next) {
        return {
          url: new URL2(next.reference, response.url)
        };
      }
      return false;
    },
    filter: () => true,
    shouldContinue: () => true,
    countLimit: Number.POSITIVE_INFINITY,
    backoff: 0,
    requestLimit: 1e4,
    stackAllItems: false
  },
  setHost: true,
  maxHeaderSize: void 0,
  signal: void 0,
  enableUnixSockets: true
};
var cloneInternals = (internals) => {
  const { hooks, retry } = internals;
  const result = {
    ...internals,
    context: { ...internals.context },
    cacheOptions: { ...internals.cacheOptions },
    https: { ...internals.https },
    agent: { ...internals.agent },
    headers: { ...internals.headers },
    retry: {
      ...retry,
      errorCodes: [...retry.errorCodes],
      methods: [...retry.methods],
      statusCodes: [...retry.statusCodes]
    },
    timeout: { ...internals.timeout },
    hooks: {
      init: [...hooks.init],
      beforeRequest: [...hooks.beforeRequest],
      beforeError: [...hooks.beforeError],
      beforeRedirect: [...hooks.beforeRedirect],
      beforeRetry: [...hooks.beforeRetry],
      afterResponse: [...hooks.afterResponse]
    },
    searchParams: internals.searchParams ? new URLSearchParams(internals.searchParams) : void 0,
    pagination: { ...internals.pagination }
  };
  if (result.url !== void 0) {
    result.prefixUrl = "";
  }
  return result;
};
var cloneRaw = (raw) => {
  const { hooks, retry } = raw;
  const result = { ...raw };
  if (dist_default.object(raw.context)) {
    result.context = { ...raw.context };
  }
  if (dist_default.object(raw.cacheOptions)) {
    result.cacheOptions = { ...raw.cacheOptions };
  }
  if (dist_default.object(raw.https)) {
    result.https = { ...raw.https };
  }
  if (dist_default.object(raw.cacheOptions)) {
    result.cacheOptions = { ...result.cacheOptions };
  }
  if (dist_default.object(raw.agent)) {
    result.agent = { ...raw.agent };
  }
  if (dist_default.object(raw.headers)) {
    result.headers = { ...raw.headers };
  }
  if (dist_default.object(retry)) {
    result.retry = { ...retry };
    if (dist_default.array(retry.errorCodes)) {
      result.retry.errorCodes = [...retry.errorCodes];
    }
    if (dist_default.array(retry.methods)) {
      result.retry.methods = [...retry.methods];
    }
    if (dist_default.array(retry.statusCodes)) {
      result.retry.statusCodes = [...retry.statusCodes];
    }
  }
  if (dist_default.object(raw.timeout)) {
    result.timeout = { ...raw.timeout };
  }
  if (dist_default.object(hooks)) {
    result.hooks = {
      ...hooks
    };
    if (dist_default.array(hooks.init)) {
      result.hooks.init = [...hooks.init];
    }
    if (dist_default.array(hooks.beforeRequest)) {
      result.hooks.beforeRequest = [...hooks.beforeRequest];
    }
    if (dist_default.array(hooks.beforeError)) {
      result.hooks.beforeError = [...hooks.beforeError];
    }
    if (dist_default.array(hooks.beforeRedirect)) {
      result.hooks.beforeRedirect = [...hooks.beforeRedirect];
    }
    if (dist_default.array(hooks.beforeRetry)) {
      result.hooks.beforeRetry = [...hooks.beforeRetry];
    }
    if (dist_default.array(hooks.afterResponse)) {
      result.hooks.afterResponse = [...hooks.afterResponse];
    }
  }
  if (dist_default.object(raw.pagination)) {
    result.pagination = { ...raw.pagination };
  }
  return result;
};
var getHttp2TimeoutOption = (internals) => {
  const delays = [internals.timeout.socket, internals.timeout.connect, internals.timeout.lookup, internals.timeout.request, internals.timeout.secureConnect].filter((delay2) => typeof delay2 === "number");
  if (delays.length > 0) {
    return Math.min(...delays);
  }
  return void 0;
};
var init = (options, withOptions, self) => {
  var _a;
  const initHooks = (_a = options.hooks) == null ? void 0 : _a.init;
  if (initHooks) {
    for (const hook of initHooks) {
      hook(withOptions, self);
    }
  }
};
var Options = class {
  constructor(input, options, defaults2) {
    Object.defineProperty(this, "_unixOptions", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_internals", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_merging", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_init", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    assert.any([dist_default.string, dist_default.urlInstance, dist_default.object, dist_default.undefined], input);
    assert.any([dist_default.object, dist_default.undefined], options);
    assert.any([dist_default.object, dist_default.undefined], defaults2);
    if (input instanceof Options || options instanceof Options) {
      throw new TypeError("The defaults must be passed as the third argument");
    }
    this._internals = cloneInternals((defaults2 == null ? void 0 : defaults2._internals) ?? defaults2 ?? defaultInternals);
    this._init = [...(defaults2 == null ? void 0 : defaults2._init) ?? []];
    this._merging = false;
    this._unixOptions = void 0;
    try {
      if (dist_default.plainObject(input)) {
        try {
          this.merge(input);
          this.merge(options);
        } finally {
          this.url = input.url;
        }
      } else {
        try {
          this.merge(options);
        } finally {
          if ((options == null ? void 0 : options.url) !== void 0) {
            if (input === void 0) {
              this.url = options.url;
            } else {
              throw new TypeError("The `url` option is mutually exclusive with the `input` argument");
            }
          } else if (input !== void 0) {
            this.url = input;
          }
        }
      }
    } catch (error) {
      error.options = this;
      throw error;
    }
  }
  merge(options) {
    if (!options) {
      return;
    }
    if (options instanceof Options) {
      for (const init2 of options._init) {
        this.merge(init2);
      }
      return;
    }
    options = cloneRaw(options);
    init(this, options, this);
    init(options, options, this);
    this._merging = true;
    if ("isStream" in options) {
      this.isStream = options.isStream;
    }
    try {
      let push = false;
      for (const key in options) {
        if (key === "mutableDefaults" || key === "handlers") {
          continue;
        }
        if (key === "url") {
          continue;
        }
        if (!(key in this)) {
          throw new Error(`Unexpected option: ${key}`);
        }
        this[key] = options[key];
        push = true;
      }
      if (push) {
        this._init.push(options);
      }
    } finally {
      this._merging = false;
    }
  }
  /**
      Custom request function.
      The main purpose of this is to [support HTTP2 using a wrapper](https://github.com/szmarczak/http2-wrapper).
  
      @default http.request | https.request
      */
  get request() {
    return this._internals.request;
  }
  set request(value) {
    assert.any([dist_default.function_, dist_default.undefined], value);
    this._internals.request = value;
  }
  /**
      An object representing `http`, `https` and `http2` keys for [`http.Agent`](https://nodejs.org/api/http.html#http_class_http_agent), [`https.Agent`](https://nodejs.org/api/https.html#https_class_https_agent) and [`http2wrapper.Agent`](https://github.com/szmarczak/http2-wrapper#new-http2agentoptions) instance.
      This is necessary because a request to one protocol might redirect to another.
      In such a scenario, Got will switch over to the right protocol agent for you.
  
      If a key is not present, it will default to a global agent.
  
      @example
      ```
      import got from 'got';
      import HttpAgent from 'agentkeepalive';
  
      const {HttpsAgent} = HttpAgent;
  
      await got('https://sindresorhus.com', {
          agent: {
              http: new HttpAgent(),
              https: new HttpsAgent()
          }
      });
      ```
      */
  get agent() {
    return this._internals.agent;
  }
  set agent(value) {
    assert.plainObject(value);
    for (const key in value) {
      if (!(key in this._internals.agent)) {
        throw new TypeError(`Unexpected agent option: ${key}`);
      }
      assert.any([dist_default.object, dist_default.undefined], value[key]);
    }
    if (this._merging) {
      Object.assign(this._internals.agent, value);
    } else {
      this._internals.agent = { ...value };
    }
  }
  get h2session() {
    return this._internals.h2session;
  }
  set h2session(value) {
    this._internals.h2session = value;
  }
  /**
      Decompress the response automatically.
  
      This will set the `accept-encoding` header to `gzip, deflate, br` unless you set it yourself.
  
      If this is disabled, a compressed response is returned as a `Buffer`.
      This may be useful if you want to handle decompression yourself or stream the raw compressed data.
  
      @default true
      */
  get decompress() {
    return this._internals.decompress;
  }
  set decompress(value) {
    assert.boolean(value);
    this._internals.decompress = value;
  }
  /**
      Milliseconds to wait for the server to end the response before aborting the request with `got.TimeoutError` error (a.k.a. `request` property).
      By default, there's no timeout.
  
      This also accepts an `object` with the following fields to constrain the duration of each phase of the request lifecycle:
  
      - `lookup` starts when a socket is assigned and ends when the hostname has been resolved.
          Does not apply when using a Unix domain socket.
      - `connect` starts when `lookup` completes (or when the socket is assigned if lookup does not apply to the request) and ends when the socket is connected.
      - `secureConnect` starts when `connect` completes and ends when the handshaking process completes (HTTPS only).
      - `socket` starts when the socket is connected. See [request.setTimeout](https://nodejs.org/api/http.html#http_request_settimeout_timeout_callback).
      - `response` starts when the request has been written to the socket and ends when the response headers are received.
      - `send` starts when the socket is connected and ends with the request has been written to the socket.
      - `request` starts when the request is initiated and ends when the response's end event fires.
      */
  get timeout() {
    return this._internals.timeout;
  }
  set timeout(value) {
    assert.plainObject(value);
    for (const key in value) {
      if (!(key in this._internals.timeout)) {
        throw new Error(`Unexpected timeout option: ${key}`);
      }
      assert.any([dist_default.number, dist_default.undefined], value[key]);
    }
    if (this._merging) {
      Object.assign(this._internals.timeout, value);
    } else {
      this._internals.timeout = { ...value };
    }
  }
  /**
      When specified, `prefixUrl` will be prepended to `url`.
      The prefix can be any valid URL, either relative or absolute.
      A trailing slash `/` is optional - one will be added automatically.
  
      __Note__: `prefixUrl` will be ignored if the `url` argument is a URL instance.
  
      __Note__: Leading slashes in `input` are disallowed when using this option to enforce consistency and avoid confusion.
      For example, when the prefix URL is `https://example.com/foo` and the input is `/bar`, there's ambiguity whether the resulting URL would become `https://example.com/foo/bar` or `https://example.com/bar`.
      The latter is used by browsers.
  
      __Tip__: Useful when used with `got.extend()` to create niche-specific Got instances.
  
      __Tip__: You can change `prefixUrl` using hooks as long as the URL still includes the `prefixUrl`.
      If the URL doesn't include it anymore, it will throw.
  
      @example
      ```
      import got from 'got';
  
      await got('unicorn', {prefixUrl: 'https://cats.com'});
      //=> 'https://cats.com/unicorn'
  
      const instance = got.extend({
          prefixUrl: 'https://google.com'
      });
  
      await instance('unicorn', {
          hooks: {
              beforeRequest: [
                  options => {
                      options.prefixUrl = 'https://cats.com';
                  }
              ]
          }
      });
      //=> 'https://cats.com/unicorn'
      ```
      */
  get prefixUrl() {
    return this._internals.prefixUrl;
  }
  set prefixUrl(value) {
    assert.any([dist_default.string, dist_default.urlInstance], value);
    if (value === "") {
      this._internals.prefixUrl = "";
      return;
    }
    value = value.toString();
    if (!value.endsWith("/")) {
      value += "/";
    }
    if (this._internals.prefixUrl && this._internals.url) {
      const { href } = this._internals.url;
      this._internals.url.href = value + href.slice(this._internals.prefixUrl.length);
    }
    this._internals.prefixUrl = value;
  }
  /**
      __Note #1__: The `body` option cannot be used with the `json` or `form` option.
  
      __Note #2__: If you provide this option, `got.stream()` will be read-only.
  
      __Note #3__: If you provide a payload with the `GET` or `HEAD` method, it will throw a `TypeError` unless the method is `GET` and the `allowGetBody` option is set to `true`.
  
      __Note #4__: This option is not enumerable and will not be merged with the instance defaults.
  
      The `content-length` header will be automatically set if `body` is a `string` / `Buffer` / [`FormData`](https://developer.mozilla.org/en-US/docs/Web/API/FormData) / [`form-data` instance](https://github.com/form-data/form-data), and `content-length` and `transfer-encoding` are not manually set in `options.headers`.
  
      Since Got 12, the `content-length` is not automatically set when `body` is a `fs.createReadStream`.
      */
  get body() {
    return this._internals.body;
  }
  set body(value) {
    assert.any([dist_default.string, dist_default.buffer, dist_default.nodeStream, dist_default.generator, dist_default.asyncGenerator, isFormData, dist_default.undefined], value);
    if (dist_default.nodeStream(value)) {
      assert.truthy(value.readable);
    }
    if (value !== void 0) {
      assert.undefined(this._internals.form);
      assert.undefined(this._internals.json);
    }
    this._internals.body = value;
  }
  /**
      The form body is converted to a query string using [`(new URLSearchParams(object)).toString()`](https://nodejs.org/api/url.html#url_constructor_new_urlsearchparams_obj).
  
      If the `Content-Type` header is not present, it will be set to `application/x-www-form-urlencoded`.
  
      __Note #1__: If you provide this option, `got.stream()` will be read-only.
  
      __Note #2__: This option is not enumerable and will not be merged with the instance defaults.
      */
  get form() {
    return this._internals.form;
  }
  set form(value) {
    assert.any([dist_default.plainObject, dist_default.undefined], value);
    if (value !== void 0) {
      assert.undefined(this._internals.body);
      assert.undefined(this._internals.json);
    }
    this._internals.form = value;
  }
  /**
      JSON body. If the `Content-Type` header is not set, it will be set to `application/json`.
  
      __Note #1__: If you provide this option, `got.stream()` will be read-only.
  
      __Note #2__: This option is not enumerable and will not be merged with the instance defaults.
      */
  get json() {
    return this._internals.json;
  }
  set json(value) {
    if (value !== void 0) {
      assert.undefined(this._internals.body);
      assert.undefined(this._internals.form);
    }
    this._internals.json = value;
  }
  /**
      The URL to request, as a string, a [`https.request` options object](https://nodejs.org/api/https.html#https_https_request_options_callback), or a [WHATWG `URL`](https://nodejs.org/api/url.html#url_class_url).
  
      Properties from `options` will override properties in the parsed `url`.
  
      If no protocol is specified, it will throw a `TypeError`.
  
      __Note__: The query string is **not** parsed as search params.
  
      @example
      ```
      await got('https://example.com/?query=a b'); //=> https://example.com/?query=a%20b
      await got('https://example.com/', {searchParams: {query: 'a b'}}); //=> https://example.com/?query=a+b
  
      // The query string is overridden by `searchParams`
      await got('https://example.com/?query=a b', {searchParams: {query: 'a b'}}); //=> https://example.com/?query=a+b
      ```
      */
  get url() {
    return this._internals.url;
  }
  set url(value) {
    assert.any([dist_default.string, dist_default.urlInstance, dist_default.undefined], value);
    if (value === void 0) {
      this._internals.url = void 0;
      return;
    }
    if (dist_default.string(value) && value.startsWith("/")) {
      throw new Error("`url` must not start with a slash");
    }
    const urlString = `${this.prefixUrl}${value.toString()}`;
    const url = new URL2(urlString);
    this._internals.url = url;
    decodeURI(urlString);
    if (url.protocol === "unix:") {
      url.href = `http://unix${url.pathname}${url.search}`;
    }
    if (url.protocol !== "http:" && url.protocol !== "https:") {
      const error = new Error(`Unsupported protocol: ${url.protocol}`);
      error.code = "ERR_UNSUPPORTED_PROTOCOL";
      throw error;
    }
    if (this._internals.username) {
      url.username = this._internals.username;
      this._internals.username = "";
    }
    if (this._internals.password) {
      url.password = this._internals.password;
      this._internals.password = "";
    }
    if (this._internals.searchParams) {
      url.search = this._internals.searchParams.toString();
      this._internals.searchParams = void 0;
    }
    if (url.hostname === "unix") {
      if (!this._internals.enableUnixSockets) {
        throw new Error("Using UNIX domain sockets but option `enableUnixSockets` is not enabled");
      }
      const matches = /(?<socketPath>.+?):(?<path>.+)/.exec(`${url.pathname}${url.search}`);
      if (matches == null ? void 0 : matches.groups) {
        const { socketPath, path } = matches.groups;
        this._unixOptions = {
          socketPath,
          path,
          host: ""
        };
      } else {
        this._unixOptions = void 0;
      }
      return;
    }
    this._unixOptions = void 0;
  }
  /**
      Cookie support. You don't have to care about parsing or how to store them.
  
      __Note__: If you provide this option, `options.headers.cookie` will be overridden.
      */
  get cookieJar() {
    return this._internals.cookieJar;
  }
  set cookieJar(value) {
    assert.any([dist_default.object, dist_default.undefined], value);
    if (value === void 0) {
      this._internals.cookieJar = void 0;
      return;
    }
    let { setCookie, getCookieString } = value;
    assert.function_(setCookie);
    assert.function_(getCookieString);
    if (setCookie.length === 4 && getCookieString.length === 0) {
      setCookie = promisify3(setCookie.bind(value));
      getCookieString = promisify3(getCookieString.bind(value));
      this._internals.cookieJar = {
        setCookie,
        getCookieString
      };
    } else {
      this._internals.cookieJar = value;
    }
  }
  /**
      You can abort the `request` using [`AbortController`](https://developer.mozilla.org/en-US/docs/Web/API/AbortController).
  
      *Requires Node.js 16 or later.*
  
      @example
      ```
      import got from 'got';
  
      const abortController = new AbortController();
  
      const request = got('https://httpbin.org/anything', {
          signal: abortController.signal
      });
  
      setTimeout(() => {
          abortController.abort();
      }, 100);
      ```
      */
  // TODO: Replace `any` with `AbortSignal` when targeting Node 16.
  get signal() {
    return this._internals.signal;
  }
  // TODO: Replace `any` with `AbortSignal` when targeting Node 16.
  set signal(value) {
    assert.object(value);
    this._internals.signal = value;
  }
  /**
      Ignore invalid cookies instead of throwing an error.
      Only useful when the `cookieJar` option has been set. Not recommended.
  
      @default false
      */
  get ignoreInvalidCookies() {
    return this._internals.ignoreInvalidCookies;
  }
  set ignoreInvalidCookies(value) {
    assert.boolean(value);
    this._internals.ignoreInvalidCookies = value;
  }
  /**
      Query string that will be added to the request URL.
      This will override the query string in `url`.
  
      If you need to pass in an array, you can do it using a `URLSearchParams` instance.
  
      @example
      ```
      import got from 'got';
  
      const searchParams = new URLSearchParams([['key', 'a'], ['key', 'b']]);
  
      await got('https://example.com', {searchParams});
  
      console.log(searchParams.toString());
      //=> 'key=a&key=b'
      ```
      */
  get searchParams() {
    if (this._internals.url) {
      return this._internals.url.searchParams;
    }
    if (this._internals.searchParams === void 0) {
      this._internals.searchParams = new URLSearchParams();
    }
    return this._internals.searchParams;
  }
  set searchParams(value) {
    assert.any([dist_default.string, dist_default.object, dist_default.undefined], value);
    const url = this._internals.url;
    if (value === void 0) {
      this._internals.searchParams = void 0;
      if (url) {
        url.search = "";
      }
      return;
    }
    const searchParameters = this.searchParams;
    let updated;
    if (dist_default.string(value)) {
      updated = new URLSearchParams(value);
    } else if (value instanceof URLSearchParams) {
      updated = value;
    } else {
      validateSearchParameters(value);
      updated = new URLSearchParams();
      for (const key in value) {
        const entry = value[key];
        if (entry === null) {
          updated.append(key, "");
        } else if (entry === void 0) {
          searchParameters.delete(key);
        } else {
          updated.append(key, entry);
        }
      }
    }
    if (this._merging) {
      for (const key of updated.keys()) {
        searchParameters.delete(key);
      }
      for (const [key, value2] of updated) {
        searchParameters.append(key, value2);
      }
    } else if (url) {
      url.search = searchParameters.toString();
    } else {
      this._internals.searchParams = searchParameters;
    }
  }
  get searchParameters() {
    throw new Error("The `searchParameters` option does not exist. Use `searchParams` instead.");
  }
  set searchParameters(_value) {
    throw new Error("The `searchParameters` option does not exist. Use `searchParams` instead.");
  }
  get dnsLookup() {
    return this._internals.dnsLookup;
  }
  set dnsLookup(value) {
    assert.any([dist_default.function_, dist_default.undefined], value);
    this._internals.dnsLookup = value;
  }
  /**
      An instance of [`CacheableLookup`](https://github.com/szmarczak/cacheable-lookup) used for making DNS lookups.
      Useful when making lots of requests to different *public* hostnames.
  
      `CacheableLookup` uses `dns.resolver4(..)` and `dns.resolver6(...)` under the hood and fall backs to `dns.lookup(...)` when the first two fail, which may lead to additional delay.
  
      __Note__: This should stay disabled when making requests to internal hostnames such as `localhost`, `database.local` etc.
  
      @default false
      */
  get dnsCache() {
    return this._internals.dnsCache;
  }
  set dnsCache(value) {
    assert.any([dist_default.object, dist_default.boolean, dist_default.undefined], value);
    if (value === true) {
      this._internals.dnsCache = getGlobalDnsCache();
    } else if (value === false) {
      this._internals.dnsCache = void 0;
    } else {
      this._internals.dnsCache = value;
    }
  }
  /**
      User data. `context` is shallow merged and enumerable. If it contains non-enumerable properties they will NOT be merged.
  
      @example
      ```
      import got from 'got';
  
      const instance = got.extend({
          hooks: {
              beforeRequest: [
                  options => {
                      if (!options.context || !options.context.token) {
                          throw new Error('Token required');
                      }
  
                      options.headers.token = options.context.token;
                  }
              ]
          }
      });
  
      const context = {
          token: 'secret'
      };
  
      const response = await instance('https://httpbin.org/headers', {context});
  
      // Let's see the headers
      console.log(response.body);
      ```
      */
  get context() {
    return this._internals.context;
  }
  set context(value) {
    assert.object(value);
    if (this._merging) {
      Object.assign(this._internals.context, value);
    } else {
      this._internals.context = { ...value };
    }
  }
  /**
  Hooks allow modifications during the request lifecycle.
  Hook functions may be async and are run serially.
  */
  get hooks() {
    return this._internals.hooks;
  }
  set hooks(value) {
    assert.object(value);
    for (const knownHookEvent in value) {
      if (!(knownHookEvent in this._internals.hooks)) {
        throw new Error(`Unexpected hook event: ${knownHookEvent}`);
      }
      const typedKnownHookEvent = knownHookEvent;
      const hooks = value[typedKnownHookEvent];
      assert.any([dist_default.array, dist_default.undefined], hooks);
      if (hooks) {
        for (const hook of hooks) {
          assert.function_(hook);
        }
      }
      if (this._merging) {
        if (hooks) {
          this._internals.hooks[typedKnownHookEvent].push(...hooks);
        }
      } else {
        if (!hooks) {
          throw new Error(`Missing hook event: ${knownHookEvent}`);
        }
        this._internals.hooks[knownHookEvent] = [...hooks];
      }
    }
  }
  /**
      Defines if redirect responses should be followed automatically.
  
      Note that if a `303` is sent by the server in response to any request type (`POST`, `DELETE`, etc.), Got will automatically request the resource pointed to in the location header via `GET`.
      This is in accordance with [the spec](https://tools.ietf.org/html/rfc7231#section-6.4.4). You can optionally turn on this behavior also for other redirect codes - see `methodRewriting`.
  
      @default true
      */
  get followRedirect() {
    return this._internals.followRedirect;
  }
  set followRedirect(value) {
    assert.boolean(value);
    this._internals.followRedirect = value;
  }
  get followRedirects() {
    throw new TypeError("The `followRedirects` option does not exist. Use `followRedirect` instead.");
  }
  set followRedirects(_value) {
    throw new TypeError("The `followRedirects` option does not exist. Use `followRedirect` instead.");
  }
  /**
      If exceeded, the request will be aborted and a `MaxRedirectsError` will be thrown.
  
      @default 10
      */
  get maxRedirects() {
    return this._internals.maxRedirects;
  }
  set maxRedirects(value) {
    assert.number(value);
    this._internals.maxRedirects = value;
  }
  /**
      A cache adapter instance for storing cached response data.
  
      @default false
      */
  get cache() {
    return this._internals.cache;
  }
  set cache(value) {
    assert.any([dist_default.object, dist_default.string, dist_default.boolean, dist_default.undefined], value);
    if (value === true) {
      this._internals.cache = globalCache;
    } else if (value === false) {
      this._internals.cache = void 0;
    } else {
      this._internals.cache = value;
    }
  }
  /**
      Determines if a `got.HTTPError` is thrown for unsuccessful responses.
  
      If this is disabled, requests that encounter an error status code will be resolved with the `response` instead of throwing.
      This may be useful if you are checking for resource availability and are expecting error responses.
  
      @default true
      */
  get throwHttpErrors() {
    return this._internals.throwHttpErrors;
  }
  set throwHttpErrors(value) {
    assert.boolean(value);
    this._internals.throwHttpErrors = value;
  }
  get username() {
    const url = this._internals.url;
    const value = url ? url.username : this._internals.username;
    return decodeURIComponent(value);
  }
  set username(value) {
    assert.string(value);
    const url = this._internals.url;
    const fixedValue = encodeURIComponent(value);
    if (url) {
      url.username = fixedValue;
    } else {
      this._internals.username = fixedValue;
    }
  }
  get password() {
    const url = this._internals.url;
    const value = url ? url.password : this._internals.password;
    return decodeURIComponent(value);
  }
  set password(value) {
    assert.string(value);
    const url = this._internals.url;
    const fixedValue = encodeURIComponent(value);
    if (url) {
      url.password = fixedValue;
    } else {
      this._internals.password = fixedValue;
    }
  }
  /**
      If set to `true`, Got will additionally accept HTTP2 requests.
  
      It will choose either HTTP/1.1 or HTTP/2 depending on the ALPN protocol.
  
      __Note__: This option requires Node.js 15.10.0 or newer as HTTP/2 support on older Node.js versions is very buggy.
  
      __Note__: Overriding `options.request` will disable HTTP2 support.
  
      @default false
  
      @example
      ```
      import got from 'got';
  
      const {headers} = await got('https://nghttp2.org/httpbin/anything', {http2: true});
  
      console.log(headers.via);
      //=> '2 nghttpx'
      ```
      */
  get http2() {
    return this._internals.http2;
  }
  set http2(value) {
    assert.boolean(value);
    this._internals.http2 = value;
  }
  /**
      Set this to `true` to allow sending body for the `GET` method.
      However, the [HTTP/2 specification](https://tools.ietf.org/html/rfc7540#section-8.1.3) says that `An HTTP GET request includes request header fields and no payload body`, therefore when using the HTTP/2 protocol this option will have no effect.
      This option is only meant to interact with non-compliant servers when you have no other choice.
  
      __Note__: The [RFC 7231](https://tools.ietf.org/html/rfc7231#section-4.3.1) doesn't specify any particular behavior for the GET method having a payload, therefore __it's considered an [anti-pattern](https://en.wikipedia.org/wiki/Anti-pattern)__.
  
      @default false
      */
  get allowGetBody() {
    return this._internals.allowGetBody;
  }
  set allowGetBody(value) {
    assert.boolean(value);
    this._internals.allowGetBody = value;
  }
  /**
      Request headers.
  
      Existing headers will be overwritten. Headers set to `undefined` will be omitted.
  
      @default {}
      */
  get headers() {
    return this._internals.headers;
  }
  set headers(value) {
    assert.plainObject(value);
    if (this._merging) {
      Object.assign(this._internals.headers, lowercaseKeys(value));
    } else {
      this._internals.headers = lowercaseKeys(value);
    }
  }
  /**
      Specifies if the HTTP request method should be [rewritten as `GET`](https://tools.ietf.org/html/rfc7231#section-6.4) on redirects.
  
      As the [specification](https://tools.ietf.org/html/rfc7231#section-6.4) prefers to rewrite the HTTP method only on `303` responses, this is Got's default behavior.
      Setting `methodRewriting` to `true` will also rewrite `301` and `302` responses, as allowed by the spec. This is the behavior followed by `curl` and browsers.
  
      __Note__: Got never performs method rewriting on `307` and `308` responses, as this is [explicitly prohibited by the specification](https://www.rfc-editor.org/rfc/rfc7231#section-6.4.7).
  
      @default false
      */
  get methodRewriting() {
    return this._internals.methodRewriting;
  }
  set methodRewriting(value) {
    assert.boolean(value);
    this._internals.methodRewriting = value;
  }
  /**
      Indicates which DNS record family to use.
  
      Values:
      - `undefined`: IPv4 (if present) or IPv6
      - `4`: Only IPv4
      - `6`: Only IPv6
  
      @default undefined
      */
  get dnsLookupIpVersion() {
    return this._internals.dnsLookupIpVersion;
  }
  set dnsLookupIpVersion(value) {
    if (value !== void 0 && value !== 4 && value !== 6) {
      throw new TypeError(`Invalid DNS lookup IP version: ${value}`);
    }
    this._internals.dnsLookupIpVersion = value;
  }
  /**
      A function used to parse JSON responses.
  
      @example
      ```
      import got from 'got';
      import Bourne from '@hapi/bourne';
  
      const parsed = await got('https://example.com', {
          parseJson: text => Bourne.parse(text)
      }).json();
  
      console.log(parsed);
      ```
      */
  get parseJson() {
    return this._internals.parseJson;
  }
  set parseJson(value) {
    assert.function_(value);
    this._internals.parseJson = value;
  }
  /**
      A function used to stringify the body of JSON requests.
  
      @example
      ```
      import got from 'got';
  
      await got.post('https://example.com', {
          stringifyJson: object => JSON.stringify(object, (key, value) => {
              if (key.startsWith('_')) {
                  return;
              }
  
              return value;
          }),
          json: {
              some: 'payload',
              _ignoreMe: 1234
          }
      });
      ```
  
      @example
      ```
      import got from 'got';
  
      await got.post('https://example.com', {
          stringifyJson: object => JSON.stringify(object, (key, value) => {
              if (typeof value === 'number') {
                  return value.toString();
              }
  
              return value;
          }),
          json: {
              some: 'payload',
              number: 1
          }
      });
      ```
      */
  get stringifyJson() {
    return this._internals.stringifyJson;
  }
  set stringifyJson(value) {
    assert.function_(value);
    this._internals.stringifyJson = value;
  }
  /**
      An object representing `limit`, `calculateDelay`, `methods`, `statusCodes`, `maxRetryAfter` and `errorCodes` fields for maximum retry count, retry handler, allowed methods, allowed status codes, maximum [`Retry-After`](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Retry-After) time and allowed error codes.
  
      Delays between retries counts with function `1000 * Math.pow(2, retry) + Math.random() * 100`, where `retry` is attempt number (starts from 1).
  
      The `calculateDelay` property is a `function` that receives an object with `attemptCount`, `retryOptions`, `error` and `computedValue` properties for current retry count, the retry options, error and default computed value.
      The function must return a delay in milliseconds (or a Promise resolving with it) (`0` return value cancels retry).
  
      By default, it retries *only* on the specified methods, status codes, and on these network errors:
  
      - `ETIMEDOUT`: One of the [timeout](#timeout) limits were reached.
      - `ECONNRESET`: Connection was forcibly closed by a peer.
      - `EADDRINUSE`: Could not bind to any free port.
      - `ECONNREFUSED`: Connection was refused by the server.
      - `EPIPE`: The remote side of the stream being written has been closed.
      - `ENOTFOUND`: Couldn't resolve the hostname to an IP address.
      - `ENETUNREACH`: No internet connection.
      - `EAI_AGAIN`: DNS lookup timed out.
  
      __Note__: If `maxRetryAfter` is set to `undefined`, it will use `options.timeout`.
      __Note__: If [`Retry-After`](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Retry-After) header is greater than `maxRetryAfter`, it will cancel the request.
      */
  get retry() {
    return this._internals.retry;
  }
  set retry(value) {
    assert.plainObject(value);
    assert.any([dist_default.function_, dist_default.undefined], value.calculateDelay);
    assert.any([dist_default.number, dist_default.undefined], value.maxRetryAfter);
    assert.any([dist_default.number, dist_default.undefined], value.limit);
    assert.any([dist_default.array, dist_default.undefined], value.methods);
    assert.any([dist_default.array, dist_default.undefined], value.statusCodes);
    assert.any([dist_default.array, dist_default.undefined], value.errorCodes);
    assert.any([dist_default.number, dist_default.undefined], value.noise);
    if (value.noise && Math.abs(value.noise) > 100) {
      throw new Error(`The maximum acceptable retry noise is +/- 100ms, got ${value.noise}`);
    }
    for (const key in value) {
      if (!(key in this._internals.retry)) {
        throw new Error(`Unexpected retry option: ${key}`);
      }
    }
    if (this._merging) {
      Object.assign(this._internals.retry, value);
    } else {
      this._internals.retry = { ...value };
    }
    const { retry } = this._internals;
    retry.methods = [...new Set(retry.methods.map((method) => method.toUpperCase()))];
    retry.statusCodes = [...new Set(retry.statusCodes)];
    retry.errorCodes = [...new Set(retry.errorCodes)];
  }
  /**
      From `http.RequestOptions`.
  
      The IP address used to send the request from.
      */
  get localAddress() {
    return this._internals.localAddress;
  }
  set localAddress(value) {
    assert.any([dist_default.string, dist_default.undefined], value);
    this._internals.localAddress = value;
  }
  /**
      The HTTP method used to make the request.
  
      @default 'GET'
      */
  get method() {
    return this._internals.method;
  }
  set method(value) {
    assert.string(value);
    this._internals.method = value.toUpperCase();
  }
  get createConnection() {
    return this._internals.createConnection;
  }
  set createConnection(value) {
    assert.any([dist_default.function_, dist_default.undefined], value);
    this._internals.createConnection = value;
  }
  /**
      From `http-cache-semantics`
  
      @default {}
      */
  get cacheOptions() {
    return this._internals.cacheOptions;
  }
  set cacheOptions(value) {
    assert.plainObject(value);
    assert.any([dist_default.boolean, dist_default.undefined], value.shared);
    assert.any([dist_default.number, dist_default.undefined], value.cacheHeuristic);
    assert.any([dist_default.number, dist_default.undefined], value.immutableMinTimeToLive);
    assert.any([dist_default.boolean, dist_default.undefined], value.ignoreCargoCult);
    for (const key in value) {
      if (!(key in this._internals.cacheOptions)) {
        throw new Error(`Cache option \`${key}\` does not exist`);
      }
    }
    if (this._merging) {
      Object.assign(this._internals.cacheOptions, value);
    } else {
      this._internals.cacheOptions = { ...value };
    }
  }
  /**
  Options for the advanced HTTPS API.
  */
  get https() {
    return this._internals.https;
  }
  set https(value) {
    assert.plainObject(value);
    assert.any([dist_default.boolean, dist_default.undefined], value.rejectUnauthorized);
    assert.any([dist_default.function_, dist_default.undefined], value.checkServerIdentity);
    assert.any([dist_default.string, dist_default.object, dist_default.array, dist_default.undefined], value.certificateAuthority);
    assert.any([dist_default.string, dist_default.object, dist_default.array, dist_default.undefined], value.key);
    assert.any([dist_default.string, dist_default.object, dist_default.array, dist_default.undefined], value.certificate);
    assert.any([dist_default.string, dist_default.undefined], value.passphrase);
    assert.any([dist_default.string, dist_default.buffer, dist_default.array, dist_default.undefined], value.pfx);
    assert.any([dist_default.array, dist_default.undefined], value.alpnProtocols);
    assert.any([dist_default.string, dist_default.undefined], value.ciphers);
    assert.any([dist_default.string, dist_default.buffer, dist_default.undefined], value.dhparam);
    assert.any([dist_default.string, dist_default.undefined], value.signatureAlgorithms);
    assert.any([dist_default.string, dist_default.undefined], value.minVersion);
    assert.any([dist_default.string, dist_default.undefined], value.maxVersion);
    assert.any([dist_default.boolean, dist_default.undefined], value.honorCipherOrder);
    assert.any([dist_default.number, dist_default.undefined], value.tlsSessionLifetime);
    assert.any([dist_default.string, dist_default.undefined], value.ecdhCurve);
    assert.any([dist_default.string, dist_default.buffer, dist_default.array, dist_default.undefined], value.certificateRevocationLists);
    for (const key in value) {
      if (!(key in this._internals.https)) {
        throw new Error(`HTTPS option \`${key}\` does not exist`);
      }
    }
    if (this._merging) {
      Object.assign(this._internals.https, value);
    } else {
      this._internals.https = { ...value };
    }
  }
  /**
      [Encoding](https://nodejs.org/api/buffer.html#buffer_buffers_and_character_encodings) to be used on `setEncoding` of the response data.
  
      To get a [`Buffer`](https://nodejs.org/api/buffer.html), you need to set `responseType` to `buffer` instead.
      Don't set this option to `null`.
  
      __Note__: This doesn't affect streams! Instead, you need to do `got.stream(...).setEncoding(encoding)`.
  
      @default 'utf-8'
      */
  get encoding() {
    return this._internals.encoding;
  }
  set encoding(value) {
    if (value === null) {
      throw new TypeError("To get a Buffer, set `options.responseType` to `buffer` instead");
    }
    assert.any([dist_default.string, dist_default.undefined], value);
    this._internals.encoding = value;
  }
  /**
      When set to `true` the promise will return the Response body instead of the Response object.
  
      @default false
      */
  get resolveBodyOnly() {
    return this._internals.resolveBodyOnly;
  }
  set resolveBodyOnly(value) {
    assert.boolean(value);
    this._internals.resolveBodyOnly = value;
  }
  /**
      Returns a `Stream` instead of a `Promise`.
      This is equivalent to calling `got.stream(url, options?)`.
  
      @default false
      */
  get isStream() {
    return this._internals.isStream;
  }
  set isStream(value) {
    assert.boolean(value);
    this._internals.isStream = value;
  }
  /**
      The parsing method.
  
      The promise also has `.text()`, `.json()` and `.buffer()` methods which return another Got promise for the parsed body.
  
      It's like setting the options to `{responseType: 'json', resolveBodyOnly: true}` but without affecting the main Got promise.
  
      __Note__: When using streams, this option is ignored.
  
      @example
      ```
      const responsePromise = got(url);
      const bufferPromise = responsePromise.buffer();
      const jsonPromise = responsePromise.json();
  
      const [response, buffer, json] = Promise.all([responsePromise, bufferPromise, jsonPromise]);
      // `response` is an instance of Got Response
      // `buffer` is an instance of Buffer
      // `json` is an object
      ```
  
      @example
      ```
      // This
      const body = await got(url).json();
  
      // is semantically the same as this
      const body = await got(url, {responseType: 'json', resolveBodyOnly: true});
      ```
      */
  get responseType() {
    return this._internals.responseType;
  }
  set responseType(value) {
    if (value === void 0) {
      this._internals.responseType = "text";
      return;
    }
    if (value !== "text" && value !== "buffer" && value !== "json") {
      throw new Error(`Invalid \`responseType\` option: ${value}`);
    }
    this._internals.responseType = value;
  }
  get pagination() {
    return this._internals.pagination;
  }
  set pagination(value) {
    assert.object(value);
    if (this._merging) {
      Object.assign(this._internals.pagination, value);
    } else {
      this._internals.pagination = value;
    }
  }
  get auth() {
    throw new Error("Parameter `auth` is deprecated. Use `username` / `password` instead.");
  }
  set auth(_value) {
    throw new Error("Parameter `auth` is deprecated. Use `username` / `password` instead.");
  }
  get setHost() {
    return this._internals.setHost;
  }
  set setHost(value) {
    assert.boolean(value);
    this._internals.setHost = value;
  }
  get maxHeaderSize() {
    return this._internals.maxHeaderSize;
  }
  set maxHeaderSize(value) {
    assert.any([dist_default.number, dist_default.undefined], value);
    this._internals.maxHeaderSize = value;
  }
  get enableUnixSockets() {
    return this._internals.enableUnixSockets;
  }
  set enableUnixSockets(value) {
    assert.boolean(value);
    this._internals.enableUnixSockets = value;
  }
  // eslint-disable-next-line @typescript-eslint/naming-convention
  toJSON() {
    return { ...this._internals };
  }
  [Symbol.for("nodejs.util.inspect.custom")](_depth, options) {
    return inspect(this._internals, options);
  }
  createNativeRequestOptions() {
    var _a;
    const internals = this._internals;
    const url = internals.url;
    let agent2;
    if (url.protocol === "https:") {
      agent2 = internals.http2 ? internals.agent : internals.agent.https;
    } else {
      agent2 = internals.agent.http;
    }
    const { https: https2 } = internals;
    let { pfx } = https2;
    if (dist_default.array(pfx) && dist_default.plainObject(pfx[0])) {
      pfx = pfx.map((object) => ({
        buf: object.buffer,
        passphrase: object.passphrase
      }));
    }
    return {
      ...internals.cacheOptions,
      ...this._unixOptions,
      // HTTPS options
      // eslint-disable-next-line @typescript-eslint/naming-convention
      ALPNProtocols: https2.alpnProtocols,
      ca: https2.certificateAuthority,
      cert: https2.certificate,
      key: https2.key,
      passphrase: https2.passphrase,
      pfx: https2.pfx,
      rejectUnauthorized: https2.rejectUnauthorized,
      checkServerIdentity: https2.checkServerIdentity ?? checkServerIdentity,
      ciphers: https2.ciphers,
      honorCipherOrder: https2.honorCipherOrder,
      minVersion: https2.minVersion,
      maxVersion: https2.maxVersion,
      sigalgs: https2.signatureAlgorithms,
      sessionTimeout: https2.tlsSessionLifetime,
      dhparam: https2.dhparam,
      ecdhCurve: https2.ecdhCurve,
      crl: https2.certificateRevocationLists,
      // HTTP options
      lookup: internals.dnsLookup ?? ((_a = internals.dnsCache) == null ? void 0 : _a.lookup),
      family: internals.dnsLookupIpVersion,
      agent: agent2,
      setHost: internals.setHost,
      method: internals.method,
      maxHeaderSize: internals.maxHeaderSize,
      localAddress: internals.localAddress,
      headers: internals.headers,
      createConnection: internals.createConnection,
      timeout: internals.http2 ? getHttp2TimeoutOption(internals) : void 0,
      // HTTP/2 options
      h2session: internals.h2session
    };
  }
  getRequestFunction() {
    const url = this._internals.url;
    const { request } = this._internals;
    if (!request && url) {
      return this.getFallbackRequestFunction();
    }
    return request;
  }
  getFallbackRequestFunction() {
    const url = this._internals.url;
    if (!url) {
      return;
    }
    if (url.protocol === "https:") {
      if (this._internals.http2) {
        if (major < 15 || major === 15 && minor < 10) {
          const error = new Error("To use the `http2` option, install Node.js 15.10.0 or above");
          error.code = "EUNSUPPORTED";
          throw error;
        }
        return import_http2_wrapper.default.auto;
      }
      return https.request;
    }
    return http.request;
  }
  freeze() {
    const options = this._internals;
    Object.freeze(options);
    Object.freeze(options.hooks);
    Object.freeze(options.hooks.afterResponse);
    Object.freeze(options.hooks.beforeError);
    Object.freeze(options.hooks.beforeRedirect);
    Object.freeze(options.hooks.beforeRequest);
    Object.freeze(options.hooks.beforeRetry);
    Object.freeze(options.hooks.init);
    Object.freeze(options.https);
    Object.freeze(options.cacheOptions);
    Object.freeze(options.agent);
    Object.freeze(options.headers);
    Object.freeze(options.timeout);
    Object.freeze(options.retry);
    Object.freeze(options.retry.errorCodes);
    Object.freeze(options.retry.methods);
    Object.freeze(options.retry.statusCodes);
  }
};

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/response.js
var isResponseOk = (response) => {
  const { statusCode } = response;
  const limitStatusCode = response.request.options.followRedirect ? 299 : 399;
  return statusCode >= 200 && statusCode <= limitStatusCode || statusCode === 304;
};
var ParseError = class extends RequestError {
  constructor(error, response) {
    const { options } = response.request;
    super(`${error.message} in "${options.url.toString()}"`, error, response.request);
    this.name = "ParseError";
    this.code = "ERR_BODY_PARSE_FAILURE";
  }
};
var parseBody = (response, responseType, parseJson, encoding) => {
  const { rawBody } = response;
  try {
    if (responseType === "text") {
      return rawBody.toString(encoding);
    }
    if (responseType === "json") {
      return rawBody.length === 0 ? "" : parseJson(rawBody.toString(encoding));
    }
    if (responseType === "buffer") {
      return rawBody;
    }
  } catch (error) {
    throw new ParseError(error, response);
  }
  throw new ParseError({
    message: `Unknown body type '${responseType}'`,
    name: "Error"
  }, response);
};

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/utils/is-client-request.js
function isClientRequest(clientRequest) {
  return clientRequest.writable && !clientRequest.writableEnded;
}
var is_client_request_default = isClientRequest;

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/utils/is-unix-socket-url.js
function isUnixSocketURL(url) {
  return url.protocol === "unix:" || url.hostname === "unix";
}

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/core/index.js
var supportsBrotli = dist_default.string(process3.versions.brotli);
var methodsWithoutBody = /* @__PURE__ */ new Set(["GET", "HEAD"]);
var cacheableStore = new WeakableMap();
var redirectCodes = /* @__PURE__ */ new Set([300, 301, 302, 303, 304, 307, 308]);
var proxiedRequestEvents = [
  "socket",
  "connect",
  "continue",
  "information",
  "upgrade"
];
var noop2 = () => {
};
var Request = class extends Duplex {
  constructor(url, options, defaults2) {
    super({
      // Don't destroy immediately, as the error may be emitted on unsuccessful retry
      autoDestroy: false,
      // It needs to be zero because we're just proxying the data to another stream
      highWaterMark: 0
    });
    Object.defineProperty(this, "constructor", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_noPipe", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "options", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "response", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "requestUrl", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "redirectUrls", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "retryCount", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_stopRetry", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_downloadedSize", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_uploadedSize", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_stopReading", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_pipedServerResponses", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_request", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_responseSize", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_bodySize", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_unproxyEvents", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_isFromCache", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_cannotHaveBody", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_triggerRead", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_cancelTimeouts", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_removeListeners", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_nativeResponse", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_flushed", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_aborted", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    Object.defineProperty(this, "_requestInitialized", {
      enumerable: true,
      configurable: true,
      writable: true,
      value: void 0
    });
    this._downloadedSize = 0;
    this._uploadedSize = 0;
    this._stopReading = false;
    this._pipedServerResponses = /* @__PURE__ */ new Set();
    this._cannotHaveBody = false;
    this._unproxyEvents = noop2;
    this._triggerRead = false;
    this._cancelTimeouts = noop2;
    this._removeListeners = noop2;
    this._jobs = [];
    this._flushed = false;
    this._requestInitialized = false;
    this._aborted = false;
    this.redirectUrls = [];
    this.retryCount = 0;
    this._stopRetry = noop2;
    this.on("pipe", (source) => {
      if (source.headers) {
        Object.assign(this.options.headers, source.headers);
      }
    });
    this.on("newListener", (event) => {
      if (event === "retry" && this.listenerCount("retry") > 0) {
        throw new Error("A retry listener has been attached already.");
      }
    });
    try {
      this.options = new Options(url, options, defaults2);
      if (!this.options.url) {
        if (this.options.prefixUrl === "") {
          throw new TypeError("Missing `url` property");
        }
        this.options.url = "";
      }
      this.requestUrl = this.options.url;
    } catch (error) {
      const { options: options2 } = error;
      if (options2) {
        this.options = options2;
      }
      this.flush = async () => {
        this.flush = async () => {
        };
        this.destroy(error);
      };
      return;
    }
    const { body } = this.options;
    if (dist_default.nodeStream(body)) {
      body.once("error", (error) => {
        if (this._flushed) {
          this._beforeError(new UploadError(error, this));
        } else {
          this.flush = async () => {
            this.flush = async () => {
            };
            this._beforeError(new UploadError(error, this));
          };
        }
      });
    }
    if (this.options.signal) {
      const abort = () => {
        this.destroy(new AbortError(this));
      };
      if (this.options.signal.aborted) {
        abort();
      } else {
        this.options.signal.addEventListener("abort", abort);
        this._removeListeners = () => {
          this.options.signal.removeEventListener("abort", abort);
        };
      }
    }
  }
  async flush() {
    var _a;
    if (this._flushed) {
      return;
    }
    this._flushed = true;
    try {
      await this._finalizeBody();
      if (this.destroyed) {
        return;
      }
      await this._makeRequest();
      if (this.destroyed) {
        (_a = this._request) == null ? void 0 : _a.destroy();
        return;
      }
      for (const job of this._jobs) {
        job();
      }
      this._jobs.length = 0;
      this._requestInitialized = true;
    } catch (error) {
      this._beforeError(error);
    }
  }
  _beforeError(error) {
    if (this._stopReading) {
      return;
    }
    const { response, options } = this;
    const attemptCount = this.retryCount + (error.name === "RetryError" ? 0 : 1);
    this._stopReading = true;
    if (!(error instanceof RequestError)) {
      error = new RequestError(error.message, error, this);
    }
    const typedError = error;
    void (async () => {
      var _a, _b;
      if ((response == null ? void 0 : response.readable) && !response.rawBody && !((_b = (_a = this._request) == null ? void 0 : _a.socket) == null ? void 0 : _b.destroyed)) {
        response.setEncoding(this.readableEncoding);
        const success = await this._setRawBody(response);
        if (success) {
          response.body = response.rawBody.toString();
        }
      }
      if (this.listenerCount("retry") !== 0) {
        let backoff;
        try {
          let retryAfter;
          if (response && "retry-after" in response.headers) {
            retryAfter = Number(response.headers["retry-after"]);
            if (Number.isNaN(retryAfter)) {
              retryAfter = Date.parse(response.headers["retry-after"]) - Date.now();
              if (retryAfter <= 0) {
                retryAfter = 1;
              }
            } else {
              retryAfter *= 1e3;
            }
          }
          const retryOptions = options.retry;
          backoff = await retryOptions.calculateDelay({
            attemptCount,
            retryOptions,
            error: typedError,
            retryAfter,
            computedValue: calculate_retry_delay_default({
              attemptCount,
              retryOptions,
              error: typedError,
              retryAfter,
              computedValue: retryOptions.maxRetryAfter ?? options.timeout.request ?? Number.POSITIVE_INFINITY
            })
          });
        } catch (error_) {
          void this._error(new RequestError(error_.message, error_, this));
          return;
        }
        if (backoff) {
          await new Promise((resolve) => {
            const timeout = setTimeout(resolve, backoff);
            this._stopRetry = () => {
              clearTimeout(timeout);
              resolve();
            };
          });
          if (this.destroyed) {
            return;
          }
          try {
            for (const hook of this.options.hooks.beforeRetry) {
              await hook(typedError, this.retryCount + 1);
            }
          } catch (error_) {
            void this._error(new RequestError(error_.message, error, this));
            return;
          }
          if (this.destroyed) {
            return;
          }
          this.destroy();
          this.emit("retry", this.retryCount + 1, error, (updatedOptions) => {
            const request = new Request(options.url, updatedOptions, options);
            request.retryCount = this.retryCount + 1;
            process3.nextTick(() => {
              void request.flush();
            });
            return request;
          });
          return;
        }
      }
      void this._error(typedError);
    })();
  }
  _read() {
    this._triggerRead = true;
    const { response } = this;
    if (response && !this._stopReading) {
      if (response.readableLength) {
        this._triggerRead = false;
      }
      let data;
      while ((data = response.read()) !== null) {
        this._downloadedSize += data.length;
        const progress = this.downloadProgress;
        if (progress.percent < 1) {
          this.emit("downloadProgress", progress);
        }
        this.push(data);
      }
    }
  }
  _write(chunk, encoding, callback) {
    const write = () => {
      this._writeRequest(chunk, encoding, callback);
    };
    if (this._requestInitialized) {
      write();
    } else {
      this._jobs.push(write);
    }
  }
  _final(callback) {
    const endRequest = () => {
      if (!this._request || this._request.destroyed) {
        callback();
        return;
      }
      this._request.end((error) => {
        var _a;
        if ((_a = this._request._writableState) == null ? void 0 : _a.errored) {
          return;
        }
        if (!error) {
          this._bodySize = this._uploadedSize;
          this.emit("uploadProgress", this.uploadProgress);
          this._request.emit("upload-complete");
        }
        callback(error);
      });
    };
    if (this._requestInitialized) {
      endRequest();
    } else {
      this._jobs.push(endRequest);
    }
  }
  _destroy(error, callback) {
    this._stopReading = true;
    this.flush = async () => {
    };
    this._stopRetry();
    this._cancelTimeouts();
    this._removeListeners();
    if (this.options) {
      const { body } = this.options;
      if (dist_default.nodeStream(body)) {
        body.destroy();
      }
    }
    if (this._request) {
      this._request.destroy();
    }
    if (error !== null && !dist_default.undefined(error) && !(error instanceof RequestError)) {
      error = new RequestError(error.message, error, this);
    }
    callback(error);
  }
  pipe(destination, options) {
    if (destination instanceof ServerResponse) {
      this._pipedServerResponses.add(destination);
    }
    return super.pipe(destination, options);
  }
  unpipe(destination) {
    if (destination instanceof ServerResponse) {
      this._pipedServerResponses.delete(destination);
    }
    super.unpipe(destination);
    return this;
  }
  async _finalizeBody() {
    const { options } = this;
    const { headers } = options;
    const isForm = !dist_default.undefined(options.form);
    const isJSON = !dist_default.undefined(options.json);
    const isBody = !dist_default.undefined(options.body);
    const cannotHaveBody = methodsWithoutBody.has(options.method) && !(options.method === "GET" && options.allowGetBody);
    this._cannotHaveBody = cannotHaveBody;
    if (isForm || isJSON || isBody) {
      if (cannotHaveBody) {
        throw new TypeError(`The \`${options.method}\` method cannot be used with a body`);
      }
      const noContentType = !dist_default.string(headers["content-type"]);
      if (isBody) {
        if (isFormData(options.body)) {
          const encoder = new FormDataEncoder(options.body);
          if (noContentType) {
            headers["content-type"] = encoder.headers["Content-Type"];
          }
          if ("Content-Length" in encoder.headers) {
            headers["content-length"] = encoder.headers["Content-Length"];
          }
          options.body = encoder.encode();
        }
        if (isFormData2(options.body) && noContentType) {
          headers["content-type"] = `multipart/form-data; boundary=${options.body.getBoundary()}`;
        }
      } else if (isForm) {
        if (noContentType) {
          headers["content-type"] = "application/x-www-form-urlencoded";
        }
        const { form } = options;
        options.form = void 0;
        options.body = new URLSearchParams2(form).toString();
      } else {
        if (noContentType) {
          headers["content-type"] = "application/json";
        }
        const { json } = options;
        options.json = void 0;
        options.body = options.stringifyJson(json);
      }
      const uploadBodySize = await getBodySize(options.body, options.headers);
      if (dist_default.undefined(headers["content-length"]) && dist_default.undefined(headers["transfer-encoding"]) && !cannotHaveBody && !dist_default.undefined(uploadBodySize)) {
        headers["content-length"] = String(uploadBodySize);
      }
    }
    if (options.responseType === "json" && !("accept" in options.headers)) {
      options.headers.accept = "application/json";
    }
    this._bodySize = Number(headers["content-length"]) || void 0;
  }
  async _onResponseBase(response) {
    if (this.isAborted) {
      return;
    }
    const { options } = this;
    const { url } = options;
    this._nativeResponse = response;
    if (options.decompress) {
      response = (0, import_decompress_response.default)(response);
    }
    const statusCode = response.statusCode;
    const typedResponse = response;
    typedResponse.statusMessage = typedResponse.statusMessage ? typedResponse.statusMessage : http2.STATUS_CODES[statusCode];
    typedResponse.url = options.url.toString();
    typedResponse.requestUrl = this.requestUrl;
    typedResponse.redirectUrls = this.redirectUrls;
    typedResponse.request = this;
    typedResponse.isFromCache = this._nativeResponse.fromCache ?? false;
    typedResponse.ip = this.ip;
    typedResponse.retryCount = this.retryCount;
    typedResponse.ok = isResponseOk(typedResponse);
    this._isFromCache = typedResponse.isFromCache;
    this._responseSize = Number(response.headers["content-length"]) || void 0;
    this.response = typedResponse;
    response.once("end", () => {
      this._responseSize = this._downloadedSize;
      this.emit("downloadProgress", this.downloadProgress);
    });
    response.once("error", (error) => {
      this._aborted = true;
      response.destroy();
      this._beforeError(new ReadError(error, this));
    });
    response.once("aborted", () => {
      this._aborted = true;
      this._beforeError(new ReadError({
        name: "Error",
        message: "The server aborted pending request",
        code: "ECONNRESET"
      }, this));
    });
    this.emit("downloadProgress", this.downloadProgress);
    const rawCookies = response.headers["set-cookie"];
    if (dist_default.object(options.cookieJar) && rawCookies) {
      let promises = rawCookies.map(async (rawCookie) => options.cookieJar.setCookie(rawCookie, url.toString()));
      if (options.ignoreInvalidCookies) {
        promises = promises.map(async (promise) => {
          try {
            await promise;
          } catch {
          }
        });
      }
      try {
        await Promise.all(promises);
      } catch (error) {
        this._beforeError(error);
        return;
      }
    }
    if (this.isAborted) {
      return;
    }
    if (options.followRedirect && response.headers.location && redirectCodes.has(statusCode)) {
      response.resume();
      this._cancelTimeouts();
      this._unproxyEvents();
      if (this.redirectUrls.length >= options.maxRedirects) {
        this._beforeError(new MaxRedirectsError(this));
        return;
      }
      this._request = void 0;
      const updatedOptions = new Options(void 0, void 0, this.options);
      const serverRequestedGet = statusCode === 303 && updatedOptions.method !== "GET" && updatedOptions.method !== "HEAD";
      const canRewrite = statusCode !== 307 && statusCode !== 308;
      const userRequestedGet = updatedOptions.methodRewriting && canRewrite;
      if (serverRequestedGet || userRequestedGet) {
        updatedOptions.method = "GET";
        updatedOptions.body = void 0;
        updatedOptions.json = void 0;
        updatedOptions.form = void 0;
        delete updatedOptions.headers["content-length"];
      }
      try {
        const redirectBuffer = Buffer3.from(response.headers.location, "binary").toString();
        const redirectUrl = new URL3(redirectBuffer, url);
        if (!isUnixSocketURL(url) && isUnixSocketURL(redirectUrl)) {
          this._beforeError(new RequestError("Cannot redirect to UNIX socket", {}, this));
          return;
        }
        if (redirectUrl.hostname !== url.hostname || redirectUrl.port !== url.port) {
          if ("host" in updatedOptions.headers) {
            delete updatedOptions.headers.host;
          }
          if ("cookie" in updatedOptions.headers) {
            delete updatedOptions.headers.cookie;
          }
          if ("authorization" in updatedOptions.headers) {
            delete updatedOptions.headers.authorization;
          }
          if (updatedOptions.username || updatedOptions.password) {
            updatedOptions.username = "";
            updatedOptions.password = "";
          }
        } else {
          redirectUrl.username = updatedOptions.username;
          redirectUrl.password = updatedOptions.password;
        }
        this.redirectUrls.push(redirectUrl);
        updatedOptions.prefixUrl = "";
        updatedOptions.url = redirectUrl;
        for (const hook of updatedOptions.hooks.beforeRedirect) {
          await hook(updatedOptions, typedResponse);
        }
        this.emit("redirect", updatedOptions, typedResponse);
        this.options = updatedOptions;
        await this._makeRequest();
      } catch (error) {
        this._beforeError(error);
        return;
      }
      return;
    }
    if (options.isStream && options.throwHttpErrors && !isResponseOk(typedResponse)) {
      this._beforeError(new HTTPError(typedResponse));
      return;
    }
    response.on("readable", () => {
      if (this._triggerRead) {
        this._read();
      }
    });
    this.on("resume", () => {
      response.resume();
    });
    this.on("pause", () => {
      response.pause();
    });
    response.once("end", () => {
      this.push(null);
    });
    if (this._noPipe) {
      const success = await this._setRawBody();
      if (success) {
        this.emit("response", response);
      }
      return;
    }
    this.emit("response", response);
    for (const destination of this._pipedServerResponses) {
      if (destination.headersSent) {
        continue;
      }
      for (const key in response.headers) {
        const isAllowed = options.decompress ? key !== "content-encoding" : true;
        const value = response.headers[key];
        if (isAllowed) {
          destination.setHeader(key, value);
        }
      }
      destination.statusCode = statusCode;
    }
  }
  async _setRawBody(from = this) {
    if (from.readableEnded) {
      return false;
    }
    try {
      const rawBody = await (0, import_get_stream2.buffer)(from);
      if (!this.isAborted) {
        this.response.rawBody = rawBody;
        return true;
      }
    } catch {
    }
    return false;
  }
  async _onResponse(response) {
    try {
      await this._onResponseBase(response);
    } catch (error) {
      this._beforeError(error);
    }
  }
  _onRequest(request) {
    const { options } = this;
    const { timeout, url } = options;
    source_default(request);
    if (this.options.http2) {
      request.setTimeout(0);
    }
    this._cancelTimeouts = timedOut(request, timeout, url);
    const responseEventName = options.cache ? "cacheableResponse" : "response";
    request.once(responseEventName, (response) => {
      void this._onResponse(response);
    });
    request.once("error", (error) => {
      this._aborted = true;
      request.destroy();
      error = error instanceof TimeoutError2 ? new TimeoutError(error, this.timings, this) : new RequestError(error.message, error, this);
      this._beforeError(error);
    });
    this._unproxyEvents = proxyEvents(request, this, proxiedRequestEvents);
    this._request = request;
    this.emit("uploadProgress", this.uploadProgress);
    this._sendBody();
    this.emit("request", request);
  }
  async _asyncWrite(chunk) {
    return new Promise((resolve, reject) => {
      super.write(chunk, (error) => {
        if (error) {
          reject(error);
          return;
        }
        resolve();
      });
    });
  }
  _sendBody() {
    const { body } = this.options;
    const currentRequest = this.redirectUrls.length === 0 ? this : this._request ?? this;
    if (dist_default.nodeStream(body)) {
      body.pipe(currentRequest);
    } else if (dist_default.generator(body) || dist_default.asyncGenerator(body)) {
      (async () => {
        try {
          for await (const chunk of body) {
            await this._asyncWrite(chunk);
          }
          super.end();
        } catch (error) {
          this._beforeError(error);
        }
      })();
    } else if (!dist_default.undefined(body)) {
      this._writeRequest(body, void 0, () => {
      });
      currentRequest.end();
    } else if (this._cannotHaveBody || this._noPipe) {
      currentRequest.end();
    }
  }
  _prepareCache(cache) {
    if (!cacheableStore.has(cache)) {
      const cacheableRequest = new dist_default2((requestOptions, handler) => {
        const result = requestOptions._request(requestOptions, handler);
        if (dist_default.promise(result)) {
          result.once = (event, handler2) => {
            if (event === "error") {
              (async () => {
                try {
                  await result;
                } catch (error) {
                  handler2(error);
                }
              })();
            } else if (event === "abort") {
              (async () => {
                try {
                  const request = await result;
                  request.once("abort", handler2);
                } catch {
                }
              })();
            } else {
              throw new Error(`Unknown HTTP2 promise event: ${event}`);
            }
            return result;
          };
        }
        return result;
      }, cache);
      cacheableStore.set(cache, cacheableRequest.request());
    }
  }
  async _createCacheableRequest(url, options) {
    return new Promise((resolve, reject) => {
      Object.assign(options, urlToOptions(url));
      let request;
      const cacheRequest = cacheableStore.get(options.cache)(options, async (response) => {
        response._readableState.autoDestroy = false;
        if (request) {
          const fix = () => {
            if (response.req) {
              response.complete = response.req.res.complete;
            }
          };
          response.prependOnceListener("end", fix);
          fix();
          (await request).emit("cacheableResponse", response);
        }
        resolve(response);
      });
      cacheRequest.once("error", reject);
      cacheRequest.once("request", async (requestOrPromise) => {
        request = requestOrPromise;
        resolve(request);
      });
    });
  }
  async _makeRequest() {
    const { options } = this;
    const { headers, username, password } = options;
    const cookieJar = options.cookieJar;
    for (const key in headers) {
      if (dist_default.undefined(headers[key])) {
        delete headers[key];
      } else if (dist_default.null_(headers[key])) {
        throw new TypeError(`Use \`undefined\` instead of \`null\` to delete the \`${key}\` header`);
      }
    }
    if (options.decompress && dist_default.undefined(headers["accept-encoding"])) {
      headers["accept-encoding"] = supportsBrotli ? "gzip, deflate, br" : "gzip, deflate";
    }
    if (username || password) {
      const credentials = Buffer3.from(`${username}:${password}`).toString("base64");
      headers.authorization = `Basic ${credentials}`;
    }
    if (cookieJar) {
      const cookieString = await cookieJar.getCookieString(options.url.toString());
      if (dist_default.nonEmptyString(cookieString)) {
        headers.cookie = cookieString;
      }
    }
    options.prefixUrl = "";
    let request;
    for (const hook of options.hooks.beforeRequest) {
      const result = await hook(options);
      if (!dist_default.undefined(result)) {
        request = () => result;
        break;
      }
    }
    if (!request) {
      request = options.getRequestFunction();
    }
    const url = options.url;
    this._requestOptions = options.createNativeRequestOptions();
    if (options.cache) {
      this._requestOptions._request = request;
      this._requestOptions.cache = options.cache;
      this._requestOptions.body = options.body;
      this._prepareCache(options.cache);
    }
    const fn = options.cache ? this._createCacheableRequest : request;
    try {
      let requestOrResponse = fn(url, this._requestOptions);
      if (dist_default.promise(requestOrResponse)) {
        requestOrResponse = await requestOrResponse;
      }
      if (dist_default.undefined(requestOrResponse)) {
        requestOrResponse = options.getFallbackRequestFunction()(url, this._requestOptions);
        if (dist_default.promise(requestOrResponse)) {
          requestOrResponse = await requestOrResponse;
        }
      }
      if (is_client_request_default(requestOrResponse)) {
        this._onRequest(requestOrResponse);
      } else if (this.writable) {
        this.once("finish", () => {
          void this._onResponse(requestOrResponse);
        });
        this._sendBody();
      } else {
        void this._onResponse(requestOrResponse);
      }
    } catch (error) {
      if (error instanceof CacheError2) {
        throw new CacheError(error, this);
      }
      throw error;
    }
  }
  async _error(error) {
    try {
      if (error instanceof HTTPError && !this.options.throwHttpErrors) {
      } else {
        for (const hook of this.options.hooks.beforeError) {
          error = await hook(error);
        }
      }
    } catch (error_) {
      error = new RequestError(error_.message, error_, this);
    }
    this.destroy(error);
  }
  _writeRequest(chunk, encoding, callback) {
    if (!this._request || this._request.destroyed) {
      return;
    }
    this._request.write(chunk, encoding, (error) => {
      if (!error && !this._request.destroyed) {
        this._uploadedSize += Buffer3.byteLength(chunk, encoding);
        const progress = this.uploadProgress;
        if (progress.percent < 1) {
          this.emit("uploadProgress", progress);
        }
      }
      callback(error);
    });
  }
  /**
  The remote IP address.
  */
  get ip() {
    var _a;
    return (_a = this.socket) == null ? void 0 : _a.remoteAddress;
  }
  /**
  Indicates whether the request has been aborted or not.
  */
  get isAborted() {
    return this._aborted;
  }
  get socket() {
    var _a;
    return ((_a = this._request) == null ? void 0 : _a.socket) ?? void 0;
  }
  /**
  Progress event for downloading (receiving a response).
  */
  get downloadProgress() {
    let percent;
    if (this._responseSize) {
      percent = this._downloadedSize / this._responseSize;
    } else if (this._responseSize === this._downloadedSize) {
      percent = 1;
    } else {
      percent = 0;
    }
    return {
      percent,
      transferred: this._downloadedSize,
      total: this._responseSize
    };
  }
  /**
  Progress event for uploading (sending a request).
  */
  get uploadProgress() {
    let percent;
    if (this._bodySize) {
      percent = this._uploadedSize / this._bodySize;
    } else if (this._bodySize === this._uploadedSize) {
      percent = 1;
    } else {
      percent = 0;
    }
    return {
      percent,
      transferred: this._uploadedSize,
      total: this._bodySize
    };
  }
  /**
      The object contains the following properties:
  
      - `start` - Time when the request started.
      - `socket` - Time when a socket was assigned to the request.
      - `lookup` - Time when the DNS lookup finished.
      - `connect` - Time when the socket successfully connected.
      - `secureConnect` - Time when the socket securely connected.
      - `upload` - Time when the request finished uploading.
      - `response` - Time when the request fired `response` event.
      - `end` - Time when the response fired `end` event.
      - `error` - Time when the request fired `error` event.
      - `abort` - Time when the request fired `abort` event.
      - `phases`
          - `wait` - `timings.socket - timings.start`
          - `dns` - `timings.lookup - timings.socket`
          - `tcp` - `timings.connect - timings.lookup`
          - `tls` - `timings.secureConnect - timings.connect`
          - `request` - `timings.upload - (timings.secureConnect || timings.connect)`
          - `firstByte` - `timings.response - timings.upload`
          - `download` - `timings.end - timings.response`
          - `total` - `(timings.end || timings.error || timings.abort) - timings.start`
  
      If something has not been measured yet, it will be `undefined`.
  
      __Note__: The time is a `number` representing the milliseconds elapsed since the UNIX epoch.
      */
  get timings() {
    var _a;
    return (_a = this._request) == null ? void 0 : _a.timings;
  }
  /**
  Whether the response was retrieved from the cache.
  */
  get isFromCache() {
    return this._isFromCache;
  }
  get reusedSocket() {
    var _a;
    return (_a = this._request) == null ? void 0 : _a.reusedSocket;
  }
};

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/as-promise/types.js
var CancelError2 = class extends RequestError {
  constructor(request) {
    super("Promise was canceled", {}, request);
    this.name = "CancelError";
    this.code = "ERR_CANCELED";
  }
  /**
  Whether the promise is canceled.
  */
  get isCanceled() {
    return true;
  }
};

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/as-promise/index.js
var proxiedRequestEvents2 = [
  "request",
  "response",
  "redirect",
  "uploadProgress",
  "downloadProgress"
];
function asPromise(firstRequest) {
  let globalRequest;
  let globalResponse;
  let normalizedOptions;
  const emitter = new EventEmitter2();
  const promise = new PCancelable((resolve, reject, onCancel) => {
    onCancel(() => {
      globalRequest.destroy();
    });
    onCancel.shouldReject = false;
    onCancel(() => {
      reject(new CancelError2(globalRequest));
    });
    const makeRequest = (retryCount) => {
      var _a;
      onCancel(() => {
      });
      const request = firstRequest ?? new Request(void 0, void 0, normalizedOptions);
      request.retryCount = retryCount;
      request._noPipe = true;
      globalRequest = request;
      request.once("response", async (response) => {
        const contentEncoding = (response.headers["content-encoding"] ?? "").toLowerCase();
        const isCompressed = contentEncoding === "gzip" || contentEncoding === "deflate" || contentEncoding === "br";
        const { options } = request;
        if (isCompressed && !options.decompress) {
          response.body = response.rawBody;
        } else {
          try {
            response.body = parseBody(response, options.responseType, options.parseJson, options.encoding);
          } catch (error) {
            response.body = response.rawBody.toString();
            if (isResponseOk(response)) {
              request._beforeError(error);
              return;
            }
          }
        }
        try {
          const hooks = options.hooks.afterResponse;
          for (const [index, hook] of hooks.entries()) {
            response = await hook(response, async (updatedOptions) => {
              options.merge(updatedOptions);
              options.prefixUrl = "";
              if (updatedOptions.url) {
                options.url = updatedOptions.url;
              }
              options.hooks.afterResponse = options.hooks.afterResponse.slice(0, index);
              throw new RetryError(request);
            });
            if (!(dist_default.object(response) && dist_default.number(response.statusCode) && !dist_default.nullOrUndefined(response.body))) {
              throw new TypeError("The `afterResponse` hook returned an invalid value");
            }
          }
        } catch (error) {
          request._beforeError(error);
          return;
        }
        globalResponse = response;
        if (!isResponseOk(response)) {
          request._beforeError(new HTTPError(response));
          return;
        }
        request.destroy();
        resolve(request.options.resolveBodyOnly ? response.body : response);
      });
      const onError = (error) => {
        if (promise.isCanceled) {
          return;
        }
        const { options } = request;
        if (error instanceof HTTPError && !options.throwHttpErrors) {
          const { response } = error;
          request.destroy();
          resolve(request.options.resolveBodyOnly ? response.body : response);
          return;
        }
        reject(error);
      };
      request.once("error", onError);
      const previousBody = (_a = request.options) == null ? void 0 : _a.body;
      request.once("retry", (newRetryCount, error) => {
        firstRequest = void 0;
        const newBody = request.options.body;
        if (previousBody === newBody && dist_default.nodeStream(newBody)) {
          error.message = "Cannot retry with consumed body stream";
          onError(error);
          return;
        }
        normalizedOptions = request.options;
        makeRequest(newRetryCount);
      });
      proxyEvents(request, emitter, proxiedRequestEvents2);
      if (dist_default.undefined(firstRequest)) {
        void request.flush();
      }
    };
    makeRequest(0);
  });
  promise.on = (event, fn) => {
    emitter.on(event, fn);
    return promise;
  };
  promise.off = (event, fn) => {
    emitter.off(event, fn);
    return promise;
  };
  const shortcut = (responseType) => {
    const newPromise = (async () => {
      await promise;
      const { options } = globalResponse.request;
      return parseBody(globalResponse, responseType, options.parseJson, options.encoding);
    })();
    Object.defineProperties(newPromise, Object.getOwnPropertyDescriptors(promise));
    return newPromise;
  };
  promise.json = () => {
    if (globalRequest.options) {
      const { headers } = globalRequest.options;
      if (!globalRequest.writableFinished && !("accept" in headers)) {
        headers.accept = "application/json";
      }
    }
    return shortcut("json");
  };
  promise.buffer = () => shortcut("buffer");
  promise.text = () => shortcut("text");
  return promise;
}

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/create.js
var delay = async (ms) => new Promise((resolve) => {
  setTimeout(resolve, ms);
});
var isGotInstance = (value) => dist_default.function_(value);
var aliases = [
  "get",
  "post",
  "put",
  "patch",
  "head",
  "delete"
];
var create = (defaults2) => {
  defaults2 = {
    options: new Options(void 0, void 0, defaults2.options),
    handlers: [...defaults2.handlers],
    mutableDefaults: defaults2.mutableDefaults
  };
  Object.defineProperty(defaults2, "mutableDefaults", {
    enumerable: true,
    configurable: false,
    writable: false
  });
  const got2 = (url, options, defaultOptions2 = defaults2.options) => {
    const request = new Request(url, options, defaultOptions2);
    let promise;
    const lastHandler = (normalized) => {
      request.options = normalized;
      request._noPipe = !normalized.isStream;
      void request.flush();
      if (normalized.isStream) {
        return request;
      }
      if (!promise) {
        promise = asPromise(request);
      }
      return promise;
    };
    let iteration = 0;
    const iterateHandlers = (newOptions) => {
      const handler = defaults2.handlers[iteration++] ?? lastHandler;
      const result = handler(newOptions, iterateHandlers);
      if (dist_default.promise(result) && !request.options.isStream) {
        if (!promise) {
          promise = asPromise(request);
        }
        if (result !== promise) {
          const descriptors = Object.getOwnPropertyDescriptors(promise);
          for (const key in descriptors) {
            if (key in result) {
              delete descriptors[key];
            }
          }
          Object.defineProperties(result, descriptors);
          result.cancel = promise.cancel;
        }
      }
      return result;
    };
    return iterateHandlers(request.options);
  };
  got2.extend = (...instancesOrOptions) => {
    const options = new Options(void 0, void 0, defaults2.options);
    const handlers = [...defaults2.handlers];
    let mutableDefaults;
    for (const value of instancesOrOptions) {
      if (isGotInstance(value)) {
        options.merge(value.defaults.options);
        handlers.push(...value.defaults.handlers);
        mutableDefaults = value.defaults.mutableDefaults;
      } else {
        options.merge(value);
        if (value.handlers) {
          handlers.push(...value.handlers);
        }
        mutableDefaults = value.mutableDefaults;
      }
    }
    return create({
      options,
      handlers,
      mutableDefaults: Boolean(mutableDefaults)
    });
  };
  const paginateEach = async function* (url, options) {
    let normalizedOptions = new Options(url, options, defaults2.options);
    normalizedOptions.resolveBodyOnly = false;
    const { pagination } = normalizedOptions;
    assert.function_(pagination.transform);
    assert.function_(pagination.shouldContinue);
    assert.function_(pagination.filter);
    assert.function_(pagination.paginate);
    assert.number(pagination.countLimit);
    assert.number(pagination.requestLimit);
    assert.number(pagination.backoff);
    const allItems = [];
    let { countLimit } = pagination;
    let numberOfRequests = 0;
    while (numberOfRequests < pagination.requestLimit) {
      if (numberOfRequests !== 0) {
        await delay(pagination.backoff);
      }
      const response = await got2(void 0, void 0, normalizedOptions);
      const parsed = await pagination.transform(response);
      const currentItems = [];
      assert.array(parsed);
      for (const item of parsed) {
        if (pagination.filter({ item, currentItems, allItems })) {
          if (!pagination.shouldContinue({ item, currentItems, allItems })) {
            return;
          }
          yield item;
          if (pagination.stackAllItems) {
            allItems.push(item);
          }
          currentItems.push(item);
          if (--countLimit <= 0) {
            return;
          }
        }
      }
      const optionsToMerge = pagination.paginate({
        response,
        currentItems,
        allItems
      });
      if (optionsToMerge === false) {
        return;
      }
      if (optionsToMerge === response.request.options) {
        normalizedOptions = response.request.options;
      } else {
        normalizedOptions.merge(optionsToMerge);
        assert.any([dist_default.urlInstance, dist_default.undefined], optionsToMerge.url);
        if (optionsToMerge.url !== void 0) {
          normalizedOptions.prefixUrl = "";
          normalizedOptions.url = optionsToMerge.url;
        }
      }
      numberOfRequests++;
    }
  };
  got2.paginate = paginateEach;
  got2.paginate.all = async (url, options) => {
    const results = [];
    for await (const item of paginateEach(url, options)) {
      results.push(item);
    }
    return results;
  };
  got2.paginate.each = paginateEach;
  got2.stream = (url, options) => got2(url, { ...options, isStream: true });
  for (const method of aliases) {
    got2[method] = (url, options) => got2(url, { ...options, method });
    got2.stream[method] = (url, options) => got2(url, { ...options, method, isStream: true });
  }
  if (!defaults2.mutableDefaults) {
    Object.freeze(defaults2.handlers);
    defaults2.options.freeze();
  }
  Object.defineProperty(got2, "defaults", {
    value: defaults2,
    writable: false,
    configurable: false,
    enumerable: true
  });
  return got2;
};
var create_default = create;

// node_modules/.pnpm/got@12.5.3/node_modules/got/dist/source/index.js
var defaults = {
  options: new Options(),
  handlers: [],
  mutableDefaults: false
};
var got = create_default(defaults);
var source_default2 = got;

// src/index.ts
import pipe from "pump";
import { extract } from "tar-fs";
import gunzipMaybe from "gunzip-maybe";
var agentOpts = {
  keepAlive: true,
  maxSockets: 20
};
var agent = {
  http: new HttpAgent(agentOpts),
  https: new HttpsAgent(agentOpts)
};
async function download({ url, gotOpts, extractOpts, dir }) {
  return new Promise((res, rej) => {
    const callback = (err) => {
      err ? rej(err) : res();
    };
    pipe(
      source_default2.stream(url, Object.assign({ agent }, gotOpts)),
      gunzipMaybe(),
      extract(dir, extractOpts),
      callback
    );
  });
}
export {
  download as default
};
