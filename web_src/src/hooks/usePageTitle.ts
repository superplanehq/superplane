import { useEffect, useMemo } from "react";

/**
 * usePageTitle
 * Pass an array of title segments; they will be joined with middots (·)
 * and "SuperPlane" is appended as the last element.
 *
 * Example:
 * usePageTitle([workflow.name]) => "{workflow.name} · SuperPlane"
 */
export function usePageTitle(parts: Array<string | undefined | null>) {
  // Derive a stable title string so the effect only fires when the content
  // actually changes. Callers pass inline array literals whose reference
  // changes every render; the derived string stays stable across renders.
  const title = useMemo(() => {
    const cleaned = parts.filter((p): p is string => typeof p === "string" && p.trim().length > 0).map((p) => p.trim());
    return [...cleaned, "SuperPlane"].join(" · ");
  }, [parts]);

  useEffect(() => {
    document.title = title;
  }, [title]);
}

export default usePageTitle;
