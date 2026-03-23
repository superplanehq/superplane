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
  useEffect(() => {
    const cleaned = parts.filter((p): p is string => typeof p === "string" && p.trim().length > 0).map((p) => p.trim());

    const segments = [...cleaned, "SuperPlane"];
    document.title = segments.join(" · ");
  }, [JSON.stringify(parts)]);
}

export default usePageTitle;
