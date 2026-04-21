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

export const formatRelativeTime = (dateString: string): { relative: string; full: string } => {
  const date = new Date(dateString);
  const now = new Date();
  const seconds = Math.max(0, Math.floor((now.getTime() - date.getTime()) / 1000));

  let relative: string;
  if (seconds < 60) {
    relative = `${seconds}s ago`;
  } else if (seconds < 3600) {
    const minutes = Math.floor(seconds / 60);
    relative = `${minutes}m ago`;
  } else if (seconds < 86400) {
    const hours = Math.floor(seconds / 3600);
    relative = `${hours}h ago`;
  } else if (seconds < 604800) {
    const days = Math.floor(seconds / 86400);
    relative = `${days}d ago`;
  } else {
    const weeks = Math.floor(seconds / 604800);
    relative = `${weeks}w ago`;
  }

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
