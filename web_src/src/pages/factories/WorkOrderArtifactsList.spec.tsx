import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { WorkOrderArtifactsList } from "./WorkOrderArtifactsList";

describe("WorkOrderArtifactsList", () => {
  it("left-aligns linked and markdown artifacts to the same row width", () => {
    render(
      <WorkOrderArtifactsList
        isLoading={false}
        artifacts={[
          {
            id: "pr",
            type: "TYPE_PR",
            data: { title: "#42", url: "https://example.com/pull/42" },
          },
          {
            id: "note",
            type: "TYPE_MARKDOWN",
            data: { title: "Release note", body: "Ready" },
          },
        ]}
      />,
    );

    const pullRequest = screen.getByRole("link");
    const note = screen.getByRole("button", { name: "Release note" });

    expect(pullRequest).toHaveClass("inline-flex", "w-full", "justify-start");
    expect(note).toHaveClass("inline-flex", "w-full", "justify-start", "p-0");
    expect(note).not.toHaveAttribute("data-slot", "button");

    fireEvent.click(note);
    expect(screen.getByRole("dialog").parentElement).toBe(document.body);
  });
});
