type ImportMetaWithRequire = ImportMeta & {
  require?: (id: string) => unknown;
};

function bunRequire(id: string): unknown {
  const requireFn = (import.meta as ImportMetaWithRequire).require;
  if (typeof requireFn !== "function") {
    throw new Error(`Cannot load unmocked module ${id}`);
  }
  return requireFn(id);
}

/**
 * Load a `src/` module through a specifier that is not the mocked alias.
 * Bun deadlocks when a mock factory import()s the same id it mocks.
 */
export function unmockedSrc<T>(pathFromSrc: string): T {
  return bunRequire(`../${pathFromSrc}`) as T;
}

/**
 * Load a package file through node_modules so the mock of the package
 * name does not intercept the load. `id` is the path under node_modules.
 */
export function unmockedPackage<T>(id: string): T {
  return bunRequire(`../../node_modules/${id}`) as T;
}
