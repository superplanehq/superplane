import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { useEffect, useState, type ReactNode } from "react";

import type { FactoryMission } from "./missionMocks";

const NO_MISSION_VALUE = "__none__";

interface WorkOrderMissionPopoverProps {
  missions: FactoryMission[];
  selectedMissionId: string | null;
  align?: "start" | "center" | "end";
  onAssign: (missionId: string | null) => Promise<void>;
  children: ReactNode;
}

/** Storybook-only picker. Matches the factory-line dispatch popover. */
export function WorkOrderMissionPopover({
  missions,
  selectedMissionId,
  align = "end",
  onAssign,
  children,
}: WorkOrderMissionPopoverProps) {
  const [open, setOpen] = useState(false);
  const [draftId, setDraftId] = useState(selectedMissionId ?? NO_MISSION_VALUE);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (open) {
      setDraftId(selectedMissionId ?? NO_MISSION_VALUE);
    }
  }, [open, selectedMissionId]);

  const handleOpenChange = (nextOpen: boolean) => {
    if (isSaving) {
      return;
    }
    setOpen(nextOpen);
  };

  const handleAssign = async () => {
    setIsSaving(true);
    try {
      await onAssign(draftId === NO_MISSION_VALUE ? null : draftId);
      setOpen(false);
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Popover open={open} onOpenChange={handleOpenChange} modal={false}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent align={align} className="w-80 p-3" sideOffset={8}>
        <div className="space-y-3">
          {missions.length === 0 ? (
            <p className="text-sm text-muted-foreground">Add a mission before you assign work orders.</p>
          ) : (
            <div className="space-y-1.5">
              <Label htmlFor="work-order-mission-select" className="text-xs">
                Mission
              </Label>
              <Select value={draftId} onValueChange={setDraftId}>
                <SelectTrigger
                  id="work-order-mission-select"
                  className="w-full"
                  data-testid="work-order-mission-select"
                >
                  <SelectValue placeholder="Select a mission" />
                </SelectTrigger>
                <SelectContent position="popper">
                  <SelectItem value={NO_MISSION_VALUE}>No mission</SelectItem>
                  {missions.map((mission) => (
                    <SelectItem key={mission.id} value={mission.id}>
                      {mission.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          <LoadingButton
            onClick={() => void handleAssign()}
            disabled={missions.length === 0}
            loading={isSaving}
            loadingText="Assigning..."
            className="w-full"
            data-testid="work-order-mission-submit"
          >
            Assign
          </LoadingButton>
        </div>
      </PopoverContent>
    </Popover>
  );
}
