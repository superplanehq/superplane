import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CanvasMemoryModal } from "./CanvasMemoryModal";

describe("CanvasMemoryModal", () => {
  it("renders URL values as clickable links", () => {
    render(
      <CanvasMemoryModal
        open
        onOpenChange={() => {}}
        entries={[
          {
            id: "memory-1",
            namespace: "ephemeral-environments",
            values: {
              url: "https://preview.example.com",
              pr_url: "https://github.com/org/repo/pull/49",
            },
          },
        ]}
      />,
    );

    const previewLink = screen.getByRole("link", { name: "https://preview.example.com" });
    expect(previewLink).toHaveAttribute("href", "https://preview.example.com");
    expect(previewLink).toHaveAttribute("target", "_blank");
    expect(previewLink).toHaveAttribute("rel", "noopener noreferrer");

    const prLink = screen.getByRole("link", { name: "https://github.com/org/repo/pull/49" });
    expect(prLink).toHaveAttribute("href", "https://github.com/org/repo/pull/49");
  });

  it("keeps non-URL values as plain text", () => {
    render(
      <CanvasMemoryModal
        open
        onOpenChange={() => {}}
        entries={[
          {
            id: "memory-2",
            namespace: "ephemeral-environments",
            values: {
              status: "ready",
              hostname: "preview.internal",
            },
          },
        ]}
      />,
    );

    expect(screen.getByText("ready")).toBeInTheDocument();
    expect(screen.getByText("preview.internal")).toBeInTheDocument();

    const statusCell = screen.getByText("ready").closest("td");
    expect(statusCell).not.toBeNull();
    expect(within(statusCell as HTMLElement).queryByRole("link")).not.toBeInTheDocument();
  });
});
