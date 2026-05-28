import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { CanvasYamlModal } from "./CanvasYamlModal";

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value }: { value?: string }) => <pre data-testid="monaco-stub">{value}</pre>,
}));

describe("CanvasYamlModal", () => {
  it("shows a dismissible dialog even before YAML content is available", () => {
    render(<CanvasYamlModal open onOpenChange={vi.fn()} onCopy={vi.fn()} onDownload={vi.fn()} />);

    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("Canvas YAML is not available yet.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Copy" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Download" })).toBeDisabled();
  });
});
