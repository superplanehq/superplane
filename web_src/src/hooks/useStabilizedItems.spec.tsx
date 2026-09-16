import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { useStabilizedItems } from "./useStabilizedItems";

describe("useStabilizedItems", () => {
  it("keeps place across renders when the same ids come back in a new sort", () => {
    const { result, rerender } = renderHook(
      ({ items }: { items: Array<{ id: string }> }) => useStabilizedItems(items, (item) => item.id, "line-1"),
      { initialProps: { items: [{ id: "a" }, { id: "b" }] } },
    );

    rerender({ items: [{ id: "b" }, { id: "a" }] });
    expect(result.current.map((item) => item.id)).toEqual(["a", "b"]);
  });

  it("rebuilds from the incoming list when the reset key changes", () => {
    const { result, rerender } = renderHook(
      ({ items, resetKey }: { items: Array<{ id: string }>; resetKey: string }) =>
        useStabilizedItems(items, (item) => item.id, resetKey),
      { initialProps: { items: [{ id: "a" }, { id: "b" }], resetKey: "line-1" } },
    );

    rerender({ items: [{ id: "b" }, { id: "a" }], resetKey: "line-2" });
    expect(result.current.map((item) => item.id)).toEqual(["b", "a"]);
  });
});
