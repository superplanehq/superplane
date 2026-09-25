import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunGithubStepper } from "./FirstRunGithubStepper";

const copy = FIRST_RUN_COPY.connect;

describe("FirstRunGithubStepper", () => {
  it("shows the connect step active with the upcoming steps dim", () => {
    render(
      <FirstRunGithubStepper current="connect" action={<button type="button">Connect</button>}>
        <p>connect content</p>
      </FirstRunGithubStepper>,
    );

    expect(screen.getByText(copy.connectGitHub)).toBeInTheDocument();
    expect(screen.getByText(copy.stepOrganization)).toBeInTheDocument();
    expect(screen.getByText(copy.stepRepository)).toBeInTheDocument();
    expect(screen.getByText("connect content")).toBeInTheDocument();
    expect(screen.queryAllByTestId("first-run-step-done")).toHaveLength(0);
  });

  it("collapses finished steps to checkmarked rows on the repository step", () => {
    render(
      <FirstRunGithubStepper current="repository" organizationName="puppies-inc">
        <p>repository content</p>
      </FirstRunGithubStepper>,
    );

    expect(screen.getByText(copy.stepConnected)).toBeInTheDocument();
    expect(screen.getByText(copy.stepOrganizationDone("puppies-inc"))).toBeInTheDocument();
    expect(screen.getAllByTestId("first-run-step-done")).toHaveLength(2);
    expect(within(screen.getByTestId("first-run-step-repository")).getByText("repository content")).toBeInTheDocument();
  });

  it("keeps the organization label plain when the name is unknown", () => {
    render(<FirstRunGithubStepper current="repository" />);

    expect(screen.getByText(copy.stepOrganization)).toBeInTheDocument();
  });

  it("renders content only on the active step", () => {
    render(
      <FirstRunGithubStepper current="organization">
        <p>organization content</p>
      </FirstRunGithubStepper>,
    );

    expect(screen.getByText(copy.stepConnected)).toBeInTheDocument();
    expect(
      within(screen.getByTestId("first-run-step-organization")).getByText("organization content"),
    ).toBeInTheDocument();
    expect(within(screen.getByTestId("first-run-step-repository")).queryByText("organization content")).toBeNull();
  });
});
