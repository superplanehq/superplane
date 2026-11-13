import type { ChangeEvent } from "react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import type { ReviewFindingsSettings } from "../lib/reviewRequestFindings";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { AgentAdditionalInstructions } from "./AgentAdditionalInstructions";
import { ImplementSettingRow } from "./ImplementSettingRow";
import { PlanningReviewForm } from "./PlanningReviewForm";
import type { ReviewRequestSettingsPopupProps } from "./ReviewRequestSettingsPopup";
import type { useReviewRequestSettingsDraft } from "./useReviewRequestSettingsDraft";

type ConfigurationProps = ReviewRequestSettingsPopupProps & {
  editor: ReturnType<typeof useReviewRequestSettingsDraft>;
};

export function ReviewRequestSettingsConfiguration(props: ConfigurationProps) {
  const { editor } = props;
  return (
    <section
      aria-label="Review request configuration"
      inert={editor.saving || undefined}
      className="min-h-0 flex-1 overflow-y-auto"
    >
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 px-6 py-8 sm:px-10">
        {!editor.enabled ? (
          <p role="status" className="text-sm text-muted-foreground">
            Review request is paused. New reports do not start reviewer selection.
          </p>
        ) : null}
        <ReviewRequestConfigurationFields {...props} />
      </div>
    </section>
  );
}

function ReviewRequestConfigurationFields(props: ConfigurationProps) {
  const { editor } = props;
  if (editor.unavailable || !editor.draft) {
    return (
      <div className="flex items-center gap-3">
        <p role={props.loading ? "status" : "alert"} className="text-sm text-muted-foreground">
          {props.loading ? "Review request is loading." : "Could not load Review request. Try again."}
        </p>
        {!props.loading && props.onRetry ? (
          <Button variant="outline" size="sm" onClick={props.onRetry}>
            Try again
          </Button>
        ) : null}
      </div>
    );
  }
  return (
    <>
      <ReviewRequestFallbackField
        value={editor.fallbackReviewer}
        reviewers={props.reviewers}
        loading={props.reviewersLoading ?? false}
        error={props.reviewersError ?? false}
        onRetry={props.onRetryReviewers}
        onChange={editor.updateFallbackReviewer}
      />
      <PlanningReviewForm
        draft={editor.draft}
        onChange={editor.setEdited}
        organizationId={props.organizationId}
        factoryId={props.factoryId}
        appearance="fields"
        view="planning"
        modelLabel="Model"
        modelDescription="The model used to select pull request reviewers."
        showResources={false}
        showConcurrency={false}
        showSteps={false}
        disabled={editor.saving}
      />
      <ReviewFindingsField
        findings={editor.findings}
        missingDestination={editor.findingsMissingDestination}
        disabled={editor.saving}
        onChange={editor.updateFindings}
      />
      <ImplementSettingRow
        label="Publish PR check"
        htmlFor="review-request-publish-check"
        description="Track required approvals. To enforce them, administrators must require superplane/review-request in GitHub branch rules."
      >
        <Switch
          id="review-request-publish-check"
          aria-describedby="review-request-publish-check-help"
          checked={editor.publishPRCheck}
          disabled={editor.saving}
          onCheckedChange={editor.updatePublishPRCheck}
        />
      </ImplementSettingRow>
      <AgentAdditionalInstructions
        draft={editor.draft}
        onChange={editor.setEdited}
        disabled={editor.saving}
        description="Guide reviewer selection, ownership preferences, and rationale. SuperPlane keeps the review requirements, eligibility checks, and fallback rules."
        placeholder="For example: Prefer the security owner for authentication changes. Explain the ownership evidence for each reviewer."
      />
    </>
  );
}

function ReviewFindingsField({
  findings,
  missingDestination,
  disabled,
  onChange,
}: {
  findings: ReviewFindingsSettings;
  missingDestination: boolean;
  disabled: boolean;
  onChange: (patch: Partial<ReviewFindingsSettings>) => void;
}) {
  return (
    <div>
      <ImplementSettingRow
        label="Publish review findings"
        htmlFor="review-request-publish-findings"
        description="Publish selected reviewers and the reason for the decision."
      >
        <Switch
          id="review-request-publish-findings"
          aria-describedby="review-request-publish-findings-help"
          checked={findings.publish}
          disabled={disabled}
          onCheckedChange={(publish: boolean) => onChange({ publish })}
        />
      </ImplementSettingRow>
      {findings.publish ? (
        <fieldset
          className="mt-3 flex flex-col gap-2"
          aria-describedby={missingDestination ? "review-request-findings-destination-error" : undefined}
        >
          <legend className="sr-only">Destinations</legend>
          {(
            [
              ["body", "PR body"],
              ["comment", "PR comment"],
            ] as const
          ).map(([destination, label]) => (
            <div key={destination} className="flex items-center gap-2">
              <Checkbox
                id={`review-request-findings-${destination}`}
                checked={findings[destination]}
                disabled={disabled}
                onChange={(event: ChangeEvent<HTMLInputElement>) => onChange({ [destination]: event.target.checked })}
              />
              <Label htmlFor={`review-request-findings-${destination}`} className="text-sm font-normal">
                {label}
              </Label>
            </div>
          ))}
          {missingDestination ? (
            <p id="review-request-findings-destination-error" role="alert" className="text-sm text-destructive">
              Select at least one destination.
            </p>
          ) : null}
        </fieldset>
      ) : null}
    </div>
  );
}

function ReviewRequestFallbackField({
  value,
  reviewers,
  loading,
  error,
  onChange,
  onRetry,
}: {
  value: string;
  reviewers: string[];
  loading: boolean;
  error: boolean;
  onChange: (login: string) => void;
  onRetry?: () => void;
}) {
  // The saved login stays selectable when GitHub is unavailable or access changes.
  const options = [
    ...new Map([...reviewers, value].filter(Boolean).map((login) => [login.toLowerCase(), login])).values(),
  ];
  return (
    <div>
      <ImplementSettingRow
        label="Fallback reviewer"
        htmlFor="review-request-fallback"
        description="Request this reviewer only when selection finds no suitable candidate."
      >
        <Select value={value || "__none__"} onValueChange={(next) => onChange(next === "__none__" ? "" : next)}>
          <SelectTrigger id="review-request-fallback" aria-label="Fallback reviewer" className="h-8 w-full shadow-none">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__none__">None</SelectItem>
            {options.map((login) => (
              <SelectItem key={login} value={login}>
                {login}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </ImplementSettingRow>
      {loading ? (
        <p role="status" className="mt-2 text-sm text-muted-foreground">
          GitHub reviewers are loading.
        </p>
      ) : null}
      {error ? (
        <div className="mt-2 flex items-center gap-3">
          <p role="alert" className="text-sm text-destructive">
            Could not load GitHub reviewers. Your saved reviewer is unchanged.
          </p>
          {onRetry ? (
            <Button variant="outline" size="sm" onClick={onRetry}>
              Try again
            </Button>
          ) : null}
        </div>
      ) : !loading && reviewers.length === 0 ? (
        <p className="mt-2 text-sm text-muted-foreground">
          No eligible GitHub reviewers were found in workspace repositories.
        </p>
      ) : null}
    </div>
  );
}
