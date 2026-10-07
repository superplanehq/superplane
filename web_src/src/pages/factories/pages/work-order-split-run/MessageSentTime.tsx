import type { HTMLAttributes, ReactNode } from "react";

import { cn } from "@/lib/utils";

import { formatWorkOrderDateTime } from "../../lib/workOrderDateTime";

type MessageSentAt = number | string | null | undefined;

const SENT_TIME_CLASSNAME =
  "pointer-events-none absolute z-10 whitespace-nowrap text-[11px] leading-4 tabular-nums text-muted-foreground opacity-0 transition-opacity duration-150 select-none group-hover/message:opacity-100 group-focus/message:opacity-100 group-focus-within/message:opacity-100";

export function MessageSentTime({ sentAt, placement }: { sentAt?: MessageSentAt; placement: "beside" | "overlay" }) {
  const date = messageSentDate(sentAt);
  if (!date) {
    return null;
  }

  return (
    <time
      dateTime={date.toISOString()}
      title={date.toLocaleString()}
      className={cn(
        SENT_TIME_CLASSNAME,
        placement === "beside"
          ? "top-1/2 end-[calc(100%+0.5rem)] -translate-y-1/2"
          : "top-1 end-2 rounded-md bg-background/95 px-1.5 py-0.5",
      )}
    >
      <span className="sr-only">Sent </span>
      {formatWorkOrderDateTime(date)}
    </time>
  );
}

export function MessageTimeRow({
  sentAt,
  className,
  children,
  ...rest
}: {
  sentAt?: MessageSentAt;
  className?: string;
  children: ReactNode;
} & Omit<HTMLAttributes<HTMLDivElement>, "children">) {
  return (
    <div
      {...rest}
      className={cn("group/message relative", className)}
      tabIndex={messageSentDate(sentAt) ? 0 : undefined}
    >
      {children}
    </div>
  );
}

function messageSentDate(sentAt: MessageSentAt): Date | null {
  if (typeof sentAt === "number") {
    if (!Number.isFinite(sentAt) || sentAt <= 0) {
      return null;
    }
  } else if (!sentAt?.trim()) {
    return null;
  }

  const date = new Date(sentAt);
  if (Number.isNaN(date.getTime()) || !formatWorkOrderDateTime(date)) {
    return null;
  }
  return date;
}
