import { ChevronRight } from "lucide-react";
import { useState } from "react";

import type { OrganizationsCreateHostedCreditCheckoutBody, OrganizationsHostedCreditProduct } from "@/api-client";
import { Button } from "@/components/ui/button";

import { HostedCreditCheckoutDialog } from "./HostedCreditCheckoutDialog";

export function HostedCreditTopUpBanner({
  organizationId,
  checkoutPending,
  packs,
  onAddCredit,
}: {
  organizationId: string;
  checkoutPending: boolean;
  packs: OrganizationsHostedCreditProduct[];
  onAddCredit: (input: OrganizationsCreateHostedCreditCheckoutBody) => Promise<void>;
}) {
  const [open, setOpen] = useState(false);
  const hasPurchasablePack = packs.some(
    (pack) =>
      pack.id &&
      (Number(pack.amountCents ?? 0) <= 0 || [5_000, 10_000, 50_000].includes(Number(pack.amountCents ?? 0))),
  );

  return (
    <>
      <div className="mt-3 inline-flex h-8 items-center gap-1.5 rounded-full bg-violet-100 py-1 pr-1.5 pl-2.5 text-[12px] hover:bg-violet-200/80 dark:bg-violet-950 dark:hover:bg-violet-900">
        <Button
          type="button"
          variant="ghost"
          size="xs"
          disabled={checkoutPending || !hasPurchasablePack}
          aria-haspopup="dialog"
          aria-label="Top up"
          data-testid="billing-top-up"
          onClick={() => setOpen(true)}
          className="h-auto gap-1 px-0 py-0 text-[12px] font-medium text-violet-800 hover:bg-transparent dark:text-violet-200 dark:hover:bg-transparent"
        >
          {checkoutPending ? "Opening checkout..." : "Top up"}
          <ChevronRight className="size-3.5" aria-hidden />
        </Button>
      </div>
      {open ? (
        <HostedCreditCheckoutDialog
          organizationId={organizationId}
          packs={packs}
          checkoutPending={checkoutPending}
          onAddCredit={onAddCredit}
          onClose={() => setOpen(false)}
        />
      ) : null}
    </>
  );
}
