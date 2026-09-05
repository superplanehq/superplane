import { Bot, ChevronDown, ClipboardList, Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { factoryCardClassName } from "@/pages/factories/pages/factoryPageLayoutStyles";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";

import type { FixDispatchTarget, Suggestion } from "./types";
import { QUALITY_DOMAIN_LABELS } from "./types";
import { EnforcementBadge, SeverityBadge } from "./SeverityBadge";
import { formatFindingLocation } from "./findingLocation";

interface SuggestionCardProps {
  suggestion: Suggestion;
  onDispatchFix: (suggestionId: string, target: FixDispatchTarget) => void;
  onDismiss: (suggestionId: string) => void;
  onAccept: (suggestionId: string) => void;
}

/**
 * One open finding as an actionable suggestion: what is wrong, where, how to
 * fix it, and one action that routes the fix as a draft work order or as a
 * direct agent run.
 */
export function SuggestionCard({ suggestion, onDispatchFix, onDismiss, onAccept }: SuggestionCardProps) {
  const { finding } = suggestion;
  const fixInProgress = suggestion.fixStatus === "in-progress";

  return (
    <article className={cn(factoryCardClassName, "flex flex-col gap-3 p-4")} aria-label={finding.ruleName}>
      <div className="flex flex-wrap items-center gap-2">
        <SeverityBadge severity={finding.severity} />
        <span className="text-[13px] font-medium text-foreground">{finding.ruleName}</span>
        <EnforcementBadge enforcement={finding.enforcement} />
        <span className="text-[12px] text-muted-foreground">{QUALITY_DOMAIN_LABELS[finding.domain]}</span>
        {suggestion.occurrences > 1 ? (
          <span className="text-[12px] text-muted-foreground">· {suggestion.occurrences} occurrences</span>
        ) : null}
      </div>

      {finding.location ? (
        <code className="w-fit rounded bg-slate-100 px-1.5 py-0.5 font-mono text-[12px] text-slate-700 dark:bg-gray-900 dark:text-gray-300">
          {formatFindingLocation(finding.location)}
        </code>
      ) : null}

      <p className="workspace-body-text text-foreground">{finding.description}</p>
      <p className="text-[13px] text-muted-foreground">
        <span className="font-medium text-foreground">Fix: </span>
        {finding.remediation}
      </p>

      <div className="flex flex-wrap items-center gap-2 border-t border-border pt-3">
        {fixInProgress ? (
          <span className="inline-flex items-center gap-1.5 text-[13px] font-medium text-blue-700 dark:text-blue-300">
            <Loader2 className="size-3.5 animate-spin" aria-hidden />
            Fix in progress
          </span>
        ) : (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button size="sm">
                Dispatch fix
                <ChevronDown className="size-4" aria-hidden />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start">
              <DropdownMenuItem onClick={() => onDispatchFix(suggestion.id, "work-order")}>
                <ClipboardList className="size-4" aria-hidden />
                Create fix work order
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onDispatchFix(suggestion.id, "agent-run")}>
                <Bot className="size-4" aria-hidden />
                Start agent run
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
        <Button variant="ghost" size="sm" onClick={() => onDismiss(suggestion.id)}>
          Dismiss
        </Button>
        <Button variant="ghost" size="sm" onClick={() => onAccept(suggestion.id)}>
          Accept
        </Button>
        {!fixInProgress ? (
          <p className="text-[12px] text-muted-foreground">
            A fix work order starts as a draft. You choose when to dispatch it.
          </p>
        ) : null}
      </div>
    </article>
  );
}
