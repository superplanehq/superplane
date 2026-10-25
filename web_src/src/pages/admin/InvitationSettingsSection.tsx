import type { ChangeEvent, FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface InvitationSettingsSectionProps {
  duration: string;
  savedDuration: number;
  saving: boolean;
  onChange: (value: string) => void;
  onSave: () => void;
}

export function InvitationSettingsSection({
  duration,
  savedDuration,
  saving,
  onChange,
  onSave,
}: InvitationSettingsSectionProps) {
  const days = Number(duration);
  const valid = Number.isInteger(days) && days >= 1 && days <= 365;
  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (valid && !saving && days !== savedDuration) onSave();
  };
  return (
    <section className="py-6">
      <h2 className="text-base font-semibold">Organization invitations</h2>
      <p className="mt-2 text-sm text-muted-foreground">
        Set how long new and resent invitations remain valid for all organizations. Existing invitations keep their
        expiration dates.
      </p>
      <form className="mt-6 flex max-w-sm flex-col items-start gap-3" onSubmit={submit}>
        <Label htmlFor="invitation-duration">Invitation duration (days)</Label>
        <Input
          id="invitation-duration"
          type="number"
          min={1}
          max={365}
          step={1}
          required
          value={duration}
          disabled={saving}
          aria-describedby="invitation-duration-help"
          onChange={(event: ChangeEvent<HTMLInputElement>) => onChange(event.target.value)}
        />
        <p id="invitation-duration-help" className="text-sm text-muted-foreground">
          Enter 1 to 365 days. The default is 7 days.
        </p>
        <Button type="submit" disabled={saving || !valid || days === savedDuration}>
          {saving ? "Saving..." : "Save invitation settings"}
        </Button>
      </form>
    </section>
  );
}
