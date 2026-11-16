import type { ChangeEvent } from "react";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import type { useMergeConfidenceSettingsDraft } from "./mergeConfidenceSettingsDraft";

export function MergeConfidencePublicationSettings({
  settings,
}: {
  settings: ReturnType<typeof useMergeConfidenceSettingsDraft>;
}) {
  const { draft, saving } = settings;
  if (!draft?.publishingBase) return null;
  const github = draft.publishingBase.component === "github.updatePullRequestBodySection";
  const missingDestination = draft.publishResults && !draft.resultsBody && !draft.resultsComment;
  return (
    <>
      {github ? (
        <div className="flex items-start justify-between gap-6">
          <div className="space-y-1">
            <Label htmlFor="merge-confidence-check">Publish PR check</Label>
            <p id="merge-confidence-check-help" className="workspace-body-text text-muted-foreground">
              Publish assessment status to the pull request. Scores remain advisory.
            </p>
          </div>
          <Switch
            id="merge-confidence-check"
            aria-describedby="merge-confidence-check-help"
            checked={draft.publishPRCheck}
            onCheckedChange={settings.updatePublishPRCheck}
            disabled={saving}
          />
        </div>
      ) : null}
      <div>
        <div className="flex items-start justify-between gap-6">
          <div className="space-y-1">
            <Label htmlFor="merge-confidence-publish">Publish results</Label>
            <p id="merge-confidence-publish-help" className="workspace-body-text text-muted-foreground">
              When off, results stay in SuperPlane. Any existing pull request report stays in place.
            </p>
          </div>
          <Switch
            id="merge-confidence-publish"
            aria-describedby="merge-confidence-publish-help"
            checked={draft.publishResults}
            onCheckedChange={settings.updatePublishResults}
            disabled={saving}
          />
        </div>
        {github && draft.publishResults ? (
          <fieldset
            className="mt-3 flex flex-col gap-2"
            aria-describedby={missingDestination ? "merge-confidence-destination-error" : undefined}
          >
            <legend className="sr-only">Destinations</legend>
            {(
              [
                ["body", "PR body", draft.resultsBody, settings.updateResultsBody],
                ["comment", "PR comment", draft.resultsComment, settings.updateResultsComment],
              ] as const
            ).map(([id, label, checked, onChange]) => (
              <div key={id} className="flex items-center gap-2">
                <Checkbox
                  id={`merge-confidence-${id}`}
                  checked={checked}
                  disabled={saving}
                  onChange={(event: ChangeEvent<HTMLInputElement>) => onChange(event.target.checked)}
                />
                <Label htmlFor={`merge-confidence-${id}`} className="text-sm font-normal">
                  {label}
                </Label>
              </div>
            ))}
            {missingDestination ? (
              <p id="merge-confidence-destination-error" role="alert" className="text-sm text-destructive">
                Select at least one destination.
              </p>
            ) : null}
          </fieldset>
        ) : null}
      </div>
    </>
  );
}
