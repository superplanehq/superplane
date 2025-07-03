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
var g = /* @__PURE__ */ new Set(), C = (e) => {
  e.called = !1, e.callCount = 0, e.calls = [], e.results = [], e.next = [];
}, M = (e) => (f(e, c, { value: { reset: () => C(e[c]) } }), e[c]), A = (e) => e[c] || M(e);
function I(e) {
  R(u("function", e) || u("undefined", e), "cannot spy on a non-function value");
  let t = function(...s) {
    let r = A(t);
    r.called = !0, r.callCount++, r.calls.push(s);
    let m = r.next.shift();
    if (m) {
      r.results.push(m);
      let [l, o] = m;
      if (l === "ok")
        return o;
      throw o;
    }
    let p, d = "ok";
    if (r.impl)
      try {
        new.target ? p = Reflect.construct(r.impl, s, new.target) : p = r.impl.apply(this, s), d = "ok";
      } catch (l) {
        throw p = l, d = "error", r.results.push([d, l]), l;
      }
    let a = [d, p];
    if (b(p)) {
      let l = p.then((o) => a[1] = o).catch((o) => {
        throw a[0] = "error", a[1] = o, o;
      });
      Object.assign(l, p), p = l;
    }
    return r.results.push(a), p;
  };
  i(t, "_isMockFunction", !0), i(t, "length", e ? e.length : 0), i(t, "name", e && e.name || "spy");
  let n = A(t);
  return n.reset(), n.impl = e, t;
}
function v(e) {
  let t = A(e);
  f(e, "returns", {
    get: () => t.results.map(([, n]) => n)
  }), ["called", "callCount", "results", "calls", "reset", "impl"].forEach((n) => f(e, n, { get: () => t[n], set: (s) => t[n] = s })), i(e, "nextError", (n) => (t.next.push(["error", n]), t)), i(e, "nextResult", (n) => (t.next.push(["ok", n]), t));
}

// src/spy.ts
function z(e) {
  let t = I(e);
  return v(t), t;
}

// src/spyOn.ts
var k = (e, t) => Object.getOwnPropertyDescriptor(e, t), P = (e, t) => {
  t != null && typeof t == "function" && t.prototype != null && Object.setPrototypeOf(e.prototype, t.prototype);
};
function E(e, t, n) {
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
  })(), m = k(e, s), p = Object.getPrototypeOf(e), d = p && k(p, s), a = m || d;
  R(a || s in e, `${String(s)} does not exist`);
  let l = !1;
  r === "value" && a && !a.value && a.get && (r = "get", l = !0, n = a.get());
  let o;
  a ? o = a[r] : r !== "value" ? o = () => e[s] : o = e[s], n || (n = o);
  let y = I(n);
  r === "value" && P(y, o);
  let O = (h) => {
    let { value: G, ...w } = a || {
      configurable: !0,
      writable: !0
    };
    r !== "value" && delete w.writable, w[r] = h, f(e, s, w);
  }, K = () => a ? f(e, s, a) : O(o), T = y[c];
  return i(T, "restore", K), i(T, "getOriginal", () => l ? o() : o), i(T, "willCall", (h) => (T.impl = h, y)), O(l ? () => (P(y, n), y) : y), g.add(y), y;
}
function W(e, t, n) {
  let s = E(e, t, n);
  return v(s), ["restore", "getOriginal", "willCall"].forEach((r) => {
    i(s, r, s[c][r]);
  }), s;
}

// src/restoreAll.ts
function Z() {
  for (let e of g)
    e.restore();
  g.clear();
}
export {
  I as createInternalSpy,
  A as getInternalState,
  E as internalSpyOn,
  Z as restoreAll,
  g as spies,
  z as spy,
  W as spyOn
};
