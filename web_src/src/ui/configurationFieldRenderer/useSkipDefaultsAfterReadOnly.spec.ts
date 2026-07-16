import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useSkipDefaultsAfterReadOnly } from "./useSkipDefaultsAfterReadOnly";

describe("useSkipDefaultsAfterReadOnly", () => {
  it("returns false before a read-only session", () => {
    const { result } = renderHook(() => useSkipDefaultsAfterReadOnly(false));
    expect(result.current).toBe(false);
  });

  it("returns true after leaving read-only mode", () => {
    const { result, rerender } = renderHook(({ readOnly }) => useSkipDefaultsAfterReadOnly(readOnly), {
      initialProps: { readOnly: true },
    });

    rerender({ readOnly: false });

    expect(result.current).toBe(true);
  });

  it("resets when the field context key changes", () => {
    const { result, rerender } = renderHook(
      ({ readOnly, contextKey }) => useSkipDefaultsAfterReadOnly(readOnly, contextKey),
      {
        initialProps: { readOnly: true, contextKey: "node-a" },
      },
    );

    rerender({ readOnly: false, contextKey: "node-a" });
    expect(result.current).toBe(true);

    rerender({ readOnly: false, contextKey: "node-b" });
    expect(result.current).toBe(false);
  });
});
