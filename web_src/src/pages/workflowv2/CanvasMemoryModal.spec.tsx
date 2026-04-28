import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";
import { CanvasMemoryModal } from "./CanvasMemoryModal";

const noop = () => {};

function renderModal(entries: CanvasMemoryEntry[]) {
  return render(<CanvasMemoryModal open={true} onOpenChange={noop} entries={entries} />);
}

describe("CanvasMemoryModal", () => {
  it("renders http and https string values as links", () => {
    renderModal([
      {
        id: "memory-1",
        namespace: "links",
        values: {
          docs: "https://docs.superplane.com",
          webhook: "http://localhost:8000/hooks",
        },
      },
    ]);

    const docsLink = screen.getByRole("link", { name: "https://docs.superplane.com" });
    expect(docsLink).toHaveAttribute("href", "https://docs.superplane.com");
    expect(docsLink).toHaveAttribute("target", "_blank");

    const webhookLink = screen.getByRole("link", { name: "http://localhost:8000/hooks" });
    expect(webhookLink).toHaveAttribute("href", "http://localhost:8000/hooks");
    expect(webhookLink).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("keeps non-http values as text", () => {
    renderModal([
      {
        id: "memory-2",
        namespace: "links",
        values: {
          email: "mailto:support@example.com",
          label: "release notes",
        },
      },
    ]);

    expect(screen.queryByRole("link", { name: "mailto:support@example.com" })).not.toBeInTheDocument();
    expect(screen.getByText("mailto:support@example.com")).toBeInTheDocument();
    expect(screen.getByText("release notes")).toBeInTheDocument();
  });
});
