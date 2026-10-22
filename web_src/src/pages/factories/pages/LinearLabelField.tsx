import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { Plus } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { addIntakeLabel, toggleIntakeLabel } from "./intakeSourceSettingsModel";
import { LINEAR_INTAKE_SETUP_COPY } from "./linearIntakeSetupCopy";

export function LinearLabelField({
  labels,
  onChange,
  organizationId = "",
  integrationId = "",
  teamId = "",
}: {
  labels: string[];
  onChange: (labels: string[]) => void;
  organizationId?: string;
  integrationId?: string;
  /** Shows the labels of this team. An intake without a team shows the workspace labels. */
  teamId?: string;
}) {
  const labelScope: Record<string, string> = teamId ? { team: teamId } : { scope: "workspace" };
  const labelsQuery = useIntegrationResources(organizationId, integrationId, "label", labelScope);
  const choices = [
    ...new Set([...(labelsQuery.data ?? []).map((resource) => resource.name ?? "").filter(Boolean), ...labels]),
  ];
  const [draft, setDraft] = useState("");
  const [typing, setTyping] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (typing) {
      inputRef.current?.focus();
    }
  }, [typing]);

  function close() {
    setDraft("");
    setTyping(false);
  }

  function addDraft() {
    onChange(addIntakeLabel(labels, draft));
    close();
  }

  return (
    <div className="flex flex-col gap-2" data-testid="linear-intake-label-options">
      {labelsQuery.isLoading ? (
        <p className="workspace-body-text text-muted-foreground">{LINEAR_INTAKE_SETUP_COPY.labelsLoading}</p>
      ) : null}
      {labelsQuery.isError ? (
        <div className="space-y-2">
          <p className="workspace-body-text text-destructive">{LINEAR_INTAKE_SETUP_COPY.labelsError}</p>
          <Button type="button" variant="outline" size="sm" onClick={() => void labelsQuery.refetch()}>
            {LINEAR_INTAKE_SETUP_COPY.wizardRetry}
          </Button>
        </div>
      ) : null}
      {!labelsQuery.isLoading && !labelsQuery.isError && choices.length === 0 ? (
        <p className="workspace-body-text text-muted-foreground">{LINEAR_INTAKE_SETUP_COPY.labelsEmpty}</p>
      ) : null}
      {choices.length > 0 ? (
        <ul className="flex flex-wrap items-center gap-1.5">
          {choices.map((label) => (
            <li key={label}>
              <label
                className={cn(
                  "inline-flex max-w-full cursor-pointer items-center gap-2 rounded-md border px-2 py-1 text-[13px]",
                  labels.includes(label)
                    ? "border-foreground/20 bg-accent/50 text-foreground"
                    : "border-border bg-card text-muted-foreground hover:border-foreground/15",
                )}
              >
                <Checkbox
                  checked={labels.includes(label)}
                  onChange={() => onChange(toggleIntakeLabel(labels, label))}
                  aria-label={label}
                />
                <span className="min-w-0 truncate">{label}</span>
              </label>
            </li>
          ))}
        </ul>
      ) : null}
      {typing ? (
        <div className="flex items-center gap-2">
          <Input
            ref={inputRef}
            value={draft}
            aria-label={LINEAR_INTAKE_SETUP_COPY.labelInput}
            placeholder={LINEAR_INTAKE_SETUP_COPY.labelPlaceholder}
            onChange={(event) => setDraft(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Escape") {
                event.preventDefault();
                close();
                return;
              }
              if (event.key !== "Enter") {
                return;
              }
              event.preventDefault();
              addDraft();
            }}
            data-testid="linear-intake-label-input"
          />
          <Button
            type="button"
            variant="outline"
            className="shrink-0"
            disabled={draft.trim().length === 0}
            onClick={() => addDraft()}
          >
            {LINEAR_INTAKE_SETUP_COPY.labelAdd}
          </Button>
          <Button type="button" variant="ghost" className="shrink-0" onClick={() => close()}>
            {LINEAR_INTAKE_SETUP_COPY.labelCancel}
          </Button>
        </div>
      ) : (
        <div>
          <Button type="button" variant="outline" size="sm" onClick={() => setTyping(true)}>
            <Plus className="size-3.5" aria-hidden />
            {LINEAR_INTAKE_SETUP_COPY.labelNew}
          </Button>
        </div>
      )}
    </div>
  );
}
