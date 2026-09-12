import type * as Vitest from "vitest";

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

  export const mock: Vitest.vi["fn"];
  export const spyOn: Vitest.vi["spyOn"];
  export const setSystemTime: Vitest.vi["setSystemTime"];
}
