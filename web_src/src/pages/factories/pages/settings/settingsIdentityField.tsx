import { Avatar } from "@/components/Avatar/avatar";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { getNameInitials } from "@/lib/nameInitials";

export function SettingsIdentityField({
  name,
  nameId,
  nameTestId,
  avatarTestId,
  initialsFrom,
  maxLength,
  disabled,
  error,
  helperText,
  onNameChange,
}: {
  name: string;
  nameId: string;
  nameTestId: string;
  avatarTestId?: string;
  initialsFrom?: string;
  maxLength: number;
  disabled?: boolean;
  error?: string;
  helperText?: string;
  onNameChange: (next: string) => void;
}) {
  const initialsSource = initialsFrom || name;

  return (
    <div className="flex items-start gap-4">
      <Avatar
        initials={getNameInitials(initialsSource) || "?"}
        alt={initialsSource}
        className="size-16 shrink-0"
        data-testid={avatarTestId}
      />
      <div className="min-w-0 flex-1 space-y-2">
        <Label htmlFor={nameId}>Name</Label>
        <Input
          id={nameId}
          data-testid={nameTestId}
          value={name}
          maxLength={maxLength}
          disabled={disabled}
          onChange={(event) => onNameChange(event.target.value)}
        />
        {helperText ? <p className="text-[12px] text-muted-foreground">{helperText}</p> : null}
        {error ? <p className="text-[11px] text-destructive">{error}</p> : null}
      </div>
    </div>
  );
}
