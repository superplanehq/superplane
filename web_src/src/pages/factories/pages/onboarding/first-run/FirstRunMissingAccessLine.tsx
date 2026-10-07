import { LoadingButton } from "@/components/ui/loading-button";

import { FIRST_RUN_COPY } from "./firstRunCopy";

/** A short hint and question, with an inline GitHub action right after it. */
export function FirstRunMissingAccessLine({
  hint,
  question,
  action,
  disabled,
  loading,
  onClick,
}: {
  hint: string;
  question: string;
  action: string;
  disabled: boolean;
  loading: boolean;
  onClick: () => void;
}) {
  return (
    <div className="space-y-1 text-[13px] text-muted-foreground">
      <p>{hint}</p>
      <p>
        {question}{" "}
        <LoadingButton
          type="button"
          variant="link"
          className="h-auto p-0 align-baseline text-[13px] font-medium text-foreground underline underline-offset-2 hover:no-underline"
          onClick={onClick}
          disabled={disabled}
          loading={loading}
          loadingText={FIRST_RUN_COPY.choose.openingGitHub}
          data-testid="first-run-grant-access"
        >
          {action}
        </LoadingButton>
      </p>
    </div>
  );
}
