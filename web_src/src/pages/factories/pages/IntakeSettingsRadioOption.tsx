import { cn } from "@/lib/utils";

export function IntakeSettingsRadioOption({
  name,
  value,
  checked,
  title,
  helper,
  disabled = false,
  variant = "card",
  onChange,
}: {
  name: string;
  value: string;
  checked: boolean;
  title: string;
  helper?: string;
  disabled?: boolean;
  variant?: "card" | "plain";
  onChange: () => void;
}) {
  return (
    <label
      className={cn(
        "flex items-start gap-3",
        variant === "card" && "rounded-lg border px-3 py-2.5 transition-colors",
        disabled ? "cursor-not-allowed opacity-60" : "cursor-pointer",
        variant === "card" &&
          (checked ? "border-foreground/20 bg-accent/50" : "border-border bg-card hover:border-foreground/15"),
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
        <span
          className={cn(
            "block text-foreground",
            variant === "plain" ? "text-sm leading-5" : "text-[13px] font-medium tracking-[-0.01em]",
          )}
        >
          {title}
        </span>
        {helper ? <span className="mt-0.5 block text-[12px] leading-5 text-muted-foreground">{helper}</span> : null}
      </span>
    </label>
  );
}
