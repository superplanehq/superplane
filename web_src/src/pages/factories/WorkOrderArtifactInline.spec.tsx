import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { WorkOrderArtifactInline } from "./WorkOrderArtifactInline";

describe("WorkOrderArtifactInline", () => {
  it("renders a branch artifact with a url as a clickable link", () => {
    render(
      <WorkOrderArtifactInline
        artifact={{
          id: "branch-with-url",
          type: "TYPE_BRANCH",
          data: {
            name: "feature/refund-retry",
            url: "https://github.com/example/repo/tree/feature/refund-retry",
          },
        }}
      />,
    );

    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "https://github.com/example/repo/tree/feature/refund-retry");
    expect(link).toHaveTextContent("feature/refund-retry");
  });

  it("renders a branch artifact without a url as plain text, not a link", () => {
    render(
      <WorkOrderArtifactInline
        artifact={{
          id: "branch-without-url",
          type: "TYPE_BRANCH",
          data: { name: "feature/refund-retry" },
        }}
      />,
    );

    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    expect(screen.getByText("feature/refund-retry")).toBeInTheDocument();
  });
});
