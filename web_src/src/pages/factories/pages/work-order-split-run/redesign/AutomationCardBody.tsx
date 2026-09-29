import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { CircleStop, History, RotateCw } from "lucide-react";
import { useState } from "react";

import type { FilesFile } from "@/api-client";

import { extractArtifactMarkdownBody, toArtifactDataRecord } from "../../../lib/workOrderArtifact";
import { WorkOrderCheckBody } from "../../../WorkOrderCheckDialog";
import { WorkOrderPullRequestInline } from "../../../WorkOrderPullRequestInline";
import type { SplitRunPhase } from "../splitRunMocks";
import { WorkOrderSplitRunDescription } from "../WorkOrderSplitRunDescription";
import type { AutomationStage, ConsoleAutomation } from "./automationsViewModel";
import { runFooterLine } from "./consoleCardText";
import { ArtifactChip } from "./consoleOutputChips";
import { consolePanes, defaultConsolePaneId, type ConsolePane } from "./consolePanes";
import { LiveAgentSteps } from "./LiveAgentSteps";
import { META_TEXT_CLASSNAME } from "./redesignFormat";

// Line tabs on the shadcn Tabs primitives, after ReUI c-tabs-2.
const PANE_TABS_LIST =
  "h-auto w-full justify-start gap-1 overflow-x-auto rounded-none border-b border-border bg-transparent p-0 dark:bg-transparent";
const PANE_TAB =
  "h-7 flex-none rounded-none border-0 border-b-2 border-transparent px-2 text-[12px] font-medium text-muted-foreground shadow-none " +
  "data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none " +
  "dark:data-[state=active]:border-primary dark:data-[state=active]:bg-transparent";

export function AutomationCardBody({
  automation,
  phase,
  organizationId,
  factoryId,
  orderId,
  taskDescription,
  canEditDescription = false,
  descriptionBusy = false,
  onDescriptionSave,
  files,
  onStop,
  onRetry,
  actionBusy,
  onOpen,
}: {
  automation: ConsoleAutomation;
  phase?: SplitRunPhase;
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  taskDescription?: string;
  canEditDescription?: boolean;
  descriptionBusy?: boolean;
  onDescriptionSave?: (next: string) => void | Promise<void>;
  files?: FilesFile[];
  onStop?: () => void;
  onRetry?: () => void;
  actionBusy: boolean;
  onOpen: () => void;
}) {
  const { latest, runs } = automation;
  const panes = consolePanes(latest, phase);
  const [chosenId, setChosenId] = useState<string>();
  const activeId = chosenId && panes.some((pane) => pane.id === chosenId) ? chosenId : defaultConsolePaneId(panes);
  const active = panes.find((pane) => pane.id === activeId);
  return (
    <div className="space-y-3">
      {panes.length > 1 ? (
        <Tabs value={activeId} onValueChange={setChosenId}>
          <TabsList aria-label="Run details" className={PANE_TABS_LIST}>
            {panes.map((pane) => (
              <TabsTrigger key={pane.id} value={pane.id} className={PANE_TAB}>
                {pane.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      ) : null}
      {/* The log stays mounted so live steps and spend keep streaming. */}
      <div className={cn(active?.kind !== "log" && "hidden")}>
        <LiveAgentSteps stage={latest} phase={phase} organizationId={organizationId} />
      </div>
      {active && active.kind !== "log" ? (
        <ConsolePaneBody
          pane={active}
          stage={latest}
          organizationId={organizationId}
          factoryId={factoryId}
          orderId={orderId}
          taskDescription={taskDescription}
          canEditDescription={canEditDescription}
          descriptionBusy={descriptionBusy}
          onDescriptionSave={onDescriptionSave}
          files={files}
        />
      ) : null}
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2 border-t pt-3">
        <span className={cn(META_TEXT_CLASSNAME, "min-w-0")}>{runFooterLine(latest)}</span>
        <div className="ms-auto flex shrink-0 items-center gap-1.5">
          {runs.length > 1 ? (
            <Button size="sm" variant="ghost" className="h-7 gap-1.5 px-2 text-[12px]" onClick={onOpen}>
              <History className="size-3.5" aria-hidden />
              View {runs.length} runs
            </Button>
          ) : null}
          {onRetry ? (
            <Button size="sm" variant="outline" className="gap-1.5" disabled={actionBusy} onClick={onRetry}>
              <RotateCw className="size-3.5" aria-hidden />
              Retry
            </Button>
          ) : null}
          {onStop ? (
            <Button size="sm" variant="outline" className="gap-1.5" disabled={actionBusy} onClick={onStop}>
              <CircleStop className="size-3.5" aria-hidden />
              Stop
            </Button>
          ) : null}
        </div>
      </div>
    </div>
  );
}

function ConsolePaneBody({
  pane,
  stage,
  organizationId,
  factoryId,
  orderId,
  taskDescription,
  canEditDescription,
  descriptionBusy,
  onDescriptionSave,
  files,
}: {
  pane: ConsolePane;
  stage: AutomationStage;
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  taskDescription?: string;
  canEditDescription: boolean;
  descriptionBusy: boolean;
  onDescriptionSave?: (next: string) => void | Promise<void>;
  files?: FilesFile[];
}) {
  if (pane.kind === "description") {
    if (stage.id === "backlog") {
      return (
        <div data-testid="redesign-console-task-description">
          <WorkOrderSplitRunDescription
            description={taskDescription ?? ""}
            canEdit={canEditDescription}
            busy={descriptionBusy}
            collapsible={!canEditDescription}
            onSave={onDescriptionSave}
            files={files}
            organizationId={organizationId}
            factoryId={factoryId}
            orderId={orderId}
          />
        </div>
      );
    }
    return (
      <div className="text-[12.5px] leading-5 text-muted-foreground">
        <MarkdownContent content={stage.description ?? ""} variant="workspace" />
      </div>
    );
  }
  if (pane.kind === "artifact" && pane.artifact) {
    return <ArtifactPane artifact={pane.artifact} />;
  }
  if (pane.kind === "check" && pane.check) {
    return <WorkOrderCheckBody check={pane.check} />;
  }
  if (pane.kind === "pullRequest") {
    const pullRequest = stage.outputs.pullRequests[0];
    return pullRequest ? (
      <WorkOrderPullRequestInline pullRequest={pullRequest} showTitle className="text-[12px]" />
    ) : null;
  }
  return null;
}

/** Markdown artifacts read inline, like the description. Other kinds keep their open and download actions. */
function ArtifactPane({ artifact }: { artifact: AutomationStage["outputs"]["artifacts"][number] }) {
  const kind = (artifact.type ?? "").replace(/^TYPE_/i, "").toLowerCase();
  const body = kind === "markdown" ? extractArtifactMarkdownBody(toArtifactDataRecord(artifact.data)) : undefined;
  if (body) {
    return (
      <div className="text-[12.5px] leading-5 text-muted-foreground">
        <MarkdownContent content={body} variant="workspace" />
      </div>
    );
  }
  return (
    <div className="flex flex-wrap items-center gap-2">
      <ArtifactChip artifact={artifact} />
    </div>
  );
}
