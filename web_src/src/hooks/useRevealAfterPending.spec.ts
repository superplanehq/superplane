import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { useRevealAfterPending } from "./useRevealAfterPending";

describe("useRevealAfterPending", () => {
  it("does not mark a reveal when the value starts ready", () => {
    const { result } = renderHook(() => useRevealAfterPending(false));
    expect(result.current).toBe(false);
  });

  it("marks a reveal after a pending stretch", () => {
    const { result, rerender } = renderHook(({ pending }) => useRevealAfterPending(pending), {
      initialProps: { pending: true },
    });

    expect(result.current).toBe(false);
    rerender({ pending: false });
    expect(result.current).toBe(true);
  });
});
