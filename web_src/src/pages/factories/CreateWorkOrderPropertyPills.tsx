import type { FactoriesFactoryLine } from "@/api-client";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { cn } from "@/lib/utils";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { Layers, User } from "lucide-react";
import { useEffect, useState, type ComponentProps, type ReactNode } from "react";

import { OrgUserReference } from "./OrgUserReference";
import { WorkOrderAssigneePicker } from "./WorkOrderAssigneePicker";

type OpenPicker = "assignee" | "line" | null;

interface CreateWorkOrderPropertyPillsProps {
  organizationId: string;
  assigneeIds: string[];
  lines: FactoriesFactoryLine[];
  selectedLineName: string;
  isSaving: boolean;
  onAssigneeChange: (ids: string[]) => void;
  onLineSelect: (lineName: string) => void;
}

export function CreateWorkOrderPropertyPills({
  organizationId,
  assigneeIds,
  lines,
  selectedLineName,
  isSaving,
  onAssigneeChange,
  onLineSelect,
}: CreateWorkOrderPropertyPillsProps) {
  const { resolveUser } = useOrgUserLookup(organizationId);
  const [openPicker, setOpenPicker] = useState<OpenPicker>(null);
  const [draftAssigneeIds, setDraftAssigneeIds] = useState(assigneeIds);
  const [portalRoot, setPortalRoot] = useState<HTMLDivElement | null>(null);
  const hasLines = lines.length > 0;

  useEffect(() => {
    if (openPicker === "assignee") {
      setDraftAssigneeIds(assigneeIds);
    }
  }, [assigneeIds, openPicker]);

  const handlePickerOpenChange = (picker: Exclude<OpenPicker, null>) => (nextOpen: boolean) => {
    if (isSaving) {
      return;
    }
    setOpenPicker(nextOpen ? picker : null);
  };

  const handleSaveAssignees = () => {
    onAssigneeChange(draftAssigneeIds);
    setOpenPicker(null);
  };

  const handleLineChange = (lineName: string) => {
    onLineSelect(lineName);
    setOpenPicker(null);
  };

  return (
    <div ref={setPortalRoot} className="relative flex min-w-0 flex-wrap items-center gap-1.5">
      <Popover modal={false} open={openPicker === "assignee"} onOpenChange={handlePickerOpenChange("assignee")}>
        <PopoverTrigger asChild>
          <PropertyPill disabled={isSaving} testId="work-order-assignees-button">
            {assigneeIds.length === 0 ? (
              <>
                <User className="size-3.5" aria-hidden />
                Assignee
              </>
            ) : (
              <AssigneePillBody assigneeIds={assigneeIds} resolveUser={resolveUser} />
            )}
          </PropertyPill>
        </PopoverTrigger>
        <AssigneePickerPanel
          organizationId={organizationId}
          selectedIds={draftAssigneeIds}
          isSaving={isSaving}
          portalRoot={portalRoot}
          onChange={setDraftAssigneeIds}
          onSave={handleSaveAssignees}
        />
      </Popover>

      {hasLines ? (
        <Popover modal={false} open={openPicker === "line"} onOpenChange={handlePickerOpenChange("line")}>
          <PopoverTrigger asChild>
            <PropertyPill disabled={isSaving} testId="work-order-line-button">
              <Layers className="size-3.5" aria-hidden />
              {selectedLineName || "Line"}
            </PropertyPill>
          </PopoverTrigger>
          <LinePickerPanel
            lines={lines}
            selectedLineName={selectedLineName}
            isSaving={isSaving}
            portalRoot={portalRoot}
            onSelect={handleLineChange}
          />
        </Popover>
      ) : (
        <PropertyPill disabled testId="work-order-line-button">
          <Layers className="size-3.5" aria-hidden />
          Line required
        </PropertyPill>
      )}
    </div>
  );
}

function AssigneePickerPanel({
  organizationId,
  selectedIds,
  isSaving,
  portalRoot,
  onChange,
  onSave,
}: {
  organizationId: string;
  selectedIds: string[];
  isSaving: boolean;
  portalRoot: HTMLElement | null;
  onChange: (ids: string[]) => void;
  onSave: () => void;
}) {
  return (
    <PopoverContent
      align="start"
      side="top"
      container={portalRoot}
      className="z-20 w-72 p-3"
      data-testid="work-order-assignee-picker-panel"
    >
      <WorkOrderAssigneePicker
        organizationId={organizationId}
        selectedIds={selectedIds}
        onChange={onChange}
        disabled={isSaving}
        variant="popover"
      />
      <LoadingButton
        type="button"
        onClick={onSave}
        disabled={isSaving}
        loading={isSaving}
        loadingText="Saving..."
        className="mt-3 w-full"
        data-testid="work-order-save-assignees"
      >
        Save
      </LoadingButton>
    </PopoverContent>
  );
}

function LinePickerPanel({
  lines,
  selectedLineName,
  isSaving,
  portalRoot,
  onSelect,
}: {
  lines: FactoriesFactoryLine[];
  selectedLineName: string;
  isSaving: boolean;
  portalRoot: HTMLElement | null;
  onSelect: (lineName: string) => void;
}) {
  return (
    <PopoverContent
      align="start"
      side="top"
      container={portalRoot}
      className="z-20 w-72 p-1"
      data-testid="work-order-line-picker-panel"
    >
      <p className="px-2 py-1.5 text-[11px] font-medium text-muted-foreground">Line</p>
      {lines.map((line) => {
        const name = line.name ?? "";
        if (!name) {
          return null;
        }
        const isSelected = name === selectedLineName;
        return (
          <Button
            key={line.id ?? name}
            type="button"
            variant="ghost"
            disabled={isSaving}
            data-testid={`work-order-line-option-${name}`}
            className={cn("h-8 w-full justify-start px-2 text-[13px] font-normal", isSelected && "bg-accent")}
            onClick={() => onSelect(name)}
          >
            {name}
          </Button>
        );
      })}
    </PopoverContent>
  );
}

function PropertyPill({
  children,
  disabled,
  testId,
  onClick,
  className,
  ...triggerProps
}: {
  children: ReactNode;
  disabled?: boolean;
  testId?: string;
  onClick?: () => void;
} & Omit<ComponentProps<typeof Button>, "children" | "disabled" | "onClick" | "type" | "variant">) {
  return (
    <Button
      type="button"
      variant="outline"
      disabled={disabled}
      data-testid={testId}
      onClick={onClick}
      className={cn("h-7 gap-1.5 rounded-md px-2 text-[12px] font-normal text-muted-foreground", className)}
      {...triggerProps}
    >
      {children}
    </Button>
  );
}

function AssigneePillBody({
  assigneeIds,
  resolveUser,
}: {
  assigneeIds: string[];
  resolveUser: ReturnType<typeof useOrgUserLookup>["resolveUser"];
}) {
  if (assigneeIds.length === 1) {
    return <OrgUserReference display={resolveUser(assigneeIds[0])} size="xs" nameClassName="truncate text-[12px]" />;
  }
  return (
    <>
      <OrgUserReference display={resolveUser(assigneeIds[0])} size="xs" nameClassName="truncate text-[12px]" />
      <span className="shrink-0 text-[12px] text-muted-foreground">+{assigneeIds.length - 1}</span>
    </>
  );
}
