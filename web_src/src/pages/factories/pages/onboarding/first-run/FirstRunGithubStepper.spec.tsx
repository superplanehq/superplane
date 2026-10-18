import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { FirstRunGithubStepper } from "./FirstRunGithubStepper";

describe("FirstRunGithubStepper", () => {
  it("shows the connect, organization, and repository steps", () => {
    render(<FirstRunGithubStepper current="connect" />);

    expect(screen.getByText("Connect GitHub")).toBeInTheDocument();
    expect(screen.getByText("Choose organization")).toBeInTheDocument();
    expect(screen.getByText("Choose repository")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-step-connect")).toHaveAttribute("data-state", "active");
    expect(screen.getByTestId("first-run-step-organization")).toHaveAttribute("data-state", "upcoming");
    expect(screen.getByTestId("first-run-step-repository")).toHaveAttribute("data-state", "upcoming");
  });

  it("marks connect complete on organization selection", () => {
    render(
      <FirstRunGithubStepper current="organization">
        <p>organization content</p>
      </FirstRunGithubStepper>,
    );

    expect(screen.getByTestId("first-run-step-connect")).toHaveAttribute("data-state", "done");
    expect(screen.getByTestId("first-run-step-organization")).toHaveAttribute("data-state", "active");
    expect(screen.getByTestId("first-run-step-repository")).toHaveAttribute("data-state", "upcoming");
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
    expect(screen.getByTestId("first-run-step-repository")).toHaveAttribute("data-state", "active");
    expect(screen.getByText("Organization: acme")).toBeInTheDocument();
    expect(within(screen.getByTestId("first-run-step-repository")).getByText("repository content")).toBeInTheDocument();
  });
});
