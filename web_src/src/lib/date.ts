export const formatTimeAgo = (date: Date, includeAgo = true): string => {
  const suffix = includeAgo ? " ago" : "";
  const seconds = Math.max(0, Math.floor((new Date().getTime() - date.getTime()) / 1000));

  if (seconds < 60) return `${seconds}s${suffix}`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m${suffix}`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h${suffix}`;
  const days = Math.floor(hours / 24);
  return `${days}d${suffix}`;
};
