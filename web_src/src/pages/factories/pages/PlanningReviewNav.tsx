import { cn } from "@/lib/utils";

import { PLANNING_REVIEW_SECTIONS, type PlanningReviewSectionId } from "./planningReviewSections";

/**
 * Left column of the agent editor. It splits the agent into groups so no
 * single panel grows long enough to bury the ones below it.
 */
export function PlanningReviewNav({
  active,
  onSelect,
  stepCount,
}: {
  active: PlanningReviewSectionId;
  onSelect: (id: PlanningReviewSectionId) => void;
  stepCount: number;
}) {
  return (
    <nav
      aria-label="Agent settings"
      className="w-56 shrink-0 overflow-y-auto border-r border-border bg-muted/50 p-3"
      data-testid="planning-review-nav"
    >
      <ul className="flex flex-col gap-0.5">
        {PLANNING_REVIEW_SECTIONS.map((section) => {
          const Icon = section.icon;
          const isActive = section.id === active;
          return (
            <li key={section.id}>
              <button
                type="button"
                aria-current={isActive ? "page" : undefined}
                onClick={() => onSelect(section.id)}
                data-testid={`planning-review-nav-${section.id}`}
                className={cn(
                  "flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-left text-[13px] font-medium transition-colors",
                  isActive
                    ? "bg-card text-foreground shadow-xs"
                    : "text-muted-foreground hover:bg-card/60 hover:text-foreground",
                )}
              >
                <Icon className="size-4 shrink-0" aria-hidden />
                <span className="min-w-0 flex-1 truncate">{section.label}</span>
                {section.id === "steps" ? (
                  <span className="shrink-0 text-xs tabular-nums text-muted-foreground">{stepCount}</span>
                ) : null}
              </button>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
