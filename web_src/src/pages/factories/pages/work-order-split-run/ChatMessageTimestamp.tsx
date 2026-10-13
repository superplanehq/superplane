import { useState, type ReactNode } from "react";

import { toDate, type TimestampInput } from "@/lib/datetime";
import { cn } from "@/lib/utils";

export function ChatMessageTimestamp({
  createdAt,
  align,
  children,
}: {
  createdAt?: TimestampInput;
  align: "start" | "end";
  children: ReactNode;
}) {
  const [hovered, setHovered] = useState(false);
  const date = toDate(createdAt);
  if (!date) {
    return <>{children}</>;
  }
  const clock = new Intl.DateTimeFormat(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(date);

  return (
    <div onMouseEnter={() => setHovered(true)} onMouseLeave={() => setHovered(false)}>
      {children}
      <div
        className={cn(
          "flex px-2 pb-1 text-[11px] leading-4 text-muted-foreground",
          align === "end" ? "-mt-1 justify-end" : "pt-1",
        )}
        style={{ visibility: hovered ? "visible" : "hidden" }}
      >
        <time dateTime={date.toISOString()}>{clock}</time>
      </div>
    </div>
  );
}
