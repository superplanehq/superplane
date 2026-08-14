import { Button } from "@/components/ui/button";
import { showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { ChevronDown, Crosshair } from "lucide-react";

import { OverviewRow } from "../../sidebar/SidebarPrimitives";
import { useMissionAssignment } from "./useMissionAssignment";
import { resolveMissionForWorkOrder } from "./missionListModel";
import { WorkOrderMissionPopover } from "./WorkOrderMissionPopover";

/** Storybook-only Overview row. Assign this work order to one mission, or to none. */
export function WorkOrderMissionOverviewRow({ workOrderId }: { workOrderId: string }) {
  const { missions, missionByWorkOrderId, assignMission } = useMissionAssignment();
  const mission = resolveMissionForWorkOrder(missions, missionByWorkOrderId, workOrderId);

  const handleAssign = async (missionId: string | null) => {
    assignMission(workOrderId, missionId);
    const next = missions.find((item) => item.id === missionId);
    if (next) {
      showSuccessToast(`Assigned to ${next.name}.`);
      return;
    }
    showSuccessToast("This work order has no mission.");
  };

  return (
    <OverviewRow icon={<Crosshair className="size-3.5" aria-hidden />} srLabel="Mission">
      <WorkOrderMissionPopover missions={missions} selectedMissionId={mission?.id ?? null} onAssign={handleAssign}>
        <Button
          type="button"
          variant="ghost"
          aria-label={mission ? `Mission: ${mission.name}` : "Attach to mission"}
          className={cn(
            "-my-1.5 -mr-1.5 h-auto w-full min-w-0 justify-start gap-1.5 whitespace-normal rounded-md py-1.5 pr-1.5 pl-0 text-left text-[13px] tracking-[-0.01em] hover:bg-accent/60 focus-visible:bg-accent/60",
            mission ? "text-foreground" : "text-muted-foreground hover:text-foreground",
          )}
          data-testid="work-order-edit-mission"
        >
          <span className="min-w-0 truncate">{mission ? mission.name : "Attach to mission"}</span>
          <ChevronDown className="ml-auto size-3.5 shrink-0 text-muted-foreground" aria-hidden />
        </Button>
      </WorkOrderMissionPopover>
    </OverviewRow>
  );
}
