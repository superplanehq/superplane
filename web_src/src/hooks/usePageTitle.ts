import { useEffect } from "react";

/**
 * usePageTitle
 * Pass an array of title segments; they will be joined with middots (·)
 * and "SuperPlane" is appended as the last element.
 *
 * Example:
 * usePageTitle([workflow.name]) => "{workflow.name} · SuperPlane"
 */
export function usePageTitle(parts: Array<string | undefined | null>) {
  // Serialize to a stable string so the effect only fires when content
  // actually changes. Callers pass inline array literals whose reference
  // changes every render.
  const partsKey = JSON.stringify(parts);

  useEffect(() => {
    const cleaned = parts.filter((p): p is string => typeof p === "string" && p.trim().length > 0).map((p) => p.trim());

    const segments = [...cleaned, "SuperPlane"];
    document.title = segments.join(" · ");
    // eslint-disable-next-line react-hooks/exhaustive-deps -- keyed on serialized value, not reference
  }, [partsKey]);
}

export default usePageTitle;
