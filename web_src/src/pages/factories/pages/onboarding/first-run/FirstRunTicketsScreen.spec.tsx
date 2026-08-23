import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunTicketsScreen } from "./FirstRunTicketsScreen";

describe("FirstRunTicketsScreen", () => {
  it("names the chosen repository and starts from GitHub Issues", async () => {
    const user = userEvent.setup();
    const onSelectTicketSource = vi.fn();

    render(
      <FirstRunTicketsScreen
        repository="acme/payments-service"
        ticketSource={null}
        onSelectTicketSource={onSelectTicketSource}
      />,
    );

    expect(screen.getByText(FIRST_RUN_COPY.tickets.repositoryCaption("acme/payments-service"))).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.tickets.trust)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /GitHub Issues/ }));
    expect(onSelectTicketSource).toHaveBeenCalledWith("github-issues");
  });
});
