import { ChevronRight } from "lucide-react";
import { useState } from "react";

import type { OrganizationsHostedCreditProduct } from "@/api-client";
import { cn } from "@/lib/utils";

import { formatUsdCents, parseWorkOrderMetric } from "../../lib/workOrderUsage";

const BUY_MORE_PACK_CENTS = [5_000, 10_000, 50_000] as const;

export function HostedCreditTopUpBanner({
  checkoutPending,
  packs,
  onAddCredit,
}: {
  checkoutPending: boolean;
  packs: OrganizationsHostedCreditProduct[];
  onAddCredit: (productId: string) => void | Promise<void>;
}) {
  const [open, setOpen] = useState(false);
  const customPack = findCustomCreditPack(packs);
  const customPackId = customPack?.id ?? "";
  const hasPurchasablePack =
    BUY_MORE_PACK_CENTS.some((cents) => findPackForCents(packs, cents)) || Boolean(customPackId);
  const disabled = checkoutPending || !hasPurchasablePack;

  return (
    <div className="mt-3 inline-flex h-8 items-center gap-1.5 rounded-full bg-violet-100 py-1 pr-1.5 pl-2.5 text-[12px] hover:bg-violet-200/80 dark:bg-violet-950 dark:hover:bg-violet-900">
      <button
        type="button"
        disabled={disabled}
        aria-expanded={open}
        aria-label="Top up"
        data-testid="billing-top-up"
        onClick={() => setOpen((current) => !current)}
        className="inline-flex items-center gap-1 whitespace-nowrap font-medium text-violet-800 disabled:pointer-events-none disabled:opacity-50 dark:text-violet-200"
      >
        {checkoutPending ? "Opening checkout..." : "Top up"}
        <ChevronRight className={cn("size-3.5 transition-transform", open && "rotate-180")} aria-hidden />
      </button>
      {open ? (
        <div className="flex items-center gap-1" data-testid="billing-top-up-options">
          {BUY_MORE_PACK_CENTS.map((cents) => {
            const product = findPackForCents(packs, cents);
            const productId = product?.id ?? "";
            return (
              <button
                key={cents}
                type="button"
                disabled={!productId || checkoutPending}
                onClick={() => productId && void onAddCredit(productId)}
                className="inline-flex h-5 items-center rounded-full bg-violet-600 px-2.5 text-[11px] leading-none font-medium text-white hover:bg-violet-700 disabled:pointer-events-none disabled:opacity-40"
              >
                {formatUsdPackLabel(cents)}
              </button>
            );
          })}
          <button
            type="button"
            disabled={!customPackId || checkoutPending}
            title={customPackId ? undefined : "Custom amounts are not available yet."}
            onClick={() => customPackId && void onAddCredit(customPackId)}
            className="inline-flex h-5 items-center rounded-full bg-violet-600 px-2.5 text-[11px] leading-none font-medium text-white hover:bg-violet-700 disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-violet-600"
          >
            Custom
          </button>
        </div>
      ) : null}
    </div>
  );
}

function findPackForCents(packs: OrganizationsHostedCreditProduct[], cents: number) {
  return packs.find(
    (product) =>
      Boolean(product.id) && !isCustomCreditPack(product) && parseWorkOrderMetric(product.amountCents) === cents,
  );
}

function findCustomCreditPack(packs: OrganizationsHostedCreditProduct[]) {
  return packs.find((product) => Boolean(product.id) && isCustomCreditPack(product));
}

function isCustomCreditPack(product: OrganizationsHostedCreditProduct) {
  return parseWorkOrderMetric(product.amountCents) <= 0;
}

function formatUsdPackLabel(cents: number): string {
  const dollars = cents / 100;
  if (Number.isInteger(dollars)) {
    return `$${dollars}`;
  }
  return formatUsdCents(cents);
}
