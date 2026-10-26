import { useState, type ChangeEvent, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { showErrorToast } from "@/lib/toast";

const MAX_GRACE_DAYS = 21;

interface PastDueGraceSettingsSectionProps {
  settings: { past_due_grace_days?: number } | null;
  patchSettings: (body: Record<string, unknown>, successMessage: string, fallbackError: string) => Promise<void>;
}

export function PastDueGraceSettingsSection({ settings, patchSettings }: PastDueGraceSettingsSectionProps) {
  const savedDays = settings?.past_due_grace_days ?? 0;
  const [draft, setDraft] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const graceDays = draft ?? String(savedDays);
  const days = Number(graceDays);
  const valid = graceDays.trim() !== "" && Number.isInteger(days) && days >= 0 && days <= MAX_GRACE_DAYS;

  const save = async () => {
    setSaving(true);
    try {
      await patchSettings(
        { past_due_grace_days: days },
        "Payment settings updated",
        "Could not save payment settings. Try again.",
      );
      setDraft(null);
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Could not save payment settings. Try again.");
    } finally {
      setSaving(false);
    }
  };
  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (valid && !saving && days !== savedDays) void save();
  };

  return (
    <section className="border-t py-6">
      <h2 className="text-base font-semibold">Failed renewal payments</h2>
      <p className="mt-2 text-sm text-muted-foreground">
        Set how long Team features stay on after a renewal payment fails. No new members can join during this time.
      </p>
      <form className="mt-6 flex max-w-sm flex-col items-start gap-3" onSubmit={submit}>
        <Label htmlFor="past-due-grace-days">Payment grace period (days)</Label>
        <Input
          id="past-due-grace-days"
          type="number"
          min={0}
          max={MAX_GRACE_DAYS}
          step={1}
          required
          value={graceDays}
          disabled={saving}
          aria-describedby="past-due-grace-days-help"
          onChange={(event: ChangeEvent<HTMLInputElement>) => setDraft(event.target.value)}
        />
        <p id="past-due-grace-days-help" className="text-sm text-muted-foreground">
          Enter 0 to 21 days. The default is 0 days, which turns Team features off at once.
        </p>
        <Button type="submit" disabled={saving || !valid || days === savedDays}>
          {saving ? "Saving..." : "Save payment settings"}
        </Button>
      </form>
    </section>
  );
}
