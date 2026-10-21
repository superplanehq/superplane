import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunGitHubEnterpriseHostScreen } from "./FirstRunGitHubEnterpriseHostScreen";

describe("FirstRunGitHubEnterpriseHostScreen", () => {
  it("asks for the GitHub Enterprise Cloud hostname", async () => {
    const user = userEvent.setup();
    const onContinue = vi.fn();
    render(<FirstRunGitHubEnterpriseHostScreen onContinue={onContinue} />);

    expect(screen.getByTestId("first-run-github-enterprise-host")).toHaveTextContent(
      FIRST_RUN_COPY.githubEnterprise.headline,
    );
    expect(screen.getByTestId("first-run-github-enterprise-continue")).toBeDisabled();

    await user.type(screen.getByTestId("first-run-github-enterprise-hostname"), "acme.ghe.com");
    await user.click(screen.getByTestId("first-run-github-enterprise-continue"));

    expect(onContinue).toHaveBeenCalledWith("acme.ghe.com");
    expect(screen.getByRole("link", { name: FIRST_RUN_COPY.githubEnterprise.ipAllowListLink })).toHaveAttribute(
      "href",
      FIRST_RUN_COPY.githubEnterprise.ipAllowListHref,
    );
  });

  it("shows a connection error so the user can retry", () => {
    render(
      <FirstRunGitHubEnterpriseHostScreen
        connectError={FIRST_RUN_COPY.githubEnterprise.connectError}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByRole("alert")).toHaveTextContent(FIRST_RUN_COPY.githubEnterprise.connectError);
  });
});
