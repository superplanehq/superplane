import type { CreateWithAgentView } from "../createWithAgentTypes";
import { PlanningSessionSurveyForm } from "../PlanningSessionSurveyForm";

type WorkOrderIntentSurveyProps = {
  survey: NonNullable<CreateWithAgentView["survey"]>;
  onSubmit: (text: string) => void;
  disabled?: boolean;
};

export function WorkOrderIntentSurvey({ survey, onSubmit, disabled }: WorkOrderIntentSurveyProps) {
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
