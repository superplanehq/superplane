import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { client } from "@/api-client/client.gen";

import { HostedCreditCheckoutDialog } from "./HostedCreditCheckoutDialog";

const packs = [
  { id: "prod-50", amountCents: "5000" },
  { id: "prod-100", amountCents: "10000" },
  { id: "prod-500", amountCents: "50000" },
  { id: "prod-custom" },
];

function renderDialog(onAddCredit = vi.fn().mockResolvedValue(undefined), nextPacks: typeof packs = packs) {
  const onClose = vi.fn();
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <HostedCreditCheckoutDialog
        organizationId="org-1"
        packs={nextPacks}
        checkoutPending={false}
        onAddCredit={onAddCredit}
        onClose={onClose}
      />
    </QueryClientProvider>,
  );
  return { onAddCredit, onClose, user: userEvent.setup() };
}

afterEach(() => {
  client.setConfig({ fetch: undefined });
});

describe("HostedCreditCheckoutDialog", () => {
  it("requires an amount and confirmation before starting checkout", async () => {
    client.setConfig({
      baseUrl: "http://localhost",
      fetch: vi.fn().mockResolvedValue(
        Response.json({
          creditAmountCents: "10000",
          processingFeeCents: "579",
          fixedFeeCents: "50",
          percentageFeeBps: 500,
          percentageFeeCents: "529",
          subtotalCents: "10579",
          currency: "usd",
        }),
      ),
    });
    const { user, onAddCredit, onClose } = renderDialog();
    const dialog = screen.getByRole("dialog", { name: "Add hosted credit" });
    const confirm = within(dialog).getByRole("button", { name: "Continue to checkout" });
    expect(confirm).toBeDisabled();

    await user.click(within(dialog).getByRole("button", { name: "$100" }));
    await waitFor(() => expect(confirm).toBeEnabled());
    expect(dialog).toHaveTextContent("$100.00");
    expect(dialog).toHaveTextContent("$5.79");
    expect(within(dialog).getByText("Fixed fee per transaction").parentElement).toHaveTextContent("$0.50");
    expect(within(dialog).getByText("Percentage fee (5% of subtotal)").parentElement).toHaveTextContent("$5.29");
    expect(dialog).toHaveTextContent("$105.79");
    expect(dialog).toHaveTextContent("Tax is calculated at checkout.");
    expect(onAddCredit).not.toHaveBeenCalled();

    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
    expect(onClose).toHaveBeenCalledOnce();
    expect(onAddCredit).not.toHaveBeenCalled();
    await user.click(confirm);
    expect(onAddCredit).toHaveBeenCalledWith({
      productId: "prod-100",
      creditAmountCents: "10000",
      expectedSubtotalCents: "10579",
    });
  });

  it("accepts custom cents and hides the previous quote when the amount changes", async () => {
    let resolveQuote: ((response: Response) => void) | undefined;
    const fetchQuote = vi.fn().mockImplementation(async (request: Request) => {
      const amount = new URL(request.url).searchParams.get("creditAmountCents");
      if (amount === "1234") {
        return Response.json({
          creditAmountCents: "1234",
          processingFeeCents: "118",
          fixedFeeCents: "50",
          percentageFeeBps: 500,
          percentageFeeCents: "68",
          subtotalCents: "1352",
          currency: "usd",
        });
      }
      if (amount === "2500")
        return new Promise<Response>((resolve) => {
          resolveQuote = resolve;
        });
      return Response.json({
        creditAmountCents: amount,
        processingFeeCents: "50",
        fixedFeeCents: "50",
        percentageFeeBps: 0,
        percentageFeeCents: "0",
        subtotalCents: String(Number(amount) + 50),
        currency: "usd",
      });
    });
    client.setConfig({ baseUrl: "http://localhost", fetch: fetchQuote });
    const { user, onAddCredit } = renderDialog();
    await user.click(screen.getByRole("button", { name: "Custom" }));
    const amount = () => screen.getByRole("textbox", { name: "Credit amount in USD" });
    const confirm = () => screen.getByRole("button", { name: "Continue to checkout" });

    for (const invalid of ["0", "-1", "1.234", "1e2", "9007199254740992"]) {
      fireEvent.change(amount(), { target: { value: invalid } });
      expect(confirm()).toBeDisabled();
    }
    expect(fetchQuote).not.toHaveBeenCalled();
    await user.clear(amount());
    await user.paste("12.34");
    await waitFor(() => expect(confirm()).toBeEnabled());
    expect(screen.getByText("$13.52")).toBeInTheDocument();

    await user.clear(amount());
    await user.paste("25");
    expect(confirm()).toBeDisabled();
    expect(screen.queryByText("$13.52")).not.toBeInTheDocument();
    await waitFor(() => expect(resolveQuote).toBeDefined());
    await act(async () => {
      resolveQuote?.(
        Response.json({
          creditAmountCents: "2500",
          processingFeeCents: "185",
          fixedFeeCents: "50",
          percentageFeeBps: 500,
          percentageFeeCents: "135",
          subtotalCents: "2685",
          currency: "usd",
        }),
      );
    });
    await waitFor(() => expect(confirm()).toBeEnabled());
    await user.click(confirm());
    expect(onAddCredit).toHaveBeenCalledWith({
      productId: "prod-custom",
      creditAmountCents: "2500",
      expectedSubtotalCents: "2685",
    });
  });

  it("refreshes a rejected checkout quote and requires confirmation again", async () => {
    const fetchQuote = vi
      .fn()
      .mockResolvedValueOnce(
        Response.json({
          creditAmountCents: "10000",
          processingFeeCents: "579",
          fixedFeeCents: "50",
          percentageFeeBps: 500,
          percentageFeeCents: "529",
          subtotalCents: "10579",
          currency: "usd",
        }),
      )
      .mockResolvedValueOnce(
        Response.json({
          creditAmountCents: "10000",
          processingFeeCents: "600",
          fixedFeeCents: "70",
          percentageFeeBps: 500,
          percentageFeeCents: "530",
          subtotalCents: "10600",
          currency: "usd",
        }),
      );
    client.setConfig({ baseUrl: "http://localhost", fetch: fetchQuote });
    const onAddCredit = vi
      .fn()
      .mockRejectedValueOnce(new Error("The checkout price changed. Review the new subtotal."))
      .mockResolvedValueOnce(undefined);
    const { user } = renderDialog(onAddCredit);
    await user.click(screen.getByRole("button", { name: "$100" }));
    const confirm = screen.getByRole("button", { name: "Continue to checkout" });
    await waitFor(() => expect(confirm).toBeEnabled());
    await user.click(confirm);

    expect(await screen.findByRole("alert")).toHaveTextContent("The checkout price changed.");
    expect(await screen.findByText("$106.00")).toBeInTheDocument();
    expect(screen.getByText("Fixed fee per transaction").parentElement).toHaveTextContent("$0.70");
    expect(screen.getByText("Percentage fee (5% of subtotal)").parentElement).toHaveTextContent("$5.30");
    expect(onAddCredit).toHaveBeenCalledOnce();
    await waitFor(() => expect(confirm).toBeEnabled());
    await user.click(confirm);
    expect(onAddCredit).toHaveBeenLastCalledWith({
      productId: "prod-100",
      creditAmountCents: "10000",
      expectedSubtotalCents: "10600",
    });
  });

  it("offers a configured fixed pack that is not a default amount", async () => {
    client.setConfig({
      baseUrl: "http://localhost",
      fetch: vi.fn().mockResolvedValue(
        Response.json({
          creditAmountCents: "2550",
          processingFeeCents: "0",
          fixedFeeCents: "0",
          percentageFeeBps: 0,
          percentageFeeCents: "0",
          subtotalCents: "2550",
          currency: "usd",
        }),
      ),
    });
    const { user, onAddCredit } = renderDialog(vi.fn().mockResolvedValue(undefined), [
      { id: "prod-100", amountCents: "10000" },
      { id: "prod-2550", amountCents: "2550" },
    ]);
    const amounts = screen.getByRole("group", { name: "Hosted credit amount" });

    expect(
      within(amounts)
        .getAllByRole("button")
        .map((button) => button.textContent),
    ).toEqual(["$25.50", "$100", "Custom"]);
    expect(within(amounts).getByRole("button", { name: "Custom" })).toBeDisabled();

    await user.click(within(amounts).getByRole("button", { name: "$25.50" }));
    const confirm = screen.getByRole("button", { name: "Continue to checkout" });
    await waitFor(() => expect(confirm).toBeEnabled());
    await user.click(confirm);
    expect(onAddCredit).toHaveBeenCalledWith({
      productId: "prod-2550",
      creditAmountCents: "2550",
      expectedSubtotalCents: "2550",
    });
  });
});
