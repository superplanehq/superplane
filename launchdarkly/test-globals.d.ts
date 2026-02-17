declare const describe: (name: string, fn: () => void) => void;
declare const it: (name: string, fn: () => void | Promise<void>) => void;
declare const beforeEach: (fn: () => void | Promise<void>) => void;
declare const afterEach: (fn: () => void | Promise<void>) => void;

declare const expect: {
  (value: unknown): {
    toBe: (expected: unknown) => void;
    toEqual: (expected: unknown) => void;
    toContain: (expected: string) => void;
    toBeDefined: () => void;
    toBeUndefined: () => void;
    toHaveBeenCalledWith: (...args: unknown[]) => void;
    toThrow: (expected?: unknown) => void;
    rejects: {
      toThrow: (expected?: unknown) => Promise<void>;
    };
  };
  stringContaining: (value: string) => unknown;
  any: (ctor: unknown) => unknown;
  objectContaining: (value: Record<string, unknown>) => unknown;
};

declare const jest: {
  fn: () => unknown;
  clearAllMocks: () => void;
  Mock: unknown;
};

declare const global: {
  fetch: unknown;
};
