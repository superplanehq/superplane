"use strict";
var w = Object.defineProperty;
var M = Object.getOwnPropertyDescriptor;
var E = Object.getOwnPropertyNames;
var G = Object.prototype.hasOwnProperty;
var j = (e, t) => {
  for (var n in t)
    w(e, n, { get: t[n], enumerable: !0 });
}, D = (e, t, n, s) => {
  if (t && typeof t == "object" || typeof t == "function")
    for (let r of E(t))
      !G.call(e, r) && r !== n && w(e, r, { get: () => t[r], enumerable: !(s = M(t, r)) || s.enumerable });
  return e;
};
var F = (e) => D(w({}, "__esModule", { value: !0 }), e);

// src/index.ts
var B = {};
j(B, {
  createInternalSpy: () => g,
  getInternalState: () => I,
  internalSpyOn: () => K,
  restoreAll: () => z,
  spies: () => d,
  spy: () => _,
  spyOn: () => $
});
module.exports = F(B);

// src/utils.ts
function R(e, t) {
  if (!e)
    throw new Error(t);
}
function u(e, t) {
  return typeof t === e;
}
function b(e) {
  return e instanceof Promise;
}
function f(e, t, n) {
  Object.defineProperty(e, t, n);
}
function i(e, t, n) {
  Object.defineProperty(e, t, { value: n });
}

// src/constants.ts
var c = Symbol.for("tinyspy:spy");

// src/internal.ts
var d = /* @__PURE__ */ new Set(), q = (e) => {
  e.called = !1, e.callCount = 0, e.calls = [], e.results = [], e.next = [];
}, V = (e) => (f(e, c, { value: { reset: () => q(e[c]) } }), e[c]), I = (e) => e[c] || V(e);
function g(e) {
  R(u("function", e) || u("undefined", e), "cannot spy on a non-function value");
  let t = function(...s) {
    let r = I(t);
    r.called = !0, r.callCount++, r.calls.push(s);
    let m = r.next.shift();
    if (m) {
      r.results.push(m);
      let [l, o] = m;
      if (l === "ok")
        return o;
      throw o;
    }
    let p, x = "ok";
    if (r.impl)
      try {
        new.target ? p = Reflect.construct(r.impl, s, new.target) : p = r.impl.apply(this, s), x = "ok";
      } catch (l) {
        throw p = l, x = "error", r.results.push([x, l]), l;
      }
    let a = [x, p];
    if (b(p)) {
      let l = p.then((o) => a[1] = o).catch((o) => {
        throw a[0] = "error", a[1] = o, o;
      });
      Object.assign(l, p), p = l;
    }
    return r.results.push(a), p;
  };
  i(t, "_isMockFunction", !0), i(t, "length", e ? e.length : 0), i(t, "name", e && e.name || "spy");
  let n = I(t);
  return n.reset(), n.impl = e, t;
}
function A(e) {
  let t = I(e);
  f(e, "returns", {
    get: () => t.results.map(([, n]) => n)
  }), ["called", "callCount", "results", "calls", "reset", "impl"].forEach((n) => f(e, n, { get: () => t[n], set: (s) => t[n] = s })), i(e, "nextError", (n) => (t.next.push(["error", n]), t)), i(e, "nextResult", (n) => (t.next.push(["ok", n]), t));
}

// src/spy.ts
function _(e) {
  let t = g(e);
  return A(t), t;
}

// src/spyOn.ts
var k = (e, t) => Object.getOwnPropertyDescriptor(e, t), P = (e, t) => {
  t != null && typeof t == "function" && t.prototype != null && Object.setPrototypeOf(e.prototype, t.prototype);
};
function K(e, t, n) {
  R(!u("undefined", e), "spyOn could not find an object to spy upon"), R(u("object", e) || u("function", e), "cannot spyOn on a primitive value");
  let [s, r] = (() => {
    if (!u("object", t))
      return [t, "value"];
    if ("getter" in t && "setter" in t)
      throw new Error("cannot spy on both getter and setter");
    if ("getter" in t)
      return [t.getter, "get"];
    if ("setter" in t)
      return [t.setter, "set"];
    throw new Error("specify getter or setter to spy on");
  })(), m = k(e, s), p = Object.getPrototypeOf(e), x = p && k(p, s), a = m || x;
  R(a || s in e, `${String(s)} does not exist`);
  let l = !1;
  r === "value" && a && !a.value && a.get && (r = "get", l = !0, n = a.get());
  let o;
  a ? o = a[r] : r !== "value" ? o = () => e[s] : o = e[s], n || (n = o);
  let y = g(n);
  r === "value" && P(y, o);
  let O = (v) => {
    let { value: H, ...h } = a || {
      configurable: !0,
      writable: !0
    };
    r !== "value" && delete h.writable, h[r] = v, f(e, s, h);
  }, C = () => a ? f(e, s, a) : O(o), T = y[c];
  return i(T, "restore", C), i(T, "getOriginal", () => l ? o() : o), i(T, "willCall", (v) => (T.impl = v, y)), O(l ? () => (P(y, n), y) : y), d.add(y), y;
}
function $(e, t, n) {
  let s = K(e, t, n);
  return A(s), ["restore", "getOriginal", "willCall"].forEach((r) => {
    i(s, r, s[c][r]);
  }), s;
}

// src/restoreAll.ts
function z() {
  for (let e of d)
    e.restore();
  d.clear();
}
// Annotate the CommonJS export names for ESM import in node:
0 && (module.exports = {
  createInternalSpy,
  getInternalState,
  internalSpyOn,
  restoreAll,
  spies,
  spy,
  spyOn
});
