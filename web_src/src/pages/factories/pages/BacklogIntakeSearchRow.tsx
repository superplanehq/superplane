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
