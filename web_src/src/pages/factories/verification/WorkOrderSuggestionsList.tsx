import { useState } from "react";

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import type { FindingSeverity, FixDispatchTarget, QualityDomain, Suggestion } from "./types";
import { QUALITY_DOMAIN_LABELS } from "./types";
import { SuggestionCard } from "./SuggestionCard";

const SEVERITY_FILTER_LABELS: Record<FindingSeverity, string> = {
  high: "High",
  medium: "Medium",
  low: "Low",
};

interface WorkOrderSuggestionsListProps {
  suggestions: Suggestion[];
  onDispatchFix: (suggestionId: string, target: FixDispatchTarget) => void;
  onDismiss: (suggestionId: string) => void;
  onAccept: (suggestionId: string) => void;
}

/**
 * Open suggestions for one work order, with severity and domain filters.
 * The empty state confirms a clean verification run.
 */
export function WorkOrderSuggestionsList({
  suggestions,
  onDispatchFix,
  onDismiss,
  onAccept,
}: WorkOrderSuggestionsListProps) {
  const [severityFilter, setSeverityFilter] = useState<"all" | FindingSeverity>("all");
  const [domainFilter, setDomainFilter] = useState<"all" | QualityDomain>("all");

  const filtered = suggestions.filter((suggestion) => {
    if (severityFilter !== "all" && suggestion.finding.severity !== severityFilter) return false;
    if (domainFilter !== "all" && suggestion.finding.domain !== domainFilter) return false;
    return true;
  });

  return (
    <section className="flex flex-col gap-3" aria-label="Suggestions">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-col gap-0.5">
          <h3 className="workspace-section-title text-foreground">Suggestions</h3>
          <p className="text-[12px] text-muted-foreground">
            Open findings from the last verification run, with a prepared fix for each.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Select value={severityFilter} onValueChange={(value) => setSeverityFilter(value as "all" | FindingSeverity)}>
            <SelectTrigger size="sm" aria-label="Filter by severity">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All severities</SelectItem>
              {(Object.keys(SEVERITY_FILTER_LABELS) as FindingSeverity[]).map((severity) => (
                <SelectItem key={severity} value={severity}>
                  {SEVERITY_FILTER_LABELS[severity]}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select value={domainFilter} onValueChange={(value) => setDomainFilter(value as "all" | QualityDomain)}>
            <SelectTrigger size="sm" aria-label="Filter by domain">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All domains</SelectItem>
              {(Object.keys(QUALITY_DOMAIN_LABELS) as QualityDomain[]).map((domain) => (
                <SelectItem key={domain} value={domain}>
                  {QUALITY_DOMAIN_LABELS[domain]}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {filtered.length === 0 ? (
        <EmptySuggestionsState hasSuggestions={suggestions.length > 0} />
      ) : (
        <div className="flex flex-col gap-3">
          {filtered.map((suggestion) => (
            <SuggestionCard
              key={suggestion.id}
              suggestion={suggestion}
              onDispatchFix={onDispatchFix}
              onDismiss={onDismiss}
              onAccept={onAccept}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function EmptySuggestionsState({ hasSuggestions }: { hasSuggestions: boolean }) {
  return (
    <div className="flex flex-col items-center gap-1 rounded-lg border border-dashed border-border bg-card px-4 py-10 text-center">
      <p className="workspace-body-text text-foreground">
        {hasSuggestions ? "No suggestions match the filters." : "No open suggestions."}
      </p>
      <p className="text-[13px] text-muted-foreground">
        {hasSuggestions
          ? "Remove a filter to see the other suggestions."
          : "The last verification run reported no open findings."}
      </p>
    </div>
  );
}
