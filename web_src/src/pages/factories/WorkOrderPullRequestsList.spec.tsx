import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { WorkOrderPullRequestsList } from "./WorkOrderPullRequestsList";

describe("WorkOrderPullRequestsList", () => {
  it("shows a loading skeleton instead of a loading sentence", () => {
    render(<WorkOrderPullRequestsList isLoading pullRequests={[]} />);

    expect(screen.getByRole("status", { name: "Loading pull requests" })).toBeInTheDocument();
    expect(screen.queryByText("Loading pull requests…")).not.toBeInTheDocument();
  });

  it("fades the pull-request list in after a loading stretch", () => {
    const { rerender } = render(<WorkOrderPullRequestsList isLoading pullRequests={[]} />);
    rerender(
      <WorkOrderPullRequestsList
        isLoading={false}
        pullRequests={[
          {
            id: "pr-1",
            number: "12",
            title: "Fix mermaid contrast",
            url: "https://github.com/acme/app/pull/12",
            state: "STATE_OPEN",
          },
        ]}
      />,
    );

    expect(screen.queryByRole("status", { name: "Loading pull requests" })).not.toBeInTheDocument();
    expect(screen.getByRole("list")).toHaveAttribute("data-reveal");
  });
});
