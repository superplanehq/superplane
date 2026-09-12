import type { CreateWithAgentView } from "../createWithAgentTypes";
import { PlanningSessionSurveyForm } from "../PlanningSessionSurveyForm";

type WorkOrderIntentSurveyProps = {
  survey: NonNullable<CreateWithAgentView["survey"]>;
  onSubmit: (text: string) => void;
};

export function WorkOrderIntentSurvey({ survey, onSubmit }: WorkOrderIntentSurveyProps) {
  return (
    <PlanningSessionSurveyForm
      key={survey.id ?? survey.questions[0]?.prompt ?? "survey"}
      survey={survey}
      onSubmit={onSubmit}
    />
  );
}
