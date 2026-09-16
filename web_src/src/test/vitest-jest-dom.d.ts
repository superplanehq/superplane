import type { TestingLibraryMatchers } from "@testing-library/jest-dom/matchers";
import type { ExpectStatic } from "vitest";
import "vitest";

type AsymmetricMatcher = ReturnType<ExpectStatic["stringContaining"]>;

declare module "vitest" {
  interface Matchers<R, _T> extends TestingLibraryMatchers<AsymmetricMatcher, R> {
    toBeInTheDocument(): R;
  }
  interface AsymmetricMatchersContaining extends TestingLibraryMatchers<AsymmetricMatcher, unknown> {
    toBeInTheDocument(): unknown;
  }
}
