import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { billingCatalogRequest } from "./billingPlansApi";

type SeatLimitState = {
  plan_limit: number | null;
  override: { seat_limit: number | null } | null;
  effective_limit: number | null;
};
type LimitSelection = "plan" | "custom" | "unlimited";

export function OrganizationSeatLimit({ orgId }: { orgId: string }) {
  const [state, setState] = useState<SeatLimitState | null>(null);
  const [selection, setSelection] = useState<LimitSelection>("plan");
  const [custom, setCustom] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [reload, setReload] = useState(0);
  const apply = (next: SeatLimitState) => {
    setState(next);
    setSelection(next.override === null ? "plan" : next.override.seat_limit === null ? "unlimited" : "custom");
    setCustom(next.override?.seat_limit?.toString() || "");
    setError("");
  };
  useEffect(() => {
    let cancelled = false;
    setState(null);
    void billingCatalogRequest<SeatLimitState>(`organizations/${orgId}/seat-limit`).then(
      (next) => {
        if (!cancelled) apply(next);
      },
      () => {
        if (!cancelled) setError("Failed to load seat limit. Try again.");
      },
    );
    return () => {
      cancelled = true;
    };
  }, [orgId, reload]);
  const save = async () => {
    const value = Number(custom);
    if (selection === "custom" && (!Number.isInteger(value) || value < 1 || value > 2147483647)) {
      setError("Enter a whole number from 1 to 2147483647.");
      return;
    }
    setBusy(true);
    try {
      apply(
        await billingCatalogRequest<SeatLimitState>(
          `organizations/${orgId}/seat-limit`,
          selection === "plan" ? "DELETE" : "PUT",
          selection === "plan" ? undefined : { seat_limit: selection === "unlimited" ? null : value },
        ),
      );
    } catch {
      setError("Failed to save seat limit. Try again.");
    } finally {
      setBusy(false);
    }
  };
  return (
    <section className="mt-6 space-y-3">
      <h3 className="font-semibold">Seats</h3>
      {error && (
        <div role="alert">
          {error}{" "}
          <Button onClick={() => setReload(reload + 1)} disabled={busy}>
            Try again
          </Button>
        </div>
      )}
      {!state && !error && <p>Loading seat limit...</p>}
      {state && (
        <>
          <p className="text-sm text-muted-foreground">
            Plan limit: {state.plan_limit ?? "Unlimited"} · Effective limit: {state.effective_limit ?? "Unlimited"}
          </p>
          <Label htmlFor="seat-limit-selection">Seat limit override</Label>
          <Select value={selection} onValueChange={(value: LimitSelection) => setSelection(value)} disabled={busy}>
            <SelectTrigger id="seat-limit-selection">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="plan">Use plan limit</SelectItem>
              <SelectItem value="custom">Custom limit</SelectItem>
              <SelectItem value="unlimited">Unlimited</SelectItem>
            </SelectContent>
          </Select>
          {selection === "custom" && (
            <div className="space-y-2">
              <Label htmlFor="seat-limit-custom">Custom limit</Label>
              <Input
                id="seat-limit-custom"
                type="number"
                min={1}
                max={2147483647}
                step={1}
                value={custom}
                onChange={(event) => setCustom(event.target.value)}
                disabled={busy}
              />
            </div>
          )}
          <Button onClick={() => void save()} disabled={busy}>
            {busy ? "Saving..." : "Save seat limit"}
          </Button>
        </>
      )}
    </section>
  );
}
