import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { CLOUD_GITHUB_GLOBEX } from "./firstRunMocks";
import { FirstRunFlow } from "./FirstRunFlow";

describe("FirstRunFlow", () => {
  it("walks welcome, GitHub, account, repository, tickets, then analysis", async () => {
    const user = userEvent.setup();
    render(<FirstRunFlow />);

    await user.click(screen.getByTestId("first-run-get-started"));
    expect(screen.getByTestId("first-run-connect")).toBeInTheDocument();

    await user.click(screen.getByTestId("first-run-connect-github"));
    expect(screen.getByTestId("first-run-github-account-picker")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-github-install-org")).toHaveTextContent(CLOUD_GITHUB_GLOBEX);
    expect(screen.getByText(FIRST_RUN_COPY.connect.installRequested)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: FIRST_RUN_COPY.connect.useAccount(CLOUD_GITHUB_GLOBEX) })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("acme") }));
    expect(screen.getByTestId("first-run-choose")).toBeInTheDocument();

    await user.click(screen.getByRole("option", { name: /acme\/payments-service/ }));
    await user.click(screen.getByTestId("first-run-continue-to-tickets"));

    expect(screen.getByTestId("first-run-tickets")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: FIRST_RUN_COPY.tickets.headline })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /GitHub Issues/ }));
    expect(screen.getByTestId("first-run-tickets")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-analysis")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("first-run-analyze-tickets"));

    expect(screen.getByTestId("first-run-analysis")).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.analysis.reassurance)).toBeInTheDocument();
  });

  it("opens the board when analysis finishes", async () => {
    render(
      <FirstRunFlow initialScreen="analysis" completeAfterMs={20} board={<div data-testid="first-run-board" />} />,
    );

    expect(await screen.findByTestId("first-run-board")).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.results.headline)).not.toBeInTheDocument();
  });
});
