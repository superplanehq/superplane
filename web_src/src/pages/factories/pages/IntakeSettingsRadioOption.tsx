import { cn } from "@/lib/utils";

export function IntakeSettingsRadioOption({
  name,
  value,
  checked,
  title,
  helper,
  disabled = false,
  onChange,
}: {
  name: string;
  value: string;
  checked: boolean;
  title: string;
  helper?: string;
  disabled?: boolean;
  onChange: () => void;
}) {
  return (
    <label
      className={cn(
        "flex items-start gap-3 rounded-lg border px-3 py-2.5 transition-colors",
        disabled ? "cursor-not-allowed opacity-60" : "cursor-pointer",
        checked ? "border-foreground/20 bg-accent/50" : "border-border bg-card hover:border-foreground/15",
      )}
    >
      <input
        type="radio"
        name={name}
        value={value}
        checked={checked}
        disabled={disabled}
        onChange={onChange}
        className="mt-0.5 size-4 accent-gray-900"
      />
      <span className="min-w-0">
        <span className="block text-[13px] font-medium tracking-[-0.01em] text-foreground">{title}</span>
        {helper ? <span className="mt-0.5 block text-[12px] leading-5 text-muted-foreground">{helper}</span> : null}
      </span>
    </label>
  );
}
