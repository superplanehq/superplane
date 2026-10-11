import type { LucideIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { toDate } from "@/lib/datetime";
import { formatIntakeCreatedTime } from "./backlogIntakeCreatedTime";
import type { BacklogIntakeItem } from "./backlogIntakeItems";

export function BacklogIntakeSearchRow({
  item,
  onImportItem,
}: {
  item: BacklogIntakeItem;
  onImportItem: (item: BacklogIntakeItem) => void;
}) {
  const createdAt = toDate(item.createdAt);
  const createdTimeLabel = formatIntakeCreatedTime(createdAt ?? undefined);

  return (
    <button
      type="button"
      className="flex w-full flex-col rounded-md px-2.5 py-2 text-left text-sm hover:bg-accent"
      data-testid={`lines-backlog-create-item-${item.id}`}
      onClick={() => onImportItem(item)}
    >
      <span className="flex w-full items-center gap-2">
        <span className="min-w-0 flex-1 truncate">{item.title}</span>
        <span className="shrink-0 text-[12px] text-muted-foreground">{item.key}</span>
      </span>
      {createdAt && createdTimeLabel ? (
        <time
          dateTime={createdAt.toISOString()}
          title={`Created ${createdAt.toLocaleString()}`}
          className="text-[12px] text-muted-foreground"
        >
          {createdTimeLabel}
        </time>
      ) : null}
    </button>
  );
}

export function CreateMenuAction({
  testId,
  icon: Icon,
  title,
  hint,
  onClick,
}: {
  testId: string;
  icon: LucideIcon;
  title: string;
  hint: string;
  onClick: () => void;
}) {
  return (
    <Button
      type="button"
      variant="ghost"
      aria-label={title}
      className="h-auto w-full items-start justify-start gap-3 rounded-md px-2.5 py-2.5 text-left font-normal whitespace-normal hover:bg-accent"
      data-testid={testId}
      onClick={onClick}
    >
      <Icon className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
      <span className="min-w-0">
        <span className="block text-sm font-medium">{title}</span>
        <span className="mt-0.5 block text-[13px] text-muted-foreground">{hint}</span>
      </span>
    </Button>
  );
}
