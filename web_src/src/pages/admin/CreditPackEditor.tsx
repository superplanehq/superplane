import { useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { centsToDollarInput, parseDollarInputToCents } from "@/lib/hostedCredit";
import type { CreditPack, CreditPackInput } from "./creditPacksApi";
import { PolarProductSelect } from "./PolarProductSelect";

export function CreditPackEditor({
  pack,
  onSave,
  onCancel,
}: {
  pack: CreditPack | null;
  onSave: (input: CreditPackInput) => Promise<void>;
  onCancel: () => void;
}) {
  const [returnFocus] = useState(() => document.activeElement);
  const {
    name,
    setName,
    amount,
    setAmount,
    custom,
    setCustom,
    productID,
    setProductID,
    enabled,
    setEnabled,
    saving,
    error,
    amountValid,
    valid,
    save,
  } = useCreditPackForm(pack, onSave);

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !saving) onCancel();
      }}
    >
      <DialogContent
        className="max-h-[90vh] overflow-y-auto"
        showCloseButton={!saving}
        onCloseAutoFocus={(event) => {
          if (returnFocus instanceof HTMLElement) {
            event.preventDefault();
            returnFocus.focus();
          }
        }}
      >
        <DialogHeader>
          <DialogTitle>{pack ? "Edit credit pack" : "Create credit pack"}</DialogTitle>
          <DialogDescription>
            Set the credit amount and link a Polar product for checkout. Saved packs replace the list from Polar.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={(event) => void save(event)} className="space-y-6">
          <fieldset disabled={saving} className="space-y-5">
            <div className="space-y-2">
              <Label htmlFor="credit-pack-name">Name</Label>
              <Input id="credit-pack-name" required value={name} onChange={(event) => setName(event.target.value)} />
            </div>
            <div className="flex items-center gap-3">
              <Switch id="credit-pack-custom" checked={custom} onCheckedChange={setCustom} />
              <Label htmlFor="credit-pack-custom">Custom amount</Label>
            </div>
            {custom ? (
              <p className="text-sm text-muted-foreground">Customers choose an amount. Polar product limits apply.</p>
            ) : (
              <div className="space-y-2">
                <Label htmlFor="credit-pack-amount">Credit amount (USD)</Label>
                <Input
                  id="credit-pack-amount"
                  inputMode="decimal"
                  value={amount}
                  onChange={(event) => setAmount(event.target.value)}
                  aria-invalid={amount !== "" && !amountValid}
                  aria-describedby="credit-pack-amount-help"
                />
                <p id="credit-pack-amount-help" className="text-sm text-muted-foreground">
                  Enter $0.50 to $999,999.99, with up to 2 decimal places. Processing fees are added at checkout.
                </p>
              </div>
            )}
            <PolarProductSelect value={productID} onChange={setProductID} disabled={saving} />
            <div className="space-y-2">
              <div className="flex items-center gap-3">
                <Switch id="credit-pack-enabled" checked={enabled} onCheckedChange={setEnabled} />
                <Label htmlFor="credit-pack-enabled">Enabled</Label>
              </div>
              <p className="text-sm text-muted-foreground">
                Enabled packs with a Polar product appear in organization billing.
              </p>
            </div>
          </fieldset>
          {error && (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          )}
          <DialogFooter>
            <Button type="button" variant="outline" disabled={saving} onClick={onCancel}>
              Cancel
            </Button>
            <Button type="submit" disabled={!valid || saving}>
              {saving ? "Saving..." : "Save pack"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function useCreditPackForm(pack: CreditPack | null, onSave: (input: CreditPackInput) => Promise<void>) {
  const [name, setName] = useState(pack?.name ?? "");
  const [amount, setAmount] = useState(pack && pack.amount_cents > 0 ? centsToDollarInput(pack.amount_cents) : "");
  const [custom, setCustom] = useState(pack?.amount_cents === 0);
  const [productID, setProductID] = useState(pack?.polar_product_id ?? "");
  const [enabled, setEnabled] = useState(pack?.enabled ?? true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const amountCents = parsePackAmount(amount);
  const amountValid = custom || amountCents !== null;
  const valid = !!name.trim() && amountValid;

  const save = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!valid || saving || (!custom && amountCents === null)) return;
    setSaving(true);
    setError("");
    try {
      await onSave({
        name: name.trim(),
        amount_cents: custom ? 0 : (amountCents ?? 0),
        enabled,
        polar_product_id: productID.trim() || null,
      });
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Could not save the credit pack. Try again.");
    } finally {
      setSaving(false);
    }
  };

  return {
    name,
    setName,
    amount,
    setAmount,
    custom,
    setCustom,
    productID,
    setProductID,
    enabled,
    setEnabled,
    saving,
    error,
    amountValid,
    valid,
    save,
  };
}

function parsePackAmount(amount: string): number | null {
  if (!/^\d+(\.\d{1,2})?$/.test(amount.trim())) return null;
  const cents = parseDollarInputToCents(amount);
  if (cents === null || cents < 50 || cents > 99999999) return null;
  return cents;
}
