import { useState } from "react";

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useAdminIntakePreview } from "@/hooks/useAdminIntakeCatalog";
import { intakeSurfaceState, type IntakeSurfaceState } from "@/lib/intakeCatalog";
import { INTAKE_SURFACE_ORDER, intakeSurfaces, type IntakeSurface } from "@/lib/intakePresentation";
import { TemplateCard } from "@/pages/factories/pages/AddIntakePicker";
import { addIntakeTemplatesFromCatalog } from "@/pages/factories/pages/lineIntakeModel";
import { FirstRunTicketRow } from "@/pages/factories/pages/onboarding/first-run/FirstRunTicketsScreen";
import {
  ComingSoonProviderCard,
  ConnectOptionRow,
  IntegrationChoiceIcon,
} from "@/pages/factories/pages/onboarding/onboardingSteps";

import { INTAKE_SURFACE_LABELS, previewNotShownReason, type AdminIntakeEntry } from "./intakeCatalogModel";

const NO_ACCESS = "no-access";

export function IntakePreviewCard({ entry }: { entry: AdminIntakeEntry }) {
  const [viewer, setViewer] = useState(NO_ACCESS);
  const organizationId = entry.organizations.some((organization) => organization.id === viewer) ? viewer : "";
  const preview = useAdminIntakePreview(entry.key, organizationId);
  const listedSurfaces = intakeSurfaces(entry.key, entry.category);
  const state = preview.data
    ? intakeSurfaceState({ status: preview.data.status, available: preview.data.available })
    : null;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <span className="text-sm text-slate-600 dark:text-gray-300">View as</span>
        <Select value={organizationId || NO_ACCESS} onValueChange={setViewer}>
          <SelectTrigger aria-label="View as" className="h-8 w-64">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={NO_ACCESS}>A company without access</SelectItem>
            {entry.organizations.map((organization) => (
              <SelectItem key={organization.id} value={organization.id}>
                {organization.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {preview.isError ? (
        <p className="text-sm text-red-600 dark:text-red-400">
          Cannot load the preview. Refresh the page to try again.
        </p>
      ) : null}
      {state ? (
        <div className="grid gap-3">
          {INTAKE_SURFACE_ORDER.map((surface) => (
            <SurfacePreview
              key={surface}
              surface={surface}
              entry={entry}
              listed={listedSurfaces.includes(surface)}
              state={state}
            />
          ))}
        </div>
      ) : null}
    </div>
  );
}

interface SurfacePreviewProps {
  surface: IntakeSurface;
  entry: AdminIntakeEntry;
  listed: boolean;
  state: IntakeSurfaceState;
}

function SurfacePreview({ surface, entry, listed, state }: SurfacePreviewProps) {
  const shown = listed && state !== "hidden";
  return (
    <section
      aria-label={INTAKE_SURFACE_LABELS[surface]}
      data-testid={`intake-preview-${surface}`}
      className="rounded-md border border-slate-200 p-3 dark:border-gray-700"
    >
      <p className="pb-2 text-[11px] font-semibold tracking-wide text-slate-400 uppercase dark:text-gray-500">
        {INTAKE_SURFACE_LABELS[surface]}
      </p>
      {shown ? (
        <div inert className="pointer-events-none max-w-sm">
          <SurfaceRow surface={surface} entry={entry} state={state} />
        </div>
      ) : (
        <p className="text-sm text-slate-500 dark:text-gray-400">
          {previewNotShownReason(surface, listed, entry.status)}
        </p>
      )}
    </section>
  );
}

function SurfaceRow({
  surface,
  entry,
  state,
}: {
  surface: IntakeSurface;
  entry: AdminIntakeEntry;
  state: IntakeSurfaceState;
}) {
  const item = { key: entry.key, name: entry.name, category: entry.category, status: entry.status };

  if (surface === "addIntake") {
    const template = addIntakeTemplatesFromCatalog([
      { ...item, available: state === "available" || state === "beta" },
    ]).find((candidate) => candidate.id === entry.key);
    return template ? (
      <ul>
        <TemplateCard template={template} taken={false} />
      </ul>
    ) : null;
  }

  if (surface === "onboardingTickets") {
    return (
      <FirstRunTicketRow
        intakeKey={entry.key}
        entry={{ key: entry.key, name: entry.name, description: "", state: state === "hidden" ? "soon" : state }}
        state={state}
        intakesLoading={false}
        ticketSource={null}
        saving={false}
        jiraConnected={false}
        onSelectTicketSource={() => undefined}
      />
    );
  }

  if (entry.key === "github" && (state === "available" || state === "beta")) {
    return (
      <ConnectOptionRow
        icon={<IntegrationChoiceIcon name="github" />}
        title="GitHub"
        detail="Connect GitHub to list repositories and open pull requests."
        meta={state === "beta" ? "Beta" : undefined}
        onSelect={() => undefined}
      />
    );
  }
  return <ComingSoonProviderCard provider={{ key: entry.key, name: entry.name }} />;
}
