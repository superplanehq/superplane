import type { FactoriesFactoryLine } from "@/api-client";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { cn } from "@/lib/utils";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { Layers, User } from "lucide-react";
import { useEffect, useRef, useState, type ReactNode } from "react";

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
  const rootRef = useRef<HTMLDivElement>(null);
  const hasLines = lines.length > 0;

  useEffect(() => {
    if (openPicker === "assignee") {
      setDraftAssigneeIds(assigneeIds);
    }
  }, [assigneeIds, openPicker]);

  useEffect(() => {
    if (!openPicker) {
      return;
    }

    const handlePointerDown = (event: MouseEvent) => {
      if (!(event.target instanceof Node)) {
        return;
      }
      if (rootRef.current?.contains(event.target)) {
        return;
      }
      setOpenPicker(null);
    };

    document.addEventListener("mousedown", handlePointerDown);
    return () => document.removeEventListener("mousedown", handlePointerDown);
  }, [openPicker]);

  const togglePicker = (picker: Exclude<OpenPicker, null>) => {
    if (isSaving) {
      return;
    }
    setOpenPicker((current) => (current === picker ? null : picker));
  };

  const handleSaveAssignees = async () => {
    onAssigneeChange(draftAssigneeIds);
    setOpenPicker(null);
  };

  const handleLineChange = (lineName: string) => {
    onLineSelect(lineName);
    setOpenPicker(null);
  };

  return (
    <div ref={rootRef} className="relative flex min-w-0 flex-wrap items-center gap-1.5">
      <PropertyPill disabled={isSaving} testId="work-order-assignees-button" onClick={() => togglePicker("assignee")}>
        {assigneeIds.length === 0 ? (
          <>
            <User className="size-3.5" aria-hidden />
            Assignee
          </>
        ) : (
          <AssigneePillBody assigneeIds={assigneeIds} resolveUser={resolveUser} />
        )}
      </PropertyPill>

      {hasLines ? (
        <PropertyPill disabled={isSaving} testId="work-order-line-button" onClick={() => togglePicker("line")}>
          <Layers className="size-3.5" aria-hidden />
          {selectedLineName || "Line"}
        </PropertyPill>
      ) : (
        <PropertyPill disabled testId="work-order-line-button">
          <Layers className="size-3.5" aria-hidden />
          Line required
        </PropertyPill>
      )}

      {openPicker === "assignee" ? (
        <AssigneePickerPanel
          organizationId={organizationId}
          selectedIds={draftAssigneeIds}
          isSaving={isSaving}
          onChange={setDraftAssigneeIds}
          onSave={() => void handleSaveAssignees()}
        />
      ) : null}

      {openPicker === "line" ? (
        <LinePickerPanel
          lines={lines}
          selectedLineName={selectedLineName}
          isSaving={isSaving}
          onSelect={handleLineChange}
        />
      ) : null}
    </div>
  );
}

function AssigneePickerPanel({
  organizationId,
  selectedIds,
  isSaving,
  onChange,
  onSave,
}: {
  organizationId: string;
  selectedIds: string[];
  isSaving: boolean;
  onChange: (ids: string[]) => void;
  onSave: () => void;
}) {
  return (
    <div
      className="absolute bottom-full left-0 z-20 mb-2 w-72 rounded-md border border-border bg-popover p-3 text-popover-foreground shadow-md"
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
    </div>
  );
}

function LinePickerPanel({
  lines,
  selectedLineName,
  isSaving,
  onSelect,
}: {
  lines: FactoriesFactoryLine[];
  selectedLineName: string;
  isSaving: boolean;
  onSelect: (lineName: string) => void;
}) {
  return (
    <div
      className="absolute bottom-full left-0 z-20 mb-2 w-72 rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-md"
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
    </div>
  );
}

function PropertyPill({
  children,
  disabled,
  testId,
  onClick,
}: {
  children: ReactNode;
  disabled?: boolean;
  testId?: string;
  onClick?: () => void;
}) {
  return (
    <Button
      type="button"
      variant="outline"
      disabled={disabled}
      data-testid={testId}
      onClick={onClick}
      className="h-7 gap-1.5 rounded-md px-2 text-[12px] font-normal text-muted-foreground"
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
