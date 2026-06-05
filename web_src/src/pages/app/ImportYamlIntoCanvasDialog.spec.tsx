import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { analytics } from "@/lib/analytics";
import { ImportYamlIntoCanvasDialog } from "./ImportYamlIntoCanvasDialog";

vi.mock("@/lib/analytics", () => ({
  analytics: {
    yamlImport: vi.fn(),
  },
}));

describe("ImportYamlIntoCanvasDialog", () => {
  it("keeps the dialog open and skips analytics when import fails", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();

    render(
      <ImportYamlIntoCanvasDialog
        open
        onOpenChange={onOpenChange}
        onImport={vi.fn().mockRejectedValue(new Error("Save failed"))}
      />,
    );

    fireEvent.change(screen.getByLabelText("YAML definition"), {
      target: {
        value: "apiVersion: v1\nkind: Canvas\nspec:\n  nodes: []\n  edges: []",
      },
    });

    await user.click(screen.getByRole("button", { name: "Import" }));

    expect(await screen.findByText("Save failed")).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
    expect(analytics.yamlImport).not.toHaveBeenCalled();
  });
});
