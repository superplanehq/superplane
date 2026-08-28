import { cn } from "@/lib/utils";
import { Plus, Radio, Settings } from "lucide-react";

/** One always-on automation that feeds a lane, for example an intake. */
export type LaneListener = {
  id: string;
  /** The whole sentence on the row, for example "Listening to GitHub issues". */
  title: string;
  iconSrc: string;
  iconAlt: string;
  healthy: boolean;
  /** Text the badge shows while the automation cannot run. */
  needsRepairLabel: string;
  settingsLabel: string;
  testId: string;
  onOpenSettings: () => void;
};

/**
 * Listeners sit at the head of a lane, above the work orders they open. Flat
 * rows, not cards: a listener is a source, not an item on the board.
 */
export function LaneListenerList({
  listeners,
  testId,
  addLabel,
  addTestId,
  onAdd,
}: {
  listeners: LaneListener[];
  testId: string;
  addLabel?: string;
  addTestId?: string;
  onAdd?: () => void;
}) {
  if (listeners.length === 0 && !onAdd) {
    return null;
  }

  return (
    <ul className="mb-2 flex shrink-0 flex-col gap-1" data-testid={testId}>
      {listeners.map((listener) => (
        <li key={listener.id}>
          <LaneListenerRow listener={listener} />
        </li>
      ))}
      {onAdd ? (
        <li>
          <button
            type="button"
            onClick={onAdd}
            data-testid={addTestId}
            className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-[12px] font-medium tracking-[-0.01em] text-muted-foreground transition-colors hover:bg-background hover:text-foreground"
          >
            <Plus className="size-3.5 shrink-0" aria-hidden />
            {addLabel}
          </button>
        </li>
      ) : null}
    </ul>
  );
}

function LaneListenerRow({ listener }: { listener: LaneListener }) {
  return (
    <button
      type="button"
      onClick={listener.onOpenSettings}
      aria-label={listener.settingsLabel}
      title={listener.settingsLabel}
      data-testid={listener.testId}
      className="group/listener flex w-full items-center gap-2 rounded-md bg-background/60 px-2 py-1.5 text-left transition-colors hover:bg-background"
    >
      <Radio
        className={cn(
          "size-3.5 shrink-0",
          listener.healthy ? "text-emerald-600 dark:text-emerald-400" : "text-amber-600 dark:text-amber-400",
        )}
        aria-hidden
      />
      <img
        src={listener.iconSrc}
        alt=""
        className={cn(
          "size-3.5 shrink-0 object-contain",
          listener.iconAlt === "GitHub" && "dark:brightness-0 dark:invert",
        )}
      />
      <span className="min-w-0 flex-1 truncate text-[12px] font-medium tracking-[-0.01em] text-muted-foreground transition-colors group-hover/listener:text-foreground">
        {listener.title}
      </span>
      {listener.healthy ? null : (
        <span
          className="shrink-0 text-[11px] font-medium text-amber-700 dark:text-amber-400"
          data-testid={`${listener.testId}-needs-repair`}
        >
          {listener.needsRepairLabel}
        </span>
      )}
      <Settings className="size-3.5 shrink-0 text-muted-foreground group-hover/listener:text-foreground" aria-hidden />
    </button>
  );
}
