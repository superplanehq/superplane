import { Link } from "@/components/Link/link";
import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Textarea } from "@/components/ui/textarea";
import { useCreateWorkOrder } from "@/hooks/useFactoryData";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { showErrorToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";
import { ArrowLeft } from "lucide-react";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { workOrderDetailPath, workOrdersPath } from "../lib/factoryPagePaths";
import { WorkOrderAssigneesPopover } from "../WorkOrderAssigneesPopover";
import {
  factoryCardClassName,
  factoryContentBodyClassName,
  factoryPageSubtitleClassName,
  factoryPageTitleClassName,
} from "./factoryPageLayoutStyles";
import { cn } from "@/lib/utils";

const MAX_TITLE_LENGTH = 256;
const MAX_DESCRIPTION_LENGTH = 5000;

export function CreateWorkOrderPage() {
  const { organizationId, factoryId, factory } = useFactoriesLayout();
  const navigate = useNavigate();
  const createWorkOrder = useCreateWorkOrder(organizationId, factoryId);
  const { data: users = [] } = useOrganizationUsers(organizationId);

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [assigneeIds, setAssigneeIds] = useState<string[]>([]);
  const [titleError, setTitleError] = useState("");

  usePageTitle(["New Work Order", factory?.name ?? "Workspace"]);

  const workOrdersHref = workOrdersPath(organizationId, factoryId);

  const selectedAssigneeLabels = useMemo(() => {
    const labelById = new Map(
      users
        .filter((user) => user.metadata?.id)
        .map((user) => [user.metadata!.id!, user.metadata?.email || user.spec?.displayName || user.metadata!.id!]),
    );
    return assigneeIds.map((id) => labelById.get(id)).filter((label): label is string => Boolean(label));
  }, [assigneeIds, users]);

  const handleCreate = async () => {
    const trimmedTitle = title.trim();
    if (!trimmedTitle) {
      setTitleError("Title is required");
      return;
    }

    try {
      const order = await createWorkOrder.mutateAsync({
        title: trimmedTitle,
        description: description.trim(),
        assigneeIds,
      });
      navigate(order.id ? workOrderDetailPath(organizationId, factoryId, order.id) : workOrdersHref);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to create work order"));
    }
  };

  return (
    <div className={factoryContentBodyClassName} data-testid="create-work-order-page">
      <Link
        href={workOrdersHref}
        className="mb-6 inline-flex items-center gap-1.5 text-[13px] text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden />
        Work Orders
      </Link>

      <div className={cn("px-6 py-8 sm:px-8", factoryCardClassName)}>
        <h1 className={factoryPageTitleClassName}>Describe the work</h1>
        <p className={cn("mt-2", factoryPageSubtitleClassName)}>
          Capture one concrete software change. Assign owners and dispatch it to a line when ready.
        </p>

        <div className="mt-8 space-y-6 border-t border-border pt-8">
          <TitleField
            value={title}
            onChange={setTitle}
            errorMessage={titleError}
            onClearError={() => setTitleError("")}
          />

          <DescriptionField value={description} onChange={setDescription} />

          <AssigneesField
            organizationId={organizationId}
            assigneeIds={assigneeIds}
            onChange={setAssigneeIds}
            disabled={createWorkOrder.isPending}
            selectedLabels={selectedAssigneeLabels}
          />

          <div className="flex flex-wrap gap-3 pt-2">
            <LoadingButton
              onClick={() => void handleCreate()}
              disabled={!title.trim()}
              loading={createWorkOrder.isPending}
              loadingText="Creating..."
              data-testid="work-order-create-button"
            >
              <Icon name="plus" />
              Create work order
            </LoadingButton>
            <Button type="button" variant="outline" asChild>
              <Link href={workOrdersHref}>Cancel</Link>
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

interface TitleFieldProps {
  value: string;
  onChange: (next: string) => void;
  errorMessage: string;
  onClearError: () => void;
}

function TitleField({ value, onChange, errorMessage, onClearError }: TitleFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor="work-order-title-input">Title</Label>
      <Input
        id="work-order-title-input"
        data-testid="work-order-title-input"
        value={value}
        onChange={(event) => {
          if (event.target.value.length <= MAX_TITLE_LENGTH) {
            onChange(event.target.value);
          }
          if (errorMessage) {
            onClearError();
          }
        }}
        maxLength={MAX_TITLE_LENGTH}
        placeholder="Add refund reconciliation test"
        autoFocus
      />
      {errorMessage ? <p className="text-[11px] text-destructive">{errorMessage}</p> : null}
    </div>
  );
}

interface DescriptionFieldProps {
  value: string;
  onChange: (next: string) => void;
}

function DescriptionField({ value, onChange }: DescriptionFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor="work-order-description-input">Description</Label>
      <Textarea
        id="work-order-description-input"
        data-testid="work-order-description-input"
        className="field-sizing-fixed min-h-80"
        value={value}
        onChange={(event) => {
          if (event.target.value.length <= MAX_DESCRIPTION_LENGTH) {
            onChange(event.target.value);
          }
        }}
        maxLength={MAX_DESCRIPTION_LENGTH}
        rows={16}
        placeholder="Describe what should change, why it matters, and anything the apps must preserve."
      />
    </div>
  );
}

interface AssigneesFieldProps {
  organizationId: string;
  assigneeIds: string[];
  onChange: (next: string[]) => void;
  disabled: boolean;
  selectedLabels: string[];
}

function AssigneesField({ organizationId, assigneeIds, onChange, disabled, selectedLabels }: AssigneesFieldProps) {
  return (
    <div className="space-y-2">
      <Label>Owners</Label>
      <WorkOrderAssigneesPopover
        organizationId={organizationId}
        selectedIds={assigneeIds}
        onChange={onChange}
        disabled={disabled}
        align="start"
      >
        <Button type="button" variant="outline" data-testid="work-order-assignees-button">
          {assigneeIds.length === 0 ? "Select owners" : `${assigneeIds.length} selected`}
        </Button>
      </WorkOrderAssigneesPopover>
      {selectedLabels.length > 0 ? (
        <p className="text-[13px] text-muted-foreground">{selectedLabels.join(", ")}</p>
      ) : (
        <p className="text-[11px] text-muted-foreground">Optional. Select one or more people to own this work order.</p>
      )}
    </div>
  );
}
