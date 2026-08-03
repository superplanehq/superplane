import { Icon } from "@/components/Icon";
import { Link } from "@/components/Link/link";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Textarea } from "@/components/ui/textarea";
import { useCreateWorkOrder, useFactory } from "@/hooks/useFactoryData";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { ArrowLeft } from "lucide-react";
import { useMemo, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import { factoryFormCardClassName, factoryPageContentClassName } from "./factoryPageStyles";
import { FactoryPageShell } from "./FactoryPageShell";
import { WorkOrderAssigneesPopover } from "./WorkOrderAssigneesPopover";

const MAX_TITLE_LENGTH = 256;
const MAX_DESCRIPTION_LENGTH = 5000;

export function CreateWorkOrderPage() {
  const navigate = useNavigate();
  const { organizationId, factoryId } = useParams<{ organizationId: string; factoryId: string }>();
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [assigneeIds, setAssigneeIds] = useState<string[]>([]);
  const [titleError, setTitleError] = useState("");

  const {
    data: factory,
    isLoading: factoryLoading,
    error: factoryError,
  } = useFactory(organizationId ?? "", factoryId ?? "");
  const createWorkOrder = useCreateWorkOrder(organizationId ?? "", factoryId ?? "");
  const { data: users = [] } = useOrganizationUsers(organizationId ?? "");

  const selectedAssigneeLabels = useMemo(() => {
    const labelById = new Map(
      users
        .filter((user) => user.metadata?.id)
        .map((user) => [user.metadata!.id!, user.metadata?.email || user.spec?.displayName || user.metadata!.id!]),
    );

    return assigneeIds.map((id) => labelById.get(id)).filter((label): label is string => Boolean(label));
  }, [assigneeIds, users]);

  usePageTitle(["New Work Order", factory?.name ?? "Factory"]);

  useReportPageReady(!factoryLoading && Boolean(factory), {
    failed: Boolean(factoryError),
  });

  if (!organizationId || !factoryId) {
    return null;
  }

  if (!factoryLoading && factoryError) {
    return <Navigate to={`/${organizationId}/factories`} replace />;
  }

  const factoryHref = `/${organizationId}/factories/${factoryId}`;

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
      if (order.id) {
        navigate(`${factoryHref}/orders/${order.id}`);
      } else {
        navigate(factoryHref);
      }
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to create work order"));
    }
  };

  return (
    <FactoryPageShell backHref={factoryHref} backLabel={factory?.name ?? "Factory"}>
      {factoryLoading ? (
        <div className="px-8 py-6">
          <Text className="text-sm text-gray-500">Loading…</Text>
        </div>
      ) : factory ? (
        <div className={factoryPageContentClassName}>
          <Link
            href={factoryHref}
            className="mb-6 inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-slate-900 dark:text-gray-400 dark:hover:text-gray-100"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden />
            {factory.name}
          </Link>

          <div className={factoryFormCardClassName}>
            <h1 className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-gray-100">
              Describe the work
            </h1>
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
                      setTitle(event.target.value);
                    }
                    if (titleError) {
                      setTitleError("");
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
                      setDescription(event.target.value);
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
                  onChange={setAssigneeIds}
                  disabled={createWorkOrder.isPending}
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
                  <Link href={factoryHref}>Cancel</Link>
                </Button>
              </div>
            </div>
          </div>
        </div>
      ) : null}
    </FactoryPageShell>
  );
}
