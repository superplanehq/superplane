"use strict";
Object.defineProperty(exports, Symbol.toStringTag, { value: "Module" });
const React = require("react");
const reactDom = require("react-dom");
const virtualCore = require("@tanstack/virtual-core");
function _interopNamespaceDefault(e) {
  const n = Object.create(null, { [Symbol.toStringTag]: { value: "Module" } });
  if (e) {
    for (const k in e) {
      if (k !== "default") {
        const d = Object.getOwnPropertyDescriptor(e, k);
        Object.defineProperty(n, k, d.get ? d : {
          enumerable: true,
          get: () => e[k]
        });
      }
    }
  }
  n.default = e;
  return Object.freeze(n);
}
const React__namespace = /* @__PURE__ */ _interopNamespaceDefault(React);
const useIsomorphicLayoutEffect = typeof document !== "undefined" ? React__namespace.useLayoutEffect : React__namespace.useEffect;
function useVirtualizerBase(options) {
  const rerender = React__namespace.useReducer(() => ({}), {})[1];
  const resolvedOptions = {
    ...options,
    onChange: (instance2, sync) => {
      var _a;
      if (sync) {
        reactDom.flushSync(rerender);
      } else {
        rerender();
      }
      (_a = options.onChange) == null ? void 0 : _a.call(options, instance2, sync);
    }
  };
  const [instance] = React__namespace.useState(
    () => new virtualCore.Virtualizer(resolvedOptions)
  );
  instance.setOptions(resolvedOptions);
  useIsomorphicLayoutEffect(() => {
    return instance._didMount();
  }, []);
  useIsomorphicLayoutEffect(() => {
    return instance._willUpdate();
  });
  return instance;
}
function useVirtualizer(options) {
  return useVirtualizerBase({
    observeElementRect: virtualCore.observeElementRect,
    observeElementOffset: virtualCore.observeElementOffset,
    scrollToFn: virtualCore.elementScroll,
    ...options
  });
}
function useWindowVirtualizer(options) {
  return useVirtualizerBase({
    getScrollElement: () => typeof document !== "undefined" ? window : null,
    observeElementRect: virtualCore.observeWindowRect,
    observeElementOffset: virtualCore.observeWindowOffset,
    scrollToFn: virtualCore.windowScroll,
    initialOffset: () => typeof document !== "undefined" ? window.scrollY : 0,
    ...options
  });
}
exports.useVirtualizer = useVirtualizer;
exports.useWindowVirtualizer = useWindowVirtualizer;
Object.keys(virtualCore).forEach((k) => {
  if (k !== "default" && !Object.prototype.hasOwnProperty.call(exports, k)) Object.defineProperty(exports, k, {
    enumerable: true,
    get: () => virtualCore[k]
  });
});
//# sourceMappingURL=index.cjs.map
