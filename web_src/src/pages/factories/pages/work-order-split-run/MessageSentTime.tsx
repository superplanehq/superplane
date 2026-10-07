import { useLayoutEffect, useRef, useState, type HTMLAttributes, type ReactNode, type Ref } from "react";

import { cn } from "@/lib/utils";

import { formatWorkOrderDateTime } from "../../lib/workOrderDateTime";

type MessageSentAt = number | string | null | undefined;

const SENT_TIME_CLASSNAME =
  "pointer-events-none z-10 whitespace-nowrap text-[11px] leading-4 tabular-nums text-muted-foreground opacity-0 transition-opacity duration-150 select-none group-hover/message:opacity-100 group-focus/message:opacity-100 group-focus-within/message:opacity-100";

export function MessageSentTime({
  sentAt,
  placement,
  labelRef,
}: {
  sentAt?: MessageSentAt;
  placement: "beside" | "overlay";
  labelRef?: Ref<HTMLSpanElement>;
}) {
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
          ? "absolute top-1/2 end-[calc(100%+0.5rem)] -translate-y-1/2"
          : "absolute top-1 end-2 rounded-md bg-background/95 px-1.5 py-0.5",
      )}
    >
      <span className="sr-only">Sent </span>
      <span ref={labelRef}>{formatWorkOrderDateTime(date)}</span>
    </time>
  );
}

const BESIDE_GAP_PX = 8;

export function BesideSentTime({ sentAt, children }: { sentAt?: MessageSentAt; children: ReactNode }) {
  const frameRef = useRef<HTMLDivElement>(null);
  const labelRef = useRef<HTMLSpanElement>(null);
  const [placement, setPlacement] = useState<"beside" | "overlay">("overlay");

  useLayoutEffect(() => {
    const frame = frameRef.current;
    if (!frame || !messageSentDate(sentAt)) {
      return;
    }

    const place = () => {
      const row = frame.closest("[data-message-time-row]");
      const label = labelRef.current;
      if (!row || !label) {
        return;
      }
      const room = frame.getBoundingClientRect().left - row.getBoundingClientRect().left;
      const needed = label.getBoundingClientRect().width + BESIDE_GAP_PX;
      const next = needed > BESIDE_GAP_PX && room + 1 >= needed ? "beside" : "overlay";
      setPlacement((current) => (current === next ? current : next));
    };

    place();
    const observer = new ResizeObserver(place);
    observer.observe(frame);
    const row = frame.closest("[data-message-time-row]");
    if (row) {
      observer.observe(row);
    }
    return () => observer.disconnect();
  }, [sentAt]);

  return (
    <div ref={frameRef} className="relative max-w-full">
      <MessageSentTime sentAt={sentAt} placement={placement} labelRef={labelRef} />
      <div className={cn("w-fit max-w-full", placement === "overlay" && "pt-7")}>{children}</div>
    </div>
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
      data-message-time-row=""
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
