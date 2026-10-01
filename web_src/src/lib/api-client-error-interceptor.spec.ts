import { beforeEach, describe, expect, it, vi } from "bun:test";

const registered = vi.hoisted(() => ({
  use: vi.fn(),
}));

vi.mock("@/api-client/client.gen", () => ({
  client: {
    interceptors: {
      error: {
        use: registered.use,
      },
    },
  },
}));

import { setupApiClientErrorInterceptor } from "./api-client-error-interceptor";

describe("api client error interceptor", () => {
  beforeEach(() => {
    registered.use.mockReset();
  });

  it("registers one interceptor that turns a raw failure into an Error", () => {
    setupApiClientErrorInterceptor();
    setupApiClientErrorInterceptor();

    expect(registered.use).toHaveBeenCalledTimes(1);
    const interceptor = registered.use.mock.calls[0]?.[0] as (
      failure: unknown,
      response: { status: number; statusText: string },
    ) => unknown;

    const fromText = interceptor("Not Found\n", { status: 404, statusText: "Not Found" });
    const fromBody = interceptor({ code: 5, message: "Not found" }, { status: 404, statusText: "Not Found" });
    const fromStatus = interceptor("", { status: 404, statusText: "Not Found" });
    const existing = new Error("already handled");

    expect(fromText).toBeInstanceOf(Error);
    expect((fromText as Error).message).toBe("Not Found");
    expect((fromText as Error & { status?: number }).status).toBe(404);
    expect((fromBody as Error).message).toBe("Not found");
    expect((fromBody as Error & { error?: { code?: number } }).error?.code).toBe(5);
    expect((fromStatus as Error).message).toBe("Not Found");
    expect(interceptor(existing, { status: 500, statusText: "Internal Server Error" })).toBe(existing);
  });
});
