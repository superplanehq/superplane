import { cn } from "@/lib/utils";
import { ChevronRight } from "lucide-react";

import githubIcon from "@/assets/icons/integrations/github.svg";
import slackIcon from "@/assets/icons/integrations/slack.svg";
import superplaneIcon from "@/assets/superplane.svg";

import { PhaseGlyph } from "../linePhaseGlyph";
import {
  overlayStatusGlyph,
  overlayStatusLabel,
  type RunOverlayProvider,
  type RunOverlayStep,
} from "./workOrderRunOverlayMocks";

function providerIcon(provider: RunOverlayProvider): string {
  if (provider === "github") return githubIcon;
  if (provider === "slack") return slackIcon;
  return superplaneIcon;
}

/**
 * Compact horizontal job graph (GitHub Actions / CircleCI). Not a ticket
 * timeline and not the full canvas.
 */
export function CompactPipelineGraph({
  steps,
  selectedId,
  onSelect,
}: {
  steps: RunOverlayStep[];
  selectedId?: string | null;
  onSelect?: (id: string) => void;
}) {
  return (
    <ol className="flex min-w-0 items-stretch gap-0 overflow-x-auto pb-1 [scrollbar-width:thin]">
      {steps.map((step, index) => (
        <li key={step.id} className="flex min-w-0 items-center">
          {index > 0 ? <ChevronRight className="mx-1 size-3.5 shrink-0 text-muted-foreground/60" aria-hidden /> : null}
          <button
            type="button"
            onClick={() => onSelect?.(step.id)}
            className={cn(
              "w-[10.5rem] shrink-0 rounded-lg border bg-card px-3 py-2.5 text-left transition-colors",
              selectedId === step.id
                ? "border-foreground/25 bg-accent/40"
                : "border-border hover:border-foreground/15 hover:bg-accent/30",
            )}
          >
            <span className="flex items-center gap-1.5">
              <img
                src={providerIcon(step.provider)}
                alt=""
                className={cn("size-3.5 shrink-0", step.provider !== "slack" && "opacity-90 dark:invert")}
              />
              <span className="truncate text-[12px] font-medium text-foreground">{step.title}</span>
            </span>
            <span className="mt-1.5 flex items-center gap-1.5 text-[11px] text-muted-foreground">
              <PhaseGlyph kind={overlayStatusGlyph(step.status)} className="size-3" />
              {overlayStatusLabel(step.status)}
              {step.duration ? ` · ${step.duration}` : ""}
            </span>
          </button>
        </li>
      ))}
    </ol>
  );
}

/** Vertical step list for the phase inspector (Vercel / Railway deploy). */
export function PhaseStepList({
  steps,
  selectedId,
  onSelect,
}: {
  steps: RunOverlayStep[];
  selectedId?: string | null;
  onSelect?: (id: string) => void;
}) {
  return (
    <ol className="space-y-1">
      {steps.map((step) => (
        <li key={step.id}>
          <button
            type="button"
            onClick={() => onSelect?.(step.id)}
            className={cn(
              "flex w-full items-start gap-2.5 rounded-lg border px-3 py-2 text-left transition-colors",
              selectedId === step.id ? "border-foreground/20 bg-accent/40" : "border-transparent hover:bg-accent/30",
            )}
          >
            <PhaseGlyph kind={overlayStatusGlyph(step.status)} className="mt-0.5" />
            <span className="min-w-0 flex-1">
              <span className="block truncate text-[13px] font-medium text-foreground">{step.title}</span>
              <span className="block truncate text-[12px] text-muted-foreground">
                {step.detail ?? overlayStatusLabel(step.status)}
                {step.duration ? ` · ${step.duration}` : ""}
              </span>
            </span>
            <img
              src={providerIcon(step.provider)}
              alt=""
              className={cn("mt-0.5 size-3.5 shrink-0", step.provider !== "slack" && "opacity-90 dark:invert")}
            />
          </button>
        </li>
      ))}
    </ol>
  );
}
