import { formatTimeAgo } from "@/lib/date";

export function addDetail(details: Record<string, string>, label: string, value?: string): void {
  const trimmed = value?.trim();
  if (!trimmed) {
    return;
  }
  details[label] = trimmed;
}

export function addFormattedTimestamp(
  details: Record<string, string>,
  label: string,
  createdAt: string | undefined,
): void {
  if (!createdAt) {
    return;
  }
  details[label] = formatTimeAgo(new Date(createdAt));
}
