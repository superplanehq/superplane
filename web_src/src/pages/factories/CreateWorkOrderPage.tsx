import { Text } from "@/components/Text/text";
import { useParams } from "react-router-dom";
import { CreateWorkOrderForm } from "./CreateWorkOrderForm";
import { FactoryPageShell } from "./FactoryPageShell";
import { useCreateWorkOrderPage } from "./useCreateWorkOrderPage";

export function CreateWorkOrderPage() {
  const { organizationId, factoryId } = useParams<{ organizationId: string; factoryId: string }>();

  if (!organizationId || !factoryId) {
    return null;
  }

  return <CreateWorkOrderPageContent organizationId={organizationId} factoryId={factoryId} />;
}

function CreateWorkOrderPageContent({ organizationId, factoryId }: { organizationId: string; factoryId: string }) {
  const page = useCreateWorkOrderPage(organizationId, factoryId);

  if (page.kind === "redirect") {
    return page.element;
  }

  return (
    <FactoryPageShell backHref={page.factoryHref} backLabel={page.factory?.name ?? "Factory"}>
      {page.factoryLoading ? (
        <div className="px-8 py-6">
          <Text className="text-sm text-gray-500">Loading…</Text>
        </div>
      ) : page.factory ? (
        <CreateWorkOrderForm
          organizationId={page.organizationId}
          factoryHref={page.factoryHref}
          factoryName={page.factory.name ?? "Factory"}
          title={page.title}
          description={page.description}
          assigneeIds={page.assigneeIds}
          titleError={page.titleError}
          selectedAssigneeLabels={page.selectedAssigneeLabels}
          isCreating={page.isCreating}
          onTitleChange={page.setTitle}
          onDescriptionChange={page.setDescription}
          onAssigneeIdsChange={page.setAssigneeIds}
          onClearTitleError={page.clearTitleError}
          onCreate={page.handleCreate}
        />
      ) : null}
    </FactoryPageShell>
  );
}
