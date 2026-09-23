import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";

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
  const switchId = `${testId}-switch`;

  return (
    <div
      className="flex w-full items-center gap-4 bg-card px-3 py-2.5 text-left"
      data-testid={testId}
      data-active={active ? "true" : "false"}
    >
      <Label htmlFor={switchId} className="min-w-0 flex-1 cursor-pointer flex-col items-start">
        <span className="block text-[13px] font-medium tracking-[-0.01em] text-foreground">{title}</span>
        <span className="mt-0.5 block text-[12px] font-normal text-muted-foreground">{description}</span>
      </Label>
      <Switch id={switchId} checked={active} onCheckedChange={() => onToggle()} aria-label={title} />
    </div>
  );
}
