import { MultiCombobox, MultiComboboxLabel } from "@/components/MultiCombobox/multi-combobox";
import { Button } from "@/components/ui/button";
import { Loader2 } from "lucide-react";

import { LINEAR_INTAKE_SETUP_COPY } from "./linearIntakeSetupCopy";

export function LinearProjectPicker({
  projects,
  selectedIds,
  loading,
  error,
  onChange,
  onRetry,
}: {
  projects: Array<{ id?: string; name?: string }>;
  selectedIds: string[];
  loading: boolean;
  error: boolean;
  onChange: (ids: string[]) => void;
  onRetry: () => void;
}) {
  if (loading) {
    return (
      <p className="workspace-body-text flex items-center gap-2 text-muted-foreground">
        <Loader2 className="size-4 animate-spin" aria-hidden />
        {LINEAR_INTAKE_SETUP_COPY.wizardProjectsLoading}
      </p>
    );
  }
  if (error) {
    return (
      <div className="space-y-3">
        <p className="workspace-body-text text-destructive">{LINEAR_INTAKE_SETUP_COPY.wizardProjectsError}</p>
        <Button type="button" variant="outline" size="sm" onClick={onRetry}>
          {LINEAR_INTAKE_SETUP_COPY.wizardRetry}
        </Button>
      </div>
    );
  }

  const projectNames = new Map(projects.map((project) => [project.id ?? "", project.name || "Untitled project"]));
  const options = projects.flatMap((project) =>
    project.id ? [{ id: project.id, name: project.name || "Untitled project" }] : [],
  );
  const value = selectedIds.map((id) => ({ id, name: projectNames.get(id) ?? id }));

  if (options.length === 0 && value.length === 0) {
    return <p className="workspace-body-text text-muted-foreground">{LINEAR_INTAKE_SETUP_COPY.wizardProjectsEmpty}</p>;
  }

  return (
    <div data-testid="linear-project-picker">
      <MultiCombobox
        aria-label={LINEAR_INTAKE_SETUP_COPY.wizardSearchProjects}
        placeholder={LINEAR_INTAKE_SETUP_COPY.wizardSearchProjects}
        options={options}
        value={value}
        displayValue={(project) => project.name}
        filter={(project, query) => project.name.toLowerCase().includes(query.trim().toLowerCase())}
        onChange={(selected) => onChange(selected.map((project) => project.id))}
      >
        {(project) => <MultiComboboxLabel>{project.name}</MultiComboboxLabel>}
      </MultiCombobox>
    </div>
  );
}
