import { useState } from "react";

// Responsive sizing keeps the avatar readable on wide dashboards without
// pushing rows taller on narrow layouts. Ring + neutral background provide
// a visible edge on both light and dark themes and double as the fallback
// disc when the image URL is missing or fails to load.
const AVATAR_CLASS =
  "inline-block size-5 shrink-0 rounded-full bg-slate-200 object-cover ring-1 ring-slate-950/10 sm:size-6 dark:bg-gray-700 dark:ring-white/10";

/**
 * Renders a resolved URL as a circular inline avatar. Blank values collapse
 * to the em-dash placeholder used by hidden cells; broken URLs swap the
 * `<img>` for a neutral disc so the browser's broken-image icon never
 * leaks into the table.
 */
export function AvatarCell({ url, label }: { url: string; label: string }) {
  // Track the URL that failed (not just a boolean) so a prop change to a new
  // URL automatically retries with a fresh <img> instead of staying stuck on
  // the fallback disc.
  const [failedUrl, setFailedUrl] = useState<string | null>(null);
  const trimmed = url.trim();
  if (!trimmed) {
    return <span className="text-slate-300 dark:text-gray-600">—</span>;
  }
  if (failedUrl === trimmed) {
    return <span className={AVATAR_CLASS} aria-label={label} data-testid="widget-avatar-fallback" />;
  }
  return (
    <img
      src={trimmed}
      alt={label}
      loading="lazy"
      referrerPolicy="no-referrer"
      onError={() => setFailedUrl(trimmed)}
      className={AVATAR_CLASS}
      data-testid="widget-avatar"
    />
  );
}
