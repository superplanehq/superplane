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

  export const mock: (typeof import("vitest"))["vi"]["fn"];
  export const spyOn: (typeof import("vitest"))["vi"]["spyOn"];
  export const setSystemTime: (typeof import("vitest"))["vi"]["setSystemTime"];
}
