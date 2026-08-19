import { useEffect, useMemo } from "react";

export interface UsePageTitleOptions {
  /**
   * Set to `false` to skip writing `document.title` for this call while still
   * running the hook unconditionally (keeps the rules-of-hooks call order
   * stable). Defaults to `true`.
   *
   * Layout components that render a nested route's `<Outlet/>` (e.g.
   * `FactoriesLayout`, `FactorySettingsLayout`) use this to stop setting a
   * baseline title once a leaf page has mounted: React fires a newly-mounted
   * child's effects before its already-mounted parent's effects, so an
   * unconditional parent `usePageTitle` call would otherwise overwrite the
   * leaf page's more specific title on the very commit it first renders
   * (e.g. a hard reload straight to a work order permalink). Once the leaf
   * is mounted it becomes the sole owner of `document.title`.
   */
  enabled?: boolean;
}

/**
 * usePageTitle
 * Pass an array of title segments; they will be joined with middots (·)
 * and "SuperPlane" is appended as the last element.
 *
 * Example:
 * usePageTitle([workflow.name]) => "{workflow.name} · SuperPlane"
 */
export function usePageTitle(parts: Array<string | undefined | null>, options?: UsePageTitleOptions) {
  const enabled = options?.enabled ?? true;

  // Derive a stable title string so the effect only fires when the content
  // actually changes. Callers pass inline array literals whose reference
  // changes every render; the derived string stays stable across renders.
  const title = useMemo(() => {
    const cleaned = parts.filter((p): p is string => typeof p === "string" && p.trim().length > 0).map((p) => p.trim());
    return [...cleaned, "SuperPlane"].join(" · ");
  }, [parts]);

  useEffect(() => {
    if (!enabled) {
      return;
    }
    document.title = title;
  }, [title, enabled]);
}

export default usePageTitle;
