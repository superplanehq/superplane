import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ChartNoAxesCombined, GitFork, LayoutDashboard, ListChecks, Plus, Settings2 } from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router-dom";

import { FactoryAutomations } from "./FactoryAutomations";
import { FactoryOverview } from "./FactoryOverview";
import { FactoryWorkDashboard } from "./FactoryWorkDashboard";
import { NewWorkDialog } from "./NewWorkDialog";
import { RepositoryVelocityPanel } from "./RepositoryVelocityPanel";
import type { CreateWorkRequest, FactoryTab, WorkspacePageData, WorkspaceProject } from "./types";

interface SoftwareFactoryHomeProps {
  data: WorkspacePageData;
  defaultTab?: FactoryTab;
  onCreateWork?: (request: CreateWorkRequest) => void;
}

export function SoftwareFactoryHome({ data, defaultTab = "overview", onCreateWork }: SoftwareFactoryHomeProps) {
  const [newWorkOpen, setNewWorkOpen] = useState(false);
  const navigate = useNavigate();
  const attentionCount = data.workItems.filter((workItem) => workItem.status === "attention").length;
  const openWorkItem = (workItemId: string) => navigate(`work/${workItemId}`);

  return (
    <>
      <div className="mx-auto w-full max-w-[1600px] px-4 py-5 sm:px-6 sm:py-6 lg:px-8">
        <FactoryHeader project={data.project} onNewWork={() => setNewWorkOpen(true)} />
        <Tabs defaultValue={defaultTab}>
          <div className="overflow-x-auto pb-1">
            <TabsList aria-label="Factory views">
              <TabsTrigger value="overview">
                <LayoutDashboard />
                Overview
                {attentionCount > 0 ? <AttentionCount count={attentionCount} /> : null}
              </TabsTrigger>
              <TabsTrigger value="work-orders">
                <ListChecks />
                Work Orders
              </TabsTrigger>
              <TabsTrigger value="automations">
                <GitFork />
                Automations
              </TabsTrigger>
              <TabsTrigger value="velocity">
                <ChartNoAxesCombined />
                Velocity
              </TabsTrigger>
            </TabsList>
          </div>

          <TabsContent value="overview" className="mt-3">
            <FactoryOverview
              metrics={data.metrics}
              workItems={data.workItems}
              automations={data.automations}
              onOpenWorkItem={(workItem) => openWorkItem(workItem.id)}
            />
          </TabsContent>
          <TabsContent value="work-orders" className="mt-3">
            <FactoryWorkDashboard workItems={data.workItems} onOpenWorkItem={(workItem) => openWorkItem(workItem.id)} />
          </TabsContent>
          <TabsContent value="automations" className="mt-3">
            <FactoryAutomations
              automations={data.automations}
              onOpenAutomation={(automation) => navigate(`automations/${automation.canvasId}`)}
            />
          </TabsContent>
          <TabsContent value="velocity" className="mt-3">
            <RepositoryVelocityPanel velocity={data.repositoryVelocity} repositories={data.project.repositories} />
          </TabsContent>
        </Tabs>
      </div>
      <NewWorkDialog open={newWorkOpen} onOpenChange={setNewWorkOpen} onCreateWork={onCreateWork} />
    </>
  );
}

function AttentionCount({ count }: { count: number }) {
  return (
    <span className="ml-0.5 flex size-4 items-center justify-center rounded-full bg-amber-100 text-[10px] font-semibold text-amber-800 dark:bg-amber-950 dark:text-amber-300">
      {count}
    </span>
  );
}

function FactoryHeader({ project, onNewWork }: { project: WorkspaceProject; onNewWork: () => void }) {
  const repositoryLabel =
    project.repositories.length === 1 ? "1 repository" : `${project.repositories.length} repositories`;

  return (
    <header className="mb-5 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
      <div className="min-w-0">
        <div className="mb-1.5 flex items-center gap-2 text-xs text-slate-500 dark:text-gray-400">
          <span>Software Factory</span>
          <span aria-hidden="true" className="text-slate-300 dark:text-gray-600">
            /
          </span>
          <span className="inline-flex items-center gap-1.5 text-emerald-700 dark:text-emerald-400">
            <span className="size-1.5 rounded-full bg-emerald-500" />
            Operational
          </span>
        </div>
        <h1 className="text-2xl font-semibold text-slate-900 dark:text-gray-100">{project.name}</h1>
        {project.description ? (
          <p className="mt-1.5 max-w-3xl text-sm leading-5 text-slate-600 dark:text-gray-400">{project.description}</p>
        ) : null}
        <p className="mt-2 text-xs text-slate-500 dark:text-gray-400">{repositoryLabel}</p>
      </div>

      <div className="flex shrink-0 items-center gap-2">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button type="button" variant="outline" size="icon-sm" aria-label="Factory settings">
              <Settings2 />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="bottom">Factory settings</TooltipContent>
        </Tooltip>
        <Button type="button" onClick={onNewWork}>
          <Plus />
          New work
        </Button>
      </div>
    </header>
  );
}
