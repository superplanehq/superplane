import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { client } from "@/api-client/client.gen";
import { hostedCreditProductsQueryKey, useHostedCreditProducts } from "@/hooks/useLLMModelAllowlists";

import { HostedCreditTopUpBanner } from "./HostedCreditTopUpBanner";

afterEach(() => client.setConfig({ fetch: undefined }));

describe("HostedCreditTopUpBanner", () => {
  it("refreshes changed products before the buyer confirms another purchase", async () => {
    const user = userEvent.setup();
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    queryClient.setQueryData(hostedCreditProductsQueryKey("org-1"), {
      products: [{ id: "prod-pack", amountCents: "10000" }],
    });
    client.setConfig({
      baseUrl: "http://localhost",
      fetch: async (input) => {
        const request = new Request(input);
        const url = new URL(request.url);
        if (url.pathname.endsWith("/hosted-credit-products")) {
          return Response.json({ products: [{ id: "prod-pack", amountCents: "50000" }] });
        }
        const creditAmountCents = url.searchParams.get("creditAmountCents");
        return Response.json({
          creditAmountCents,
          subtotalCents: creditAmountCents,
          processingFeeCents: "0",
          currency: "usd",
        });
      },
    });
    const onAddCredit = vi
      .fn()
      .mockRejectedValueOnce(new Error("The credit amount changed. Review the updated price."))
      .mockResolvedValueOnce(undefined);
    function PurchaseCredit() {
      const products = useHostedCreditProducts("org-1", true);
      return (
        <HostedCreditTopUpBanner
          organizationId="org-1"
          packs={products.data?.products ?? []}
          checkoutPending={false}
          onAddCredit={onAddCredit}
        />
      );
    }
    render(
      <QueryClientProvider client={queryClient}>
        <PurchaseCredit />
      </QueryClientProvider>,
    );
    await user.click(screen.getByRole("button", { name: "Top up" }));
    await user.click(screen.getByRole("button", { name: "$100" }));
    const confirm = screen.getByRole("button", { name: "Continue to checkout" });
    await waitFor(() => expect(confirm).toBeEnabled());
    await user.click(confirm);
    await waitFor(() => expect(screen.getByRole("button", { name: "$100" })).toBeDisabled());
    expect(confirm).toBeDisabled();
    expect(onAddCredit).toHaveBeenCalledOnce();
    await user.click(screen.getByRole("button", { name: "$500" }));
    await waitFor(() => expect(confirm).toBeEnabled());
    await user.click(confirm);
    expect(onAddCredit).toHaveBeenLastCalledWith({
      productId: "prod-pack",
      creditAmountCents: "50000",
      expectedSubtotalCents: "50000",
    });
  });

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
