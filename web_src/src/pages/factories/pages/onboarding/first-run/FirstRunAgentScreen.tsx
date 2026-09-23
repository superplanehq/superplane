import { LoadingButton } from "@/components/ui/loading-button";
import type { ReactNode } from "react";

import { JiraCompletionColumnFields } from "../../JiraCompletionColumnFields";
import type { JiraCompletionColumnValue } from "../../jiraCompletionColumn";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

export type FirstRunAgentJiraCompletion = {
  organizationId: string;
  integrationId: string;
  projectId: string;
  value: JiraCompletionColumnValue;
  onChange: (next: JiraCompletionColumnValue) => void;
};

export function FirstRunAgentScreen({
  chrome,
  sphere,
  saving,
  loading,
  agentReady,
  intro,
  jiraCompletion,
  children,
  onContinue,
}: {
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  saving: boolean;
  loading: boolean;
  agentReady: boolean;
  intro: string;
  jiraCompletion?: FirstRunAgentJiraCompletion;
  children: ReactNode;
  onContinue: () => void;
}) {
  return (
    <FirstRunShell testId="first-run-agent" chrome={chrome} busy={saving || loading} width="wide" sphere={sphere}>
      <FirstRunHeading headline={FIRST_RUN_COPY.agent.headline}>
        <p className="text-[13px] text-muted-foreground">{intro}</p>
      </FirstRunHeading>

      <div className="mt-8 flex flex-col gap-6">
        {loading ? (
          <p className="text-[13px] text-muted-foreground" role="status">
            {FIRST_RUN_COPY.agent.loading}
          </p>
        ) : null}
        <fieldset disabled={saving || loading} className="m-0 flex min-w-0 flex-col gap-6 border-0 p-0">
          <FirstRunPanel>{children}</FirstRunPanel>
          {jiraCompletion ? (
            <JiraCompletionColumnFields
              organizationId={jiraCompletion.organizationId}
              integrationId={jiraCompletion.integrationId}
              projectId={jiraCompletion.projectId}
              value={jiraCompletion.value}
              onChange={jiraCompletion.onChange}
              layout="plain"
            />
          ) : null}
        </fieldset>
        <LoadingButton
          type="button"
          className="w-full"
          disabled={!agentReady || loading}
          loading={saving}
          loadingText={FIRST_RUN_COPY.finish.saving}
          onClick={onContinue}
          data-testid="first-run-finish-setup"
        >
          {FIRST_RUN_COPY.finish.action}
        </LoadingButton>
      </div>
    </FirstRunShell>
  );
}
