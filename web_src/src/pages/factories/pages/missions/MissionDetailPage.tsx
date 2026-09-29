import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import { useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { ChartNoAxesColumn, CircleDot, User } from "lucide-react";
import { useMemo } from "react";
import { Link, Navigate, useParams } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { workOrdersPath } from "../../lib/factoryPagePaths";
import { applyWorkOrderOrdering, buildWorkOrderListEntries } from "../../lib/workOrderListModel";
import { WorkOrdersErrorState } from "../../workOrders/WorkOrdersEmptyStates";
import { WorkOrderDescription } from "../../WorkOrderDescription";
import { factoryContentBodyClassName } from "../factoryPageLayoutStyles";
import { OverviewRow, SidebarSectionHeading } from "../../sidebar/SidebarPrimitives";
import { useOptionalMissionAssignment } from "./useMissionAssignment";
import {
  applyMissionFilter,
  findMissionById,
  resolveMissionDisplayStatus,
  resolveMissionProgress,
  resolveMissionStatus,
} from "./missionListModel";
import { SEEDED_MISSIONS, missionByWorkOrderId as seededAssignments, type FactoryMission } from "./missionMocks";
import { MissionAssignee } from "./MissionAssignee";
import { MissionDetailHeader } from "./MissionDetailHeader";
import { MissionStatusValue } from "./MissionStatusValue";
import { MissionWorkOrderList } from "./MissionWorkOrderList";

/** Storybook-only mission detail. Open one mission and see its tasks. */
export function MissionDetailPage() {
  const { missionId } = useParams<{ missionId: string }>();
  const { organizationId, factoryId, factoryKey, factory } = useFactoriesLayout();
  const assignment = useOptionalMissionAssignment();
  const missions = assignment?.missions ?? SEEDED_MISSIONS;
  const {
    data: workOrders = [],
    isLoading,
    error: workOrdersError,
    refetch,
  } = useFactoryWorkOrders(organizationId, factoryId);
  const listHref = workOrdersPath(organizationId, factoryKey);

  if (!missionId) {
    return <Navigate to={listHref} replace />;
  }

  const mission = findMissionById(missions, missionId);
  if (!mission) {
    return (
      <div className={factoryContentBodyClassName} data-testid="mission-detail-not-found">
        <p className="text-[14px] font-medium text-foreground">Mission not found</p>
        <p className="mt-1 text-[13px] text-muted-foreground">This mission is not in the Storybook seed data.</p>
        <Link to={listHref} className="mt-3 inline-flex text-[13px] font-medium text-foreground underline">
          Back to tasks
        </Link>
      </div>
    );
  }

  if (workOrdersError) {
    return (
      <div className={factoryContentBodyClassName} data-testid="mission-detail-error">
        <WorkOrdersErrorState onRetry={() => void refetch()} />
      </div>
    );
  }

  if (isLoading || !factory) {
    return (
      <div className={factoryContentBodyClassName}>
        <p className="text-[13px] text-muted-foreground">Loading mission…</p>
      </div>
    );
  }

  return (
    <MissionDetailLoadedView
      organizationId={organizationId}
      factoryKey={factoryKey}
      factory={factory}
      mission={mission}
      workOrders={workOrders}
    />
  );
}

function MissionDetailLoadedView({
  organizationId,
  factoryKey,
  factory,
  mission,
  workOrders,
}: {
  organizationId: string;
  factoryKey: string;
  factory: FactoriesFactory;
  mission: FactoryMission;
  workOrders: FactoriesWorkOrder[];
}) {
  const assignment = useOptionalMissionAssignment();
  const missionByWorkOrderId = assignment?.missionByWorkOrderId ?? seededAssignments;
  const closedByMissionId = assignment?.closedByMissionId ?? {};
  const allEntries = useMemo(() => buildWorkOrderListEntries(workOrders, factory), [workOrders, factory]);
  const entries = useMemo(() => {
    const inMission = applyMissionFilter(allEntries, mission.id, missionByWorkOrderId);
    return applyWorkOrderOrdering(inMission, "updated");
  }, [allEntries, mission.id, missionByWorkOrderId]);
  const progress = resolveMissionProgress(allEntries, mission.id, missionByWorkOrderId);
  const status = resolveMissionDisplayStatus(
    resolveMissionStatus(allEntries, mission.id, missionByWorkOrderId),
    closedByMissionId[mission.id],
  );

  return (
    <div className={factoryContentBodyClassName} data-testid="mission-detail">
      <div className="grid gap-x-[var(--workspace-column-gap)] gap-y-0 lg:grid-cols-[minmax(0,1fr)_var(--workspace-detail-sidebar-width)]">
        <MissionDetailHeader
          missionName={mission.name}
          status={status}
          isManuallyClosed={Boolean(closedByMissionId[mission.id])}
          onClose={(reason) => assignment?.closeMission(mission.id, reason)}
          onReopen={() => assignment?.reopenMission(mission.id)}
        />

        <div className="min-w-0">
          {mission.description ? <WorkOrderDescription description={mission.description} /> : null}

          <section className={mission.description ? "mt-10" : "mt-8"} aria-label="Tasks in this mission">
            <h2 className="workspace-section-label mb-1">Tasks</h2>
            {entries.length === 0 ? (
              <p className="py-2 text-[13px] text-muted-foreground">This mission has no tasks.</p>
            ) : (
              <MissionWorkOrderList entries={entries} organizationId={organizationId} factoryKey={factoryKey} />
            )}
          </section>
        </div>

        <aside className="mt-6 flex flex-col gap-6 lg:mt-0" data-testid="mission-detail-sidebar">
          <section>
            <SidebarSectionHeading>Overview</SidebarSectionHeading>
            <div className="mt-2">
              <OverviewRow icon={<CircleDot className="size-3.5" aria-hidden />} srLabel="Status">
                <MissionStatusValue status={status} variant="plain" />
              </OverviewRow>
              <OverviewRow icon={<User className="size-3.5" aria-hidden />} srLabel="Assignee">
                <MissionAssignee organizationId={organizationId} assignee={mission.assignee} />
              </OverviewRow>
              <OverviewRow icon={<ChartNoAxesColumn className="size-3.5" aria-hidden />} srLabel="Progress">
                <div className="flex min-w-0 flex-col gap-1.5">
                  <span>{progress.label}</span>
                  <div className="h-1 overflow-hidden rounded-full bg-muted" aria-hidden>
                    <div className="h-full rounded-full bg-foreground/70" style={{ width: `${progress.percent}%` }} />
                  </div>
                </div>
              </OverviewRow>
            </div>
          </section>
        </aside>
      </div>
    </div>
  );
}
