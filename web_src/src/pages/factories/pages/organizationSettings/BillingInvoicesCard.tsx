import { Check, ExternalLink, Receipt } from "lucide-react";

import type { OrganizationsHostedCreditInvoice } from "@/api-client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

import { formatUsdCents, parseWorkOrderMetric } from "../../lib/workOrderUsage";
import { FactorySettingsCard } from "../settings/FactorySettingsCard";

const INVOICES_DESCRIPTION = "Recent paid invoices for this organization.";
const INVOICES_EMPTY_TITLE = "No invoices yet";
const INVOICES_EMPTY_BODY = "Paid invoices appear here after checkout.";

export function BillingInvoicesCard({
  invoices,
  portalPending,
  onManageInvoices,
}: {
  invoices: OrganizationsHostedCreditInvoice[];
  portalPending: boolean;
  onManageInvoices: () => void | Promise<void>;
}) {
  return (
    <FactorySettingsCard
      title="Invoices"
      data-testid="billing-polar-invoices"
      action={
        <Button
          type="button"
          size="sm"
          variant="ghost"
          disabled={portalPending}
          onClick={() => void onManageInvoices()}
        >
          {portalPending ? "Opening invoices..." : "Manage invoices"}
          <ExternalLink className="size-3.5" aria-hidden />
        </Button>
      }
    >
      <p className="text-[12px] text-muted-foreground">{INVOICES_DESCRIPTION}</p>
      {invoices.length === 0 ? <InvoiceEmptyState /> : <InvoiceList invoices={invoices} />}
    </FactorySettingsCard>
  );
}

function InvoiceEmptyState() {
  return (
    <div className="mt-4 flex flex-col items-center rounded-lg border border-dashed border-border px-6 py-8 text-center">
      <Receipt className="size-5 text-muted-foreground" aria-hidden />
      <p className="mt-2 text-sm font-medium text-foreground">{INVOICES_EMPTY_TITLE}</p>
      <p className="mt-1 text-xs text-muted-foreground">{INVOICES_EMPTY_BODY}</p>
    </div>
  );
}

function InvoiceList({ invoices }: { invoices: OrganizationsHostedCreditInvoice[] }) {
  return (
    <ul className="mt-3 divide-y divide-border">
      {invoices.map((invoice) => (
        <InvoiceRow key={invoice.id} invoice={invoice} />
      ))}
    </ul>
  );
}

function InvoiceRow({ invoice }: { invoice: OrganizationsHostedCreditInvoice }) {
  const productName = invoice.productName || "Hosted credit";
  const amount = formatUsdCents(parseWorkOrderMetric(invoice.amountCents));
  const dateLabel = formatInvoiceDate(invoice.createdAt);

  return (
    <li className="flex items-center gap-3 py-3 first:pt-1 last:pb-0">
      <div className="flex size-8 shrink-0 items-center justify-center rounded-md bg-muted">
        <Receipt className="size-3.5 text-muted-foreground" aria-hidden />
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-[13px] font-medium tracking-[-0.01em]">{productName}</p>
        {invoice.createdAt ? (
          <time className="mt-0.5 block text-xs text-muted-foreground" dateTime={invoice.createdAt}>
            {dateLabel}
          </time>
        ) : (
          <p className="mt-0.5 text-xs text-muted-foreground">{dateLabel}</p>
        )}
      </div>
      <p className="shrink-0 text-[13px] font-medium tracking-[-0.01em] tabular-nums">{amount}</p>
      <InvoiceStatusBadge status={invoice.status} />
    </li>
  );
}

function InvoiceStatusBadge({ status }: { status: string | undefined }) {
  const label = invoiceStatusLabel(status);
  if (status === "paid") {
    return (
      <Badge
        variant="outline"
        className="border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400"
      >
        <Check aria-hidden />
        {label}
      </Badge>
    );
  }

  return (
    <Badge variant="outline" className="text-muted-foreground">
      {label}
    </Badge>
  );
}

function formatInvoiceDate(value: string | undefined) {
  if (!value) {
    return "—";
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
}

function invoiceStatusLabel(status: string | undefined) {
  switch (status) {
    case "paid":
      return "Paid";
    case "refunded":
      return "Refunded";
    case "partially_refunded":
      return "Partially refunded";
    default:
      return status || "Unknown";
  }
}
