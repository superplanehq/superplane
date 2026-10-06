import { Text } from "@/components/Text/text";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import React from "react";

import { BrokerTaskSection } from "./BrokerTaskSection";
import { FleetCapacitySummary, FleetMachineType } from "./FleetCapacitySummary";
import { RunnerList, TaskList } from "./FleetRecordLists";
import { REFRESH_INTERVAL_MS } from "./fleetAdmin";
import { useAdminFleets } from "./useAdminFleets";

const FleetBody = ({ page }: { page: ReturnType<typeof useAdminFleets> }) => {
  if (!page.fleets || page.fleets.length === 0) {
    if (page.fleetLoadFailed) {
      return null;
    }
    return (
      <div className="rounded-xl border border-dashed border-slate-300 bg-white p-8 text-center shadow-sm dark:border-gray-700 dark:bg-gray-900">
        <Text className="text-sm text-gray-600 dark:text-gray-400">No machine types are configured.</Text>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <FleetMachineType
        fleets={page.fleets}
        selectedId={page.selectedFleetId ?? undefined}
        onChange={page.chooseFleet}
      />
      {page.capacity ? (
        <FleetCapacitySummary capacity={page.capacity} />
      ) : (
        <Text className="text-sm text-gray-500 dark:text-gray-400">Loading fleet capacity...</Text>
      )}
      <Tabs defaultValue="runners">
        <TabsList>
          <TabsTrigger value="runners">Runners</TabsTrigger>
          <TabsTrigger value="tasks">Tasks</TabsTrigger>
        </TabsList>
        <TabsContent value="runners" className="mt-3">
          <RunnerList runners={page.runners} cursors={page.runnerCursors} setCursors={page.setRunnerCursors} />
        </TabsContent>
        <TabsContent value="tasks" className="mt-3">
          <TaskList tasks={page.tasks} cursors={page.taskCursors} setCursors={page.setTaskCursors} />
        </TabsContent>
      </Tabs>
    </div>
  );
};

const RunnerTasks: React.FC = () => {
  const page = useAdminFleets();
  useReportPageReady(page.fleets !== null && page.brokerLoaded);

  if (page.fleets === null) {
    return (
      <div className="flex flex-col items-center space-y-4 py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-b border-gray-500 dark:border-gray-400"></div>
        <Text className="text-gray-500 dark:text-gray-400">Loading fleets...</Text>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">Fleets</h1>
        <Text className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Installation fleet capacity, runners, and tasks. Refreshes every {REFRESH_INTERVAL_MS / 1000} seconds.
        </Text>
      </div>
      <FleetBody page={page} />
      <BrokerTaskSection configured={page.configured} tasks={page.brokerTasks} />
    </div>
  );
};

export default RunnerTasks;
