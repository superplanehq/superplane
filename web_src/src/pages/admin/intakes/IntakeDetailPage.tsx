import { ArrowLeft } from "lucide-react";
import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router";

import {
  useAddAdminIntakeOrganization,
  useAdminIntakeCatalog,
  useDeleteAdminIntake,
  useRemoveAdminIntakeOrganization,
  useUpdateAdminIntake,
  type UpdateAdminIntakeInput,
} from "@/hooks/useAdminIntakeCatalog";
import { usePageTitle } from "@/hooks/usePageTitle";
import { showErrorToast } from "@/lib/toast";

import { IntakeAccessCard } from "./IntakeAccessCard";
import type { AdminIntakeEntry } from "./intakeCatalogModel";
import { DeleteDialog, DetailCard, EntryMenu, MetadataRail, RenameDialog, StatusNoteEditor } from "./IntakeDetailParts";
import { IntakeIcon, IntakeStatusPill } from "./IntakeStatusPill";
import { MaturityStepper } from "./MaturityStepper";

export function IntakeDetailPage() {
  const { key = "" } = useParams();
  const { data: entries, isLoading, isError } = useAdminIntakeCatalog();
  const entry = entries?.find((item) => item.key === key);
  usePageTitle([entry?.name ?? "Intake", "Intakes", "Installation Admin"]);

  if (isLoading) {
    return <DetailMessage>Loading the intake...</DetailMessage>;
  }
  if (isError) {
    return <DetailMessage>Cannot load the intake. Refresh the page to try again.</DetailMessage>;
  }
  if (!entry) {
    return <DetailMessage>This intake does not exist.</DetailMessage>;
  }
  return <IntakeDetail entry={entry} />;
}

function IntakeDetail({ entry }: { entry: AdminIntakeEntry }) {
  const navigate = useNavigate();
  const update = useUpdateAdminIntake(entry.key);
  const addOrganization = useAddAdminIntakeOrganization(entry.key);
  const removeOrganization = useRemoveAdminIntakeOrganization(entry.key);
  const deleteEntry = useDeleteAdminIntake(entry.key);
  const [noteSavedAt, setNoteSavedAt] = useState<number | null>(null);
  const [renameOpen, setRenameOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);

  const onError = (error: Error) => showErrorToast(error.message);
  const patch = (input: UpdateAdminIntakeInput, onSuccess?: () => void) =>
    update.mutate(input, { onError, onSuccess: () => onSuccess?.() });
  const accessPending = update.isPending || addOrganization.isPending || removeOrganization.isPending;

  return (
    <div className="flex flex-col gap-5">
      <Link
        to="/admin/intakes"
        className="flex w-fit items-center gap-1 text-sm text-slate-500 hover:text-slate-800 dark:text-gray-400 dark:hover:text-gray-100"
      >
        <ArrowLeft size={14} />
        Intakes
      </Link>

      <div className="flex items-center gap-3">
        <IntakeIcon intakeKey={entry.key} name={entry.name} size="lg" />
        <h1 className="text-xl font-semibold text-slate-900 dark:text-gray-100">{entry.name}</h1>
        <code className="rounded bg-slate-200/70 px-1.5 py-0.5 font-mono text-xs text-slate-600 dark:bg-gray-800 dark:text-gray-300">
          {entry.key}
        </code>
        <IntakeStatusPill status={entry.status} />
        <div className="ml-auto">
          <EntryMenu entry={entry} onRename={() => setRenameOpen(true)} onDelete={() => setDeleteOpen(true)} />
        </div>
      </div>

      <div className="grid grid-cols-[1fr_17rem] items-start gap-6">
        <div className="flex flex-col gap-5">
          <DetailCard title="Maturity" description="Move the intake to the next step when it is ready.">
            <MaturityStepper entry={entry} pending={update.isPending} onChangeStatus={(status) => patch({ status })} />
          </DetailCard>
          <DetailCard title="Status note">
            <StatusNoteEditor
              entry={entry}
              saving={update.isPending && update.variables?.status_note !== undefined}
              savedAt={noteSavedAt}
              onSave={(note) => patch({ status_note: note }, () => setNoteSavedAt(Date.now()))}
            />
          </DetailCard>
          <DetailCard title="Access" description="Select the companies that can create this intake.">
            <IntakeAccessCard
              entry={entry}
              pending={accessPending}
              onSetEnabledForAll={(enabled) => patch({ enabled_for_all: enabled })}
              onAddOrganization={(organizationId) => addOrganization.mutate(organizationId, { onError })}
              onRemoveOrganization={(organizationId) => removeOrganization.mutate(organizationId, { onError })}
            />
          </DetailCard>
        </div>
        <MetadataRail entry={entry} disabled={update.isPending} onChangeCategory={(category) => patch({ category })} />
      </div>

      <RenameDialog
        open={renameOpen}
        name={entry.name}
        pending={update.isPending}
        onOpenChange={setRenameOpen}
        onRename={(name) => patch({ name }, () => setRenameOpen(false))}
      />
      <DeleteDialog
        open={deleteOpen}
        name={entry.name}
        pending={deleteEntry.isPending}
        onOpenChange={setDeleteOpen}
        onDelete={() => deleteEntry.mutate(undefined, { onError, onSuccess: () => navigate("/admin/intakes") })}
      />
    </div>
  );
}

function DetailMessage({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-3">
      <Link to="/admin/intakes" className="flex w-fit items-center gap-1 text-sm text-slate-500 hover:text-slate-800">
        <ArrowLeft size={14} />
        Intakes
      </Link>
      <div className="rounded-lg border border-dashed border-slate-300 bg-white px-6 py-10 text-center text-sm text-slate-500 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-400">
        {children}
      </div>
    </div>
  );
}
