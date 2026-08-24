import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunTicketsScreen } from "./FirstRunTicketsScreen";

describe("FirstRunTicketsScreen", () => {
  it("keeps analysis stopped until a ticket system is selected", async () => {
    const user = userEvent.setup();
    const onSelectTicketSource = vi.fn();
    const onAnalyzeTickets = vi.fn();

    render(
      <FirstRunTicketsScreen
        ticketSource={null}
        onSelectTicketSource={onSelectTicketSource}
        onAnalyzeTickets={onAnalyzeTickets}
      />,
    );

    expect(screen.getByRole("heading", { name: FIRST_RUN_COPY.tickets.headline })).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.tickets.trust)).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.tickets.scoreHint)).toBeInTheDocument();
    expect(screen.queryByText(/The analysis starts when you choose/)).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-analyze-tickets")).toBeDisabled();

    await user.click(screen.getByRole("button", { name: /GitHub Issues/ }));
    expect(onSelectTicketSource).toHaveBeenCalledWith("github-issues");
    expect(onAnalyzeTickets).not.toHaveBeenCalled();
  });

  it("starts analysis from the button after a ticket system is selected", async () => {
    const user = userEvent.setup();
    const onAnalyzeTickets = vi.fn();

    render(
      <FirstRunTicketsScreen
        ticketSource="github-issues"
        onSelectTicketSource={vi.fn()}
        onAnalyzeTickets={onAnalyzeTickets}
      />,
    );

    const analyze = screen.getByRole("button", { name: FIRST_RUN_COPY.tickets.analyze });
    expect(analyze).toBeEnabled();
    await user.click(analyze);
    expect(onAnalyzeTickets).toHaveBeenCalledTimes(1);
  });

  it("uses the next-step label when setup names the coding agent step", () => {
    render(
      <FirstRunTicketsScreen
        ticketSource="github-issues"
        continueLabel={FIRST_RUN_COPY.tickets.continue}
        onSelectTicketSource={vi.fn()}
        onAnalyzeTickets={vi.fn()}
      />,
    );

    expect(screen.getByRole("button", { name: FIRST_RUN_COPY.tickets.continue })).toBeEnabled();
  });
});
