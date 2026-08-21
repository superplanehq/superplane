import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { MemorySourceControls, RunSourceControls } from "./MarkdownVariableSourceControls";

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvasMemoryEntries: () => ({
    data: [
      { id: "1", namespace: "orders", values: {}, source: "node" as const },
      { id: "2", namespace: "orders", values: {}, source: "node" as const },
    ],
    isLoading: false,
  }),
}));

describe("RunSourceControls", () => {
  it("drops status selections that cannot match the selected run bucket", () => {
    const onChange = vi.fn();
    render(<RunSourceControls source={{ kind: "run", select: "latest_passed" }} onChange={onChange} />);

    fireEvent.click(screen.getByRole("button", { name: "Toggle run filters" }));
    fireEvent.click(screen.getByText("Failed").closest("label")!);

    expect(onChange).toHaveBeenCalledWith({ kind: "run", select: "latest_passed" });
  });
});

describe("MemorySourceControls", () => {
  // Radix's Select emits its controlled/uncontrolled mismatch warning via
  // `console.warn` (see @radix-ui/react-use-controllable-state), not
  // `console.error`, so that's what regression tests for this bug need to spy on.
  let consoleWarnSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
  });

  afterEach(() => {
    consoleWarnSpy.mockRestore();
  });

  it("keeps the namespace Select controlled when going from an empty to a non-empty namespace", () => {
    const onChange = vi.fn();
    const { rerender } = render(
      <MemorySourceControls canvasId="canvas-1" source={{ kind: "memory", namespace: "" }} onChange={onChange} />,
    );

    rerender(
      <MemorySourceControls canvasId="canvas-1" source={{ kind: "memory", namespace: "orders" }} onChange={onChange} />,
    );

    const warnings = consoleWarnSpy.mock.calls
      .map((args: unknown[]) => args.join(" "))
      .filter((message: string) => message.includes("controlled"));
    expect(warnings).toEqual([]);
  });
});
