import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { FactoryCanvasYamlModal } from "./FactoryCanvasYamlModal";

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value }: { value?: string }) => <pre data-testid="monaco-editor">{value}</pre>,
}));

vi.mock("@/contexts/useTheme", () => ({
  useTheme: () => ({ preference: "light", resolvedTheme: "light", setPreference: () => undefined }),
}));

describe("FactoryCanvasYamlModal", () => {
  const canvas = {
    metadata: { id: "app-1", name: "Refund Implementer" },
    spec: { nodes: [], edges: [] },
  };

  it("shows read-only canvas YAML", () => {
    render(<FactoryCanvasYamlModal open onOpenChange={vi.fn()} canvas={canvas} />);

    expect(screen.getByTestId("factory-canvas-yaml-modal")).toBeInTheDocument();
    expect(screen.getByText("View YAML")).toBeInTheDocument();
    expect(screen.getByTestId("monaco-editor")).toHaveTextContent("name: Refund Implementer");
  });

  it("copies the YAML", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });

    render(<FactoryCanvasYamlModal open onOpenChange={vi.fn()} canvas={canvas} />);
    fireEvent.click(screen.getByTestId("factory-canvas-yaml-copy"));

    expect(writeText).toHaveBeenCalled();
    expect(writeText.mock.calls[0]?.[0]).toContain("name: Refund Implementer");
  });
});
