import { ChevronRight } from "lucide-react";
import { useRef, useState } from "react";

import type { OrganizationsCreateHostedCreditCheckoutBody, OrganizationsHostedCreditProduct } from "@/api-client";
import { Button } from "@/components/ui/button";

import { HostedCreditCheckoutDialog } from "./HostedCreditCheckoutDialog";
import { hasPurchasableHostedCreditPack } from "./hostedCreditPacks";

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
  const topUpButton = useRef<HTMLButtonElement>(null);
  const hasPurchasablePack = hasPurchasableHostedCreditPack(packs);

  return (
    <>
      <Button
        ref={topUpButton}
        type="button"
        variant="outline"
        size="sm"
        disabled={checkoutPending || !hasPurchasablePack}
        aria-haspopup="dialog"
        aria-label="Top up"
        data-testid="billing-top-up"
        onClick={() => setOpen(true)}
      >
        {checkoutPending ? "Opening checkout..." : "Top up"}
        <ChevronRight className="size-3.5" aria-hidden />
      </Button>
      {open ? (
        <HostedCreditCheckoutDialog
          organizationId={organizationId}
          packs={packs}
          checkoutPending={checkoutPending}
          onAddCredit={onAddCredit}
          onClose={() => setOpen(false)}
          onCloseAutoFocus={(event) => {
            event.preventDefault();
            topUpButton.current?.focus();
          }}
        />
      ) : null}
    </>
  );
}
