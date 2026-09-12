import { afterEach, setSystemTime, vi } from "bun:test";
import { GlobalRegistrator } from "@happy-dom/global-registrator";

type ImportMetaEnv = Record<string, string | boolean | undefined>;

const importMetaEnv: ImportMetaEnv = {
  MODE: "test",
  DEV: true,
  PROD: false,
  SSR: false,
  BASE_URL: "/",
  ...process.env,
};

Object.defineProperty(globalThis, "__IMPORT_META_ENV", {
  configurable: true,
  writable: true,
  value: importMetaEnv,
});

if (!GlobalRegistrator.isRegistered) {
  GlobalRegistrator.register();
}

syncWindowUrl("http://localhost/");
patchHistoryLocation();
patchFormMethod();
patchVitestCompat(vi as unknown as Record<string, unknown>);

await import("./setup");

const { cleanup } = await import("@testing-library/react");

afterEach(() => {
  cleanup();
});

function patchVitestCompat(viRecord: Record<string, unknown>) {
  const globalDescriptors = new Map<string, PropertyDescriptor | undefined>();
  const extraRestores: Array<() => void> = [];
  const envPrevious = new Map<string, string | undefined>();
  const advanceTimersByTime = (viRecord.advanceTimersByTime as (ms: number) => void).bind(viRecord);
  const runOnlyPendingTimers = (viRecord.runOnlyPendingTimers as () => void).bind(viRecord);

  Object.defineProperties(viRecord, {
    hoisted: {
      configurable: true,
      enumerable: true,
      value: <T>(factory: () => T) => factory(),
    },
    mocked: {
      configurable: true,
      enumerable: true,
      value: <T>(value: T) => value,
    },
    setSystemTime: {
      configurable: true,
      enumerable: true,
      value: setSystemTime,
    },
    resetModules: {
      configurable: true,
      enumerable: true,
      value: () => undefined,
    },
    advanceTimersByTimeAsync: {
      configurable: true,
      enumerable: true,
      value: async (ms: number) => {
        advanceTimersByTime(ms);
      },
    },
    runOnlyPendingTimersAsync: {
      configurable: true,
      enumerable: true,
      value: async () => {
        runOnlyPendingTimers();
      },
    },
    stubEnv: {
      configurable: true,
      enumerable: true,
      value: (name: string, value: string) => {
        if (!envPrevious.has(name)) {
          envPrevious.set(name, process.env[name]);
        }
        process.env[name] = value;
        importMetaEnv[name] = value;
        return vi;
      },
    },
    unstubAllEnvs: {
      configurable: true,
      enumerable: true,
      value: () => {
        for (const [name, previous] of envPrevious) {
          if (previous === undefined) {
            delete process.env[name];
            delete importMetaEnv[name];
          } else {
            process.env[name] = previous;
            importMetaEnv[name] = previous;
          }
        }
        envPrevious.clear();
        return vi;
      },
    },
    stubGlobal: {
      configurable: true,
      enumerable: true,
      value: (name: string, value: unknown) => {
        if (!globalDescriptors.has(name)) {
          globalDescriptors.set(name, Object.getOwnPropertyDescriptor(globalThis, name));
        }

        Object.defineProperty(globalThis, name, {
          configurable: true,
          enumerable: true,
          writable: true,
          value,
        });

        const locationValue = locationStubFromGlobal(name, value);
        if (locationValue !== undefined) {
          extraRestores.push(replaceWindowLocation(locationValue));
        }

        return vi;
      },
    },
    unstubAllGlobals: {
      configurable: true,
      enumerable: true,
      value: () => {
        for (const restore of extraRestores.splice(0).reverse()) {
          restore();
        }
        for (const [name, descriptor] of globalDescriptors) {
          if (descriptor) {
            Object.defineProperty(globalThis, name, descriptor);
          } else {
            Reflect.deleteProperty(globalThis, name);
          }
        }
        globalDescriptors.clear();
        return vi;
      },
    },
  });
}

function locationStubFromGlobal(name: string, value: unknown): object | undefined {
  if (name === "location" && value && typeof value === "object") {
    return value;
  }
  if (name === "window" && value && typeof value === "object" && "location" in value) {
    const location = (value as { location?: unknown }).location;
    if (location && typeof location === "object") {
      return location;
    }
  }
  return undefined;
}

function replaceWindowLocation(locationValue: object): () => void {
  const previousHref = window.location.href;
  const previousDescriptor = Object.getOwnPropertyDescriptor(window, "location");

  try {
    Object.defineProperty(window, "location", {
      configurable: true,
      enumerable: true,
      writable: true,
      value: locationValue,
    });

    return () => {
      if (previousDescriptor) {
        Object.defineProperty(window, "location", previousDescriptor);
      } else {
        Reflect.deleteProperty(window, "location");
      }
      syncWindowUrl(previousHref);
    };
  } catch {
    const locationObject = window.location;
    const patched: Array<{ key: string; descriptor: PropertyDescriptor | undefined }> = [];

    for (const key of Object.getOwnPropertyNames(locationValue)) {
      const nextDescriptor = Object.getOwnPropertyDescriptor(locationValue, key);
      if (!nextDescriptor) {
        continue;
      }
      patched.push({
        key,
        descriptor: Object.getOwnPropertyDescriptor(locationObject, key),
      });
      try {
        Object.defineProperty(locationObject, key, {
          configurable: true,
          enumerable: true,
          ...nextDescriptor,
        });
      } catch {
        try {
          (locationObject as unknown as Record<string, unknown>)[key] = (locationValue as Record<string, unknown>)[key];
        } catch {
          // Happy DOM locks some Location fields.
        }
      }
    }

    return () => {
      for (const { key, descriptor } of patched) {
        if (descriptor) {
          Object.defineProperty(locationObject, key, descriptor);
        }
      }
      syncWindowUrl(previousHref);
    };
  }
}

function patchHistoryLocation() {
  const history = window.history;
  const pushState = history.pushState.bind(history);
  const replaceState = history.replaceState.bind(history);

  history.pushState = (data, unused, url) => {
    pushState(data, unused, url);
    if (typeof url === "string" || url instanceof URL) {
      syncWindowUrl(url);
    }
  };
  history.replaceState = (data, unused, url) => {
    replaceState(data, unused, url);
    if (typeof url === "string" || url instanceof URL) {
      syncWindowUrl(url);
    }
  };
}

function syncWindowUrl(url: string | URL) {
  const href = resolveWindowUrl(url);
  if (!href) {
    return;
  }

  const happyDOM = (window as Window & { happyDOM?: { setURL?: (next: string) => void } }).happyDOM;
  if (typeof happyDOM?.setURL === "function") {
    happyDOM.setURL(href);
    return;
  }

  try {
    window.location.href = href;
  } catch {
    // Some Happy DOM builds reject href assignment.
  }
}

function resolveWindowUrl(url: string | URL): string | undefined {
  const href = url.toString();
  const currentHref = window.location.href;
  const bases =
    currentHref.startsWith("http://") || currentHref.startsWith("https://")
      ? [currentHref, "http://localhost/"]
      : ["http://localhost/", currentHref];

  for (const base of bases) {
    try {
      return new URL(href, base).href;
    } catch {
      // Happy DOM starts at about:blank, which cannot resolve a path-only URL.
    }
  }

  return undefined;
}

function patchFormMethod() {
  const descriptor = Object.getOwnPropertyDescriptor(HTMLFormElement.prototype, "method");
  if (descriptor?.get && descriptor.set) {
    return;
  }

  const methods = new WeakMap<HTMLFormElement, string>();
  Object.defineProperty(HTMLFormElement.prototype, "method", {
    configurable: true,
    enumerable: true,
    get() {
      return methods.get(this) ?? this.getAttribute("method") ?? "get";
    },
    set(value: string) {
      methods.set(this, value);
      this.setAttribute("method", value);
    },
  });
}
