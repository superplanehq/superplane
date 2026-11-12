import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
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
    </>
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
