import type { CreateWithAgentView } from "../createWithAgentTypes";
import { PlanningSessionSurveyForm } from "../PlanningSessionSurveyForm";

type WorkOrderIntentSurveyProps = {
  survey: NonNullable<CreateWithAgentView["survey"]>;
  onSubmit: (text: string) => void;
  disabled?: boolean;
  planningReviewEnabled?: boolean;
};

export function WorkOrderIntentSurvey({
  survey,
  onSubmit,
  disabled,
  planningReviewEnabled,
}: WorkOrderIntentSurveyProps) {
  if (!planningReviewEnabled) {
    return (
      <PlanningSessionSurveyForm
        key={survey.id ?? survey.questions[0]?.prompt ?? "survey"}
        survey={survey}
        onSubmit={onSubmit}
      />
    );
  }
  return (
    <fieldset disabled={disabled} className="min-w-0">
      <PlanningSessionSurveyForm
        key={survey.id ?? survey.questions[0]?.prompt ?? "survey"}
        survey={survey}
        nextVariant="default"
        onSubmit={onSubmit}
      />
    </fieldset>
  );
}
