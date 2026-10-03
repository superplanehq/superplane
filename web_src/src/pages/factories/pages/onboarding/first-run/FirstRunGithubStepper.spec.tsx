import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { FirstRunGithubStepper } from "./FirstRunGithubStepper";

describe("FirstRunGithubStepper", () => {
  it("shows the connect, organization, and repository steps", () => {
    render(<FirstRunGithubStepper current="connect" />);

    expect(screen.getByText("Connect GitHub")).toBeInTheDocument();
    expect(screen.getByText("Choose organization")).toBeInTheDocument();
    expect(screen.getByText("Choose repository")).toBeInTheDocument();
    expect(screen.queryByText("Grant access")).not.toBeInTheDocument();
  });

  it("marks connect complete on organization selection", () => {
    render(
      <FirstRunGithubStepper current="organization">
        <p>organization content</p>
      </FirstRunGithubStepper>,
    );

    expect(screen.getAllByTestId("first-run-step-done")).toHaveLength(1);
    expect(
      within(screen.getByTestId("first-run-step-organization")).getByText("organization content"),
    ).toBeInTheDocument();
  });

  it("names the chosen organization on repository selection", () => {
    render(
      <FirstRunGithubStepper current="repository" organizationName="acme">
        <p>repository content</p>
      </FirstRunGithubStepper>,
    );

    expect(screen.getAllByTestId("first-run-step-done")).toHaveLength(2);
    expect(screen.getByText("Organization: acme")).toBeInTheDocument();
    expect(within(screen.getByTestId("first-run-step-repository")).getByText("repository content")).toBeInTheDocument();
  });
});
