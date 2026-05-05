import { formatRelativeTime as formatTimezoneRelativeTime } from "./timezone";

export const formatTimeAgo = (date: Date): string => {
  const seconds = Math.max(0, Math.floor((new Date().getTime() - date.getTime()) / 1000));

  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
};

export const formatRelativeTimeWithTooltip = (dateString: string): { relative: string; full: string } => {
  const date = new Date(dateString);
  const relative = formatTimezoneRelativeTime(dateString, true); // Use existing function with abbreviated format

  const full = date.toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
    hour12: true,
  });

  return { relative, full };
};

