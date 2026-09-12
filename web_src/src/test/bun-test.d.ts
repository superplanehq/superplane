// Keep this file a script (no top-level import/export). A top-level import
// turns `declare module` into an augmentation of a missing package, and
// `tsc -b` then reports TS2307 for every `from "bun:test"` spec.
declare module "bun:test" {
  export {
    afterAll,
    afterEach,
    beforeAll,
    beforeEach,
    describe,
    expect,
    it,
    test,
    vi,
    type Mock,
    type MockedFunction,
    type MockInstance,
  } from "vitest";

  type VitestVi = (typeof import("vitest"))["vi"];

  export const mock: VitestVi["fn"];
  export const spyOn: VitestVi["spyOn"];
  export const setSystemTime: VitestVi["setSystemTime"];
}
