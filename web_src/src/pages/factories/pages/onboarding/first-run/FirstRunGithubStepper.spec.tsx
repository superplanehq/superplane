import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { FirstRunGithubStepper } from "./FirstRunGithubStepper";

describe("FirstRunGithubStepper", () => {
  it("shows only identity and repository selection", () => {
    render(<FirstRunGithubStepper current="connect" />);

    expect(screen.getByText("Connect GitHub")).toBeInTheDocument();
    expect(screen.getByText("Choose repository")).toBeInTheDocument();
    expect(screen.queryByText("Grant access")).not.toBeInTheDocument();
  });

  it("marks identity complete on repository selection", () => {
    render(
      <FirstRunGithubStepper current="repository">
        <p>repository content</p>
      </FirstRunGithubStepper>,
    );

    expect(screen.getAllByTestId("first-run-step-done")).toHaveLength(1);
    expect(within(screen.getByTestId("first-run-step-repository")).getByText("repository content")).toBeInTheDocument();
  });
});
