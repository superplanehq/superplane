import { Frame, FrameHeader, FramePanel, FrameTitle } from "@/components/reui/frame";
import {
  TimelineContent,
  TimelineHeader,
  TimelineIndicator,
  TimelineItem,
  TimelineSeparator,
  TimelineTitle,
} from "@/components/reui/timeline";
import superplaneIcon from "@/assets/superplane.svg";
import { logoDarkInvertClass } from "@/lib/logoDarkMode";
import { cn } from "@/lib/utils";
import { Bot } from "lucide-react";

import type { FilesFile } from "@/api-client";

import {
  addedByForSource,
  CREATED_MANUALLY,
  type SplitRunAddedBy,
  type SplitRunSource,
  isMonochromeSourceLogo,
} from "../splitRunSource";
import { WorkOrderSplitRunDescription } from "../WorkOrderSplitRunDescription";
import { WorkOrderSplitRunSource } from "../WorkOrderSplitRunSource";

function intakeSourceIcon(source?: SplitRunSource): { src: string; alt: string } {
  if (source?.kind === "intake") {
    return { src: source.iconSrc, alt: source.iconAlt };
  }
  return { src: superplaneIcon, alt: "SuperPlane" };
}

/** Invert a dark logo on the foreground disk. The disk is light in dark mode. */
function intakeMarkerIconClass(icon: { src: string; alt: string }): string | undefined {
  if (!isMonochromeSourceLogo(icon.alt) && !logoDarkInvertClass(icon.src)) {
    return undefined;
  }
  return "brightness-0 invert dark:brightness-100 dark:invert-0";
}

export function IntakeTimelineEvent({
  source,
  description,
  canEdit,
  busy,
  onSave,
  files,
  organizationId,
  factoryId,
  orderId,
}: {
  source?: SplitRunSource;
  description?: string;
  canEdit: boolean;
  busy: boolean;
  onSave?: (next: string) => void | Promise<void>;
  files?: FilesFile[];
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
}) {
  const icon = intakeSourceIcon(source);
  return (
    <TimelineItem step={1} data-testid="redesign-console-column-intake">
      <TimelineHeader className="flex items-center gap-2">
        <TimelineSeparator />
        <TimelineIndicator
          aria-hidden={false}
          className="flex size-5 items-center justify-center overflow-hidden border-none bg-foreground"
          data-testid="redesign-console-column-marker-intake"
        >
          <img src={icon.src} alt="" className={cn("size-3.5 object-contain", intakeMarkerIconClass(icon))} />
          <span className="sr-only">{icon.alt}</span>
        </TimelineIndicator>
        <TimelineTitle className="font-semibold">Intake</TimelineTitle>
      </TimelineHeader>
      <TimelineContent className="mt-2 text-foreground">
        <IntakeTaskDescription
          description={description}
          canEdit={canEdit}
          busy={busy}
          onSave={onSave}
          source={source}
          files={files}
          organizationId={organizationId}
          factoryId={factoryId}
          orderId={orderId}
        />
      </TimelineContent>
    </TimelineItem>
  );
}

/** Preview height so the timeline stays on screen. Show more reveals the rest. */
const TASK_DESCRIPTION_PREVIEW_PX = 160;

function IntakeTaskDescription({
  description,
  canEdit,
  busy,
  onSave,
  source,
  files,
  organizationId,
  factoryId,
  orderId,
}: {
  description?: string;
  canEdit: boolean;
  busy: boolean;
  onSave?: (next: string) => void | Promise<void>;
  source?: SplitRunSource;
  files?: FilesFile[];
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
}) {
  const hasBody = Boolean(description?.trim()) || canEdit;
  if (!hasBody && !source) {
    return null;
  }
  return (
    <Frame
      variant="default"
      spacing="sm"
      stacked
      className="[--frame-radius:var(--radius-lg)]"
      data-testid="redesign-console-task-description"
    >
      <FrameHeader className="flex flex-row items-center justify-between gap-2">
        {source ? (
          <WorkOrderSplitRunSource compact source={source} />
        ) : (
          <FrameTitle className="text-[13px] font-medium">Task</FrameTitle>
        )}
        {source ? <IntakeAddedBy source={source} /> : null}
      </FrameHeader>
      {hasBody ? (
        <FramePanel>
          <WorkOrderSplitRunDescription
            description={description ?? ""}
            canEdit={canEdit}
            busy={busy}
            previewHeight={TASK_DESCRIPTION_PREVIEW_PX}
            fadeClassName="from-card via-card/90"
            onSave={onSave}
            files={files}
            organizationId={organizationId}
            factoryId={factoryId}
            orderId={orderId}
          />
        </FramePanel>
      ) : null}
    </Frame>
  );
}

function IntakeAddedBy({ source }: { source: SplitRunSource }) {
  const addedBy = addedByForSource(source);
  return (
    <span
      className="flex shrink-0 items-center gap-1 text-[11px] font-medium text-muted-foreground"
      data-testid="redesign-console-intake-added-by"
    >
      {addedBy.kind === "intake" ? <Bot className="size-3" aria-hidden /> : null}
      <span>{addedByLabel(addedBy)}</span>
    </span>
  );
}

function addedByLabel(addedBy: SplitRunAddedBy): string {
  if (addedBy.kind === "intake") {
    return `Intake ${addedBy.name}`;
  }
  if (addedBy.kind === "imported") {
    return `Imported by ${addedBy.personName}`;
  }
  return CREATED_MANUALLY;
}
