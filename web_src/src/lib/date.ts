export const formatTimeAgo = (date: Date, includeAgo = true): string => {
  const deltaSeconds = Math.floor((Date.now() - date.getTime()) / 1000);
  const isFuture = deltaSeconds < 0;
  const seconds = Math.abs(deltaSeconds);

  let unit: string;
  if (seconds < 60) {
    unit = `${seconds}s`;
  } else {
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) {
      unit = `${minutes}m`;
    } else {
      const hours = Math.floor(minutes / 60);
      unit = hours < 24 ? `${hours}h` : `${Math.floor(hours / 24)}d`;
    }
  }

  if (isFuture) return `in ${unit}`;
  return includeAgo ? `${unit} ago` : unit;
};
