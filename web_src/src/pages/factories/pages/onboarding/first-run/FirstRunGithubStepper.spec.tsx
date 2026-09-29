import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { FirstRunGithubStepper } from "./FirstRunGithubStepper";

describe("FirstRunGithubStepper", () => {
  it("uses the Semaphore-style step names", () => {
    render(<FirstRunGithubStepper current="connect" />);

    expect(screen.getByText("Connect GitHub")).toBeInTheDocument();
    expect(screen.getByText("Grant access")).toBeInTheDocument();
    expect(screen.getByText("Choose repository")).toBeInTheDocument();
  });

  it("marks identity and access complete on repository selection", () => {
    render(
      <FirstRunGithubStepper current="repository">
        <p>repository content</p>
      </FirstRunGithubStepper>,
    );

    expect(screen.getAllByTestId("first-run-step-done")).toHaveLength(2);
    expect(within(screen.getByTestId("first-run-step-repository")).getByText("repository content")).toBeInTheDocument();
  });

  it("renders pending approval status inside the access step", () => {
    render(
      <FirstRunGithubStepper current="grant" grantStatus={<p>Waiting for approval</p>}>
        <p>grant content</p>
      </FirstRunGithubStepper>,
    );

    expect(within(screen.getByTestId("first-run-step-grant")).getByText("Waiting for approval")).toBeInTheDocument();
    expect(within(screen.getByTestId("first-run-step-grant")).getByText("grant content")).toBeInTheDocument();
  });
});
