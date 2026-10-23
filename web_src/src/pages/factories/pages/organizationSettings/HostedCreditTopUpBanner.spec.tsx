import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { HostedCreditTopUpBanner } from "./HostedCreditTopUpBanner";

describe("HostedCreditTopUpBanner", () => {
  it("returns keyboard focus to Top up after closing and resets the next purchase", async () => {
    const user = userEvent.setup();
    const onAddCredit = vi.fn();
    render(
      <QueryClientProvider client={new QueryClient()}>
        <HostedCreditTopUpBanner
          organizationId="org-1"
          packs={[{ id: "prod-custom" }]}
          checkoutPending={false}
          onAddCredit={onAddCredit}
        />
      </QueryClientProvider>,
    );
    const topUp = screen.getByRole("button", { name: "Top up" });
    for (const close of ["cancel", "escape"]) {
      topUp.focus();
      await user.keyboard("{Enter}");
      await user.click(screen.getByRole("button", { name: "Custom" }));
      expect(screen.getByRole("textbox", { name: "Credit amount in USD" })).toHaveFocus();
      if (close === "cancel") {
        await user.click(screen.getByRole("button", { name: "Cancel" }));
      } else {
        await user.keyboard("{Escape}");
      }
      await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
      await waitFor(() => expect(topUp).toHaveFocus());
    }
    await user.keyboard("{Enter}");
    expect(screen.queryByRole("textbox", { name: "Credit amount in USD" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Continue to checkout" })).toBeDisabled();
    expect(onAddCredit).not.toHaveBeenCalled();
  });
});
