import { ScopePills, type ScopePillOption } from "../../workOrders/header/ScopePills";

export type OverviewScope = "all" | "my";

const OVERVIEW_SCOPES: ReadonlyArray<ScopePillOption<OverviewScope>> = [
  { id: "all", label: "All", tooltip: "Everything in this workspace." },
  { id: "my", label: "My", tooltip: "Only tasks assigned to you." },
];

/**
 * All / My scope toggle in the page header actions. Reuses the Tasks
 * scope pills so the control stays one pattern across pages. Scopes the
 * three task tables; health metrics and workspace proposals always
 * stay workspace-wide.
 */
export function OverviewScopeToggle({
  value,
  onChange,
}: {
  value: OverviewScope;
  onChange: (scope: OverviewScope) => void;
}) {
  return <ScopePills value={value} onChange={onChange} options={OVERVIEW_SCOPES} testIdPrefix="overview-scope" />;
}
