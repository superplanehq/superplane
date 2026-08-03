import { Icon } from "@/components/Icon";
import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Textarea } from "@/components/ui/textarea";
import { ArrowLeft } from "lucide-react";
import { factoryFormCardClassName, factoryPageContentClassName } from "./factoryPageStyles";
import { WorkOrderAssigneesPopover } from "./WorkOrderAssigneesPopover";

const MAX_TITLE_LENGTH = 256;
const MAX_DESCRIPTION_LENGTH = 5000;

interface CreateWorkOrderFormProps {
  organizationId: string;
  factoryHref: string;
  factoryName: string;
  title: string;
  description: string;
  assigneeIds: string[];
  titleError: string;
  selectedAssigneeLabels: string[];
  isCreating: boolean;
  onTitleChange: (value: string) => void;
  onDescriptionChange: (value: string) => void;
  onAssigneeIdsChange: (ids: string[]) => void;
  onClearTitleError: () => void;
  onCreate: () => void;
}

export function CreateWorkOrderForm({
  organizationId,
  factoryHref,
  factoryName,
  title,
  description,
  assigneeIds,
  titleError,
  selectedAssigneeLabels,
  isCreating,
  onTitleChange,
  onDescriptionChange,
  onAssigneeIdsChange,
  onClearTitleError,
  onCreate,
}: CreateWorkOrderFormProps) {
  return (
    <div className={factoryPageContentClassName}>
      <Link
        href={factoryHref}
        className="mb-6 inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-slate-900 dark:text-gray-400 dark:hover:text-gray-100"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden />
        {factoryName}
      </Link>

      <div className={factoryFormCardClassName}>
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-gray-100">Describe the work</h1>
        <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
          Capture one concrete software change. Assign owners and dispatch it to a line when ready.
        </p>

        <div className="mt-8 space-y-6 border-t border-slate-200 pt-8 dark:border-gray-700/70">
          <div className="space-y-2">
            <Label htmlFor="work-order-title-input">Title</Label>
            <Input
              id="work-order-title-input"
              data-testid="work-order-title-input"
              value={title}
              onChange={(event) => {
                if (event.target.value.length <= MAX_TITLE_LENGTH) {
                  onTitleChange(event.target.value);
                }
                if (titleError) {
                  onClearTitleError();
                }
              }}
              maxLength={MAX_TITLE_LENGTH}
              placeholder="Add refund reconciliation test"
              autoFocus
            />
            {titleError ? <p className="text-xs text-red-600">{titleError}</p> : null}
          </div>

          <div className="space-y-2">
            <Label htmlFor="work-order-description-input">Description</Label>
            <Textarea
              id="work-order-description-input"
              data-testid="work-order-description-input"
              className="field-sizing-fixed min-h-80"
              value={description}
              onChange={(event) => {
                if (event.target.value.length <= MAX_DESCRIPTION_LENGTH) {
                  onDescriptionChange(event.target.value);
                }
              }}
              maxLength={MAX_DESCRIPTION_LENGTH}
              rows={16}
              placeholder="Describe what should change, why it matters, and anything the apps must preserve."
            />
          </div>

          <div className="space-y-2">
            <Label>Assignees</Label>
            <WorkOrderAssigneesPopover
              organizationId={organizationId}
              selectedIds={assigneeIds}
              onChange={onAssigneeIdsChange}
              disabled={isCreating}
              align="start"
            >
              <Button type="button" variant="outline" data-testid="work-order-assignees-button">
                {assigneeIds.length === 0 ? "Select assignees" : `${assigneeIds.length} selected`}
              </Button>
            </WorkOrderAssigneesPopover>
            {selectedAssigneeLabels.length > 0 ? (
              <p className="text-sm text-gray-600 dark:text-gray-400">{selectedAssigneeLabels.join(", ")}</p>
            ) : (
              <p className="text-xs text-gray-500 dark:text-gray-400">
                Optional. Select one or more people to own this work order.
              </p>
            )}
          </div>

          <div className="flex flex-wrap gap-3 pt-2">
            <LoadingButton
              onClick={() => void onCreate()}
              disabled={!title.trim()}
              loading={isCreating}
              loadingText="Creating..."
              data-testid="work-order-create-button"
            >
              <Icon name="plus" />
              Create work order
            </LoadingButton>
            <Button type="button" variant="outline" asChild>
              <Link href={factoryHref}>Cancel</Link>
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
