import { useEffect, useRef, useState, type ChangeEvent, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";

import type {
  OrganizationsCreateHostedCreditCheckoutBody,
  OrganizationsHostedCreditProduct,
  OrganizationsQuoteHostedCreditCheckoutResponse,
} from "@/api-client";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { hostedCreditProductsQueryKey, useHostedCreditCheckoutQuote } from "@/hooks/useLLMModelAllowlists";
import { getApiErrorMessage } from "@/lib/errors";
import { cn } from "@/lib/utils";

import { formatUsdCents } from "../../lib/workOrderUsage";
import {
  fixedHostedCreditPackCents,
  hasCustomHostedCreditPack,
  hostedCreditPackButtonLabel,
  hostedCreditPackCents,
} from "./hostedCreditPacks";

export function HostedCreditCheckoutDialog({
  organizationId,
  packs,
  checkoutPending,
  onAddCredit,
  onClose,
  onCloseAutoFocus,
}: {
  organizationId: string;
  packs: OrganizationsHostedCreditProduct[];
  checkoutPending: boolean;
  onAddCredit: (input: OrganizationsCreateHostedCreditCheckoutBody) => Promise<void>;
  onClose: () => void;
  onCloseAutoFocus?: (event: Event) => void;
}) {
  const queryClient = useQueryClient();
  const [selection, setSelection] = useState<number | "custom" | null>(null);
  const [dollars, setDollars] = useState("");
  const [checkoutError, setCheckoutError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const quoteInput = creditQuoteInput(packs, selection, dollars);
  const quote = useHostedCreditCheckoutQuote(organizationId, quoteInput);
  const subtotal = Number(quote.data?.subtotalCents);
  const quoteValid = Boolean(
    quoteInput && quote.isSuccess && !quote.isFetching && validQuote(quote.data, quoteInput.creditAmountCents),
  );
  const pending = checkoutPending || submitting;
  const quoteError = quoteErrorMessage(quote, Boolean(quoteInput), quoteValid);
  const refreshQuote = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: hostedCreditProductsQueryKey(organizationId) }),
      quote.refetch(),
    ]);

  const chooseAmount = (next: number | "custom") => {
    setSelection(next);
    setCheckoutError("");
  };
  const startCheckout = async () => {
    if (!quoteInput || !quoteValid || pending) return;
    setSubmitting(true);
    setCheckoutError("");
    try {
      await onAddCredit({ ...quoteInput, expectedSubtotalCents: String(subtotal) });
    } catch (error) {
      setCheckoutError(getApiErrorMessage(error, "Unable to start checkout. Try again."));
      await refreshQuote();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !pending) onClose();
      }}
    >
      <DialogContent
        className="max-h-[calc(100dvh-2rem)] overflow-y-auto"
        showCloseButton={!pending}
        onCloseAutoFocus={onCloseAutoFocus}
      >
        <DialogHeader>
          <DialogTitle>Add hosted credit</DialogTitle>
          <DialogDescription>Choose the amount of hosted credit to add to this organization.</DialogDescription>
        </DialogHeader>
        <CreditAmountSelection
          packs={packs}
          selection={selection}
          dollars={dollars}
          pending={pending}
          onChoose={chooseAmount}
          onDollarsChange={(value) => {
            setDollars(value);
            setCheckoutError("");
          }}
        />
        <CreditQuoteSummary quote={quoteValid ? quote.data : undefined} loading={quote.isFetching} />
        {quoteError || checkoutError ? (
          <p className="text-sm text-destructive" role="alert">
            {quoteError || checkoutError}
          </p>
        ) : null}
        {quoteError ? (
          <Button type="button" variant="outline" onClick={() => void refreshQuote()}>
            Try again
          </Button>
        ) : null}
        <p className="text-xs text-muted-foreground">Complete payment on Polar.</p>
        <DialogFooter>
          <Button type="button" variant="outline" disabled={pending} onClick={onClose}>
            Cancel
          </Button>
          <Button type="button" disabled={pending || !quoteValid} onClick={() => void startCheckout()}>
            {pending ? "Opening checkout..." : "Continue to checkout"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function CreditAmountSelection({
  packs,
  selection,
  dollars,
  pending,
  onChoose,
  onDollarsChange,
}: {
  packs: OrganizationsHostedCreditProduct[];
  selection: number | "custom" | null;
  dollars: string;
  pending: boolean;
  onChoose: (selection: number | "custom") => void;
  onDollarsChange: (value: string) => void;
}) {
  const amountInput = useRef<HTMLInputElement>(null);
  const invalidAmount = dollars !== "" && parseCreditAmount(dollars) === null;
  useEffect(() => {
    if (selection === "custom") amountInput.current?.focus();
  }, [selection]);
  return (
    <>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4" role="group" aria-label="Hosted credit amount">
        {fixedHostedCreditPackCents(packs).map((cents) => (
          <CreditAmountButton
            key={cents}
            selected={selection === cents}
            disabled={pending}
            onClick={() => onChoose(cents)}
          >
            {hostedCreditPackButtonLabel(cents)}
          </CreditAmountButton>
        ))}
        <CreditAmountButton
          selected={selection === "custom"}
          disabled={pending || !hasCustomHostedCreditPack(packs)}
          onClick={() => onChoose("custom")}
        >
          Custom
        </CreditAmountButton>
      </div>
      {selection === "custom" ? (
        <div className="space-y-2">
          <Label htmlFor="hosted-credit-amount">Credit amount in USD</Label>
          <Input
            id="hosted-credit-amount"
            ref={amountInput}
            inputMode="decimal"
            value={dollars}
            placeholder="100.00"
            disabled={pending}
            aria-invalid={invalidAmount}
            aria-describedby={invalidAmount ? "hosted-credit-amount-error" : undefined}
            onChange={(event: ChangeEvent<HTMLInputElement>) => onDollarsChange(event.target.value)}
          />
          {invalidAmount ? (
            <p id="hosted-credit-amount-error" className="text-sm text-destructive" role="alert">
              Enter a positive USD amount with no more than two decimal places.
            </p>
          ) : null}
        </div>
      ) : null}
    </>
  );
}

function CreditQuoteSummary({
  quote,
  loading,
}: {
  quote?: OrganizationsQuoteHostedCreditCheckoutResponse;
  loading: boolean;
}) {
  return (
    <div aria-live="polite" aria-atomic="true" className="rounded-lg border border-border bg-muted/30 p-4">
      {quote ? (
        <dl className="space-y-3 text-sm">
          <div className="flex justify-between gap-4">
            <dt>Hosted credit</dt>
            <dd className="tabular-nums">{formatUsdCents(Number(quote.creditAmountCents))}</dd>
          </div>
          <div className="flex justify-between gap-4">
            <dt>Processing fee</dt>
            <dd className="tabular-nums">{formatUsdCents(Number(quote.processingFeeCents))}</dd>
          </div>
          <div className="flex justify-between gap-4 pl-3 text-xs text-muted-foreground">
            <dt>Fixed fee per transaction</dt>
            <dd className="shrink-0 tabular-nums">{formatUsdCents(Number(quote.fixedFeeCents))}</dd>
          </div>
          <div className="flex justify-between gap-4 pl-3 text-xs text-muted-foreground">
            <dt>Percentage fee ({Number(quote.percentageFeeBps) / 100}% of subtotal)</dt>
            <dd className="shrink-0 tabular-nums">{formatUsdCents(Number(quote.percentageFeeCents))}</dd>
          </div>
          <div className="flex justify-between gap-4 border-t border-border pt-3 font-semibold">
            <dt>Subtotal</dt>
            <dd className="tabular-nums">{formatUsdCents(Number(quote.subtotalCents))}</dd>
          </div>
        </dl>
      ) : (
        <p className="text-sm text-muted-foreground">
          {loading ? "Calculating subtotal..." : "Choose an amount to see the subtotal."}
        </p>
      )}
      {quote && Number(quote.percentageFeeBps) > 0 ? (
        <p className="mt-3 text-xs text-muted-foreground">The percentage fee is rounded up to the nearest cent.</p>
      ) : null}
      <p className="mt-3 text-xs text-muted-foreground">Tax is calculated at checkout.</p>
    </div>
  );
}

function creditQuoteInput(
  packs: OrganizationsHostedCreditProduct[],
  selection: number | "custom" | null,
  dollars: string,
) {
  const custom = selection === "custom";
  const product = packs.find((pack) => {
    const cents = hostedCreditPackCents(pack);
    return cents !== null && (custom ? cents === 0 : cents === selection);
  });
  const creditCents = custom ? parseCreditAmount(dollars) : selection;
  if (!product?.id || creditCents === null || creditCents <= 0) return undefined;
  return { productId: product.id, creditAmountCents: String(creditCents) };
}

function quoteErrorMessage(quote: ReturnType<typeof useHostedCreditCheckoutQuote>, hasInput: boolean, valid: boolean) {
  if (!hasInput || quote.isFetching) return "";
  if (quote.isError) return getApiErrorMessage(quote.error, "Unable to calculate the subtotal. Try again.");
  if (quote.isSuccess && !valid) return "Unable to calculate the subtotal. Try again.";
  return "";
}

function validQuote(quote: OrganizationsQuoteHostedCreditCheckoutResponse, creditAmountCents: string) {
  const fee = Number(quote.processingFeeCents);
  const fixedFee = Number(quote.fixedFeeCents);
  const percentageFee = Number(quote.percentageFeeCents);
  const percentageBps = Number(quote.percentageFeeBps);
  const subtotal = Number(quote.subtotalCents);
  return (
    quote.creditAmountCents === creditAmountCents &&
    Number.isSafeInteger(fee) &&
    fee >= 0 &&
    Number.isSafeInteger(fixedFee) &&
    fixedFee >= 0 &&
    Number.isSafeInteger(percentageFee) &&
    percentageFee >= 0 &&
    fixedFee + percentageFee === fee &&
    Number.isSafeInteger(percentageBps) &&
    percentageBps >= 0 &&
    percentageBps < 10000 &&
    Number.isSafeInteger(subtotal) &&
    subtotal === Number(creditAmountCents) + fee &&
    quote.currency?.toLowerCase() === "usd"
  );
}

function CreditAmountButton({
  selected,
  children,
  ...props
}: {
  selected: boolean;
  children: ReactNode;
  disabled: boolean;
  onClick: () => void;
}) {
  return (
    <Button
      type="button"
      variant="outline"
      aria-pressed={selected}
      className={cn(
        "h-12",
        selected &&
          "border-violet-500 bg-violet-50 text-violet-800 hover:bg-violet-100 dark:bg-violet-950 dark:text-violet-200",
      )}
      {...props}
    >
      {children}
    </Button>
  );
}

function parseCreditAmount(value: string): number | null {
  const match = /^(\d+)(?:\.(\d{1,2}))?$/.exec(value.trim());
  if (!match) return null;
  const cents = Number(match[1]) * 100 + Number((match[2] ?? "").padEnd(2, "0"));
  return Number.isSafeInteger(cents) && cents > 0 ? cents : null;
}
