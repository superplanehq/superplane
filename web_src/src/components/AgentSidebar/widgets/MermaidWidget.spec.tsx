import { render, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

const { resolvedTheme, mermaidInitialize, mermaidRender } = vi.hoisted(() => ({
  resolvedTheme: { current: "light" as "light" | "dark" },
  mermaidInitialize: vi.fn(),
  mermaidRender: vi.fn(),
}));

vi.mock("mermaid", () => ({
  default: {
    initialize: mermaidInitialize,
    render: mermaidRender,
  },
}));

vi.mock("@/contexts/useTheme", () => ({
  useTheme: () => ({
    preference: resolvedTheme.current,
    resolvedTheme: resolvedTheme.current,
    setPreference: () => undefined,
  }),
}));

import { MermaidWidget } from "./MermaidWidget";

const DIAGRAM = "sequenceDiagram\n    User->>Agent: Stop";

describe("MermaidWidget", () => {
  beforeEach(() => {
    resolvedTheme.current = "light";
    mermaidInitialize.mockReset();
    mermaidRender.mockReset();
    mermaidRender.mockResolvedValue({ svg: "<svg></svg>" });
  });

  it("re-initializes and re-renders mermaid when the resolved theme changes", async () => {
    const { rerender } = render(<MermaidWidget content={DIAGRAM} />);

    await waitFor(() => {
      expect(mermaidInitialize).toHaveBeenCalledTimes(1);
      expect(mermaidRender).toHaveBeenCalledTimes(1);
    });

    expect(mermaidInitialize).toHaveBeenCalledWith(
      expect.objectContaining({
        themeVariables: expect.objectContaining({ textColor: "#1e293b" }),
      }),
    );

    resolvedTheme.current = "dark";
    rerender(<MermaidWidget content={DIAGRAM} />);

    await waitFor(() => {
      expect(mermaidInitialize).toHaveBeenCalledTimes(2);
      expect(mermaidRender).toHaveBeenCalledTimes(2);
    });

    expect(mermaidInitialize).toHaveBeenLastCalledWith(
      expect.objectContaining({
        themeVariables: expect.objectContaining({ textColor: "#e2e8f0" }),
      }),
    );
  });
});
