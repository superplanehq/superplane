import { cn } from "@/lib/utils";

/** Stacked title, description, and switch used by Jira and Sentry intake events. */
export function IntakeEventRow({
  title,
  description,
  active,
  onToggle,
  testId,
}: {
  title: string;
  description: string;
  active: boolean;
  onToggle: () => void;
  testId: string;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={active}
      aria-label={title}
      data-testid={testId}
      data-active={active ? "true" : "false"}
      onClick={onToggle}
      className="flex w-full items-center gap-4 bg-card px-3 py-2.5 text-left hover:bg-accent/40"
    >
      <span className="min-w-0 flex-1">
        <span className="block text-[13px] font-medium tracking-[-0.01em] text-foreground">{title}</span>
        <span className="mt-0.5 block text-[12px] text-muted-foreground">{description}</span>
      </span>
      <span
        aria-hidden
        className={cn(
          "relative inline-flex h-5 w-9 shrink-0 rounded-full transition-colors",
          active ? "bg-foreground" : "bg-muted-foreground/30",
        )}
      >
        <span
          className={cn(
            "absolute top-0.5 size-4 rounded-full bg-background shadow-sm transition-transform",
            active ? "translate-x-4" : "translate-x-0.5",
          )}
        />
      </span>
    </button>
  );
}
