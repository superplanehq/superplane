import { eventStatusBadgeClassName } from "@/lib/eventStatusBadge";

export function EventStatusBadge({ badgeColor, label }: { badgeColor: string; label: string }) {
  return <span className={eventStatusBadgeClassName(badgeColor)}>{label}</span>;
}
