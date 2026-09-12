import "@testing-library/jest-dom/vitest";

// Some test DOMs omit ResizeObserver; several UI primitives depend on it.
// Provide a no-op so every test file gets it for free instead of having to
// declare it locally.
if (typeof globalThis.ResizeObserver === "undefined") {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
}

// Some test DOMs omit DOMMatrixReadOnly. React Flow reads viewport zoom
// with `new DOMMatrixReadOnly(style.transform)` when node internals update.
if (typeof window.DOMMatrixReadOnly !== "function") {
  class DOMMatrixReadOnlyStub {
    m11 = 1;
    m12 = 0;
    m21 = 0;
    m22 = 1;
    m41 = 0;
    m42 = 0;

    constructor(init?: string) {
      const values = parseCssMatrix2d(init);
      if (!values) {
        return;
      }
      const [a, b, c, d, e, f] = values;
      this.m11 = a;
      this.m12 = b;
      this.m21 = c;
      this.m22 = d;
      this.m41 = e;
      this.m42 = f;
    }
  }

  Object.defineProperty(window, "DOMMatrixReadOnly", {
    configurable: true,
    writable: true,
    value: DOMMatrixReadOnlyStub,
  });
}

function parseCssMatrix2d(init: string | undefined): number[] | undefined {
  if (!init || init === "none") {
    return undefined;
  }
  const match = /matrix\(([^)]+)\)/.exec(init);
  if (!match) {
    return undefined;
  }
  const values = match[1].split(",").map((part) => Number(part.trim()));
  if (values.length < 6 || values.some((value) => Number.isNaN(value))) {
    return undefined;
  }
  return values;
}

// Some test DOMs omit matchMedia; ThemeProvider reads it to resolve
// "system" theme preference. Tests can override this per file when needed.
if (typeof window.matchMedia === "undefined") {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener() {},
      removeListener() {},
      addEventListener() {},
      removeEventListener() {},
      dispatchEvent: () => false,
    }),
  });
}

// happy-dom owns Window.fetch. Tests stub globalThis.fetch; route window.fetch
// through that binding so `vi.stubGlobal("fetch", ...)` applies to both.
Object.defineProperty(window, "fetch", {
  configurable: true,
  writable: true,
  value: ((...args: Parameters<typeof fetch>) => globalThis.fetch(...args)) as typeof fetch,
});

Object.defineProperty(HTMLCanvasElement.prototype, "getContext", {
  configurable: true,
  writable: true,
  value: () => ({
    font: "",
    measureText: (text: string) => ({ width: text.length * 7 }),
  }),
});

// happy-dom follows <a href> clicks as navigations, including download links.
// That replaces location.href (often with a blob: URL) and breaks later
// `new URL("/account", location.href)` fixture matching.
window.addEventListener(
  "click",
  (event) => {
    const target = event.target;
    if (!(target instanceof Element)) {
      return;
    }
    if (target.closest("a[download]")) {
      event.preventDefault();
    }
  },
  true,
);

// happy-dom hardcodes Node.prototype.nodeName to "" and shadows it on Element.
// DOMPurify reads the base getter to resist clobbering, so every tag looks
// empty and the allow-list misfires. Delegate to the instance getter.
patchHappyDomNodeName();

function patchHappyDomNodeName(): void {
  const nodeProto = globalThis.Node?.prototype;
  if (!nodeProto || !document) {
    return;
  }

  const baseDesc = Object.getOwnPropertyDescriptor(nodeProto, "nodeName");
  if (!baseDesc?.get || !baseDesc.configurable) {
    return;
  }

  const probe = document.createElement("div");
  if (baseDesc.get.call(probe) === probe.nodeName) {
    return;
  }

  const baseGet = baseDesc.get;
  Object.defineProperty(nodeProto, "nodeName", {
    configurable: true,
    enumerable: baseDesc.enumerable,
    get(this: Node) {
      let proto: object | null = Object.getPrototypeOf(this);
      while (proto && proto !== nodeProto) {
        const desc = Object.getOwnPropertyDescriptor(proto, "nodeName");
        if (desc?.get) {
          return desc.get.call(this);
        }
        proto = Object.getPrototypeOf(proto);
      }
      return baseGet.call(this);
    },
  });
}
