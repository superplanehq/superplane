import type { CreateWithAgentSurvey, CreateWithAgentSurveyQuestion } from "./createWithAgentTypes";
import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";

export function parsePlanningSurvey(raw: string | undefined): CreateWithAgentSurvey | undefined {
  if (!raw?.trim()) {
    return undefined;
  }
  try {
    const parsed = JSON.parse(raw) as { questions?: unknown };
    const questions = Array.isArray(parsed.questions) ? parsed.questions.flatMap(planningSurveyQuestion) : [];
    if (!questions.length) {
      return undefined;
    }
    return { questions };
  } catch {
    return undefined;
  }
}

export function formatPlanningSurveyReply(
  questions: CreateWithAgentSurveyQuestion[],
  answers: Array<string | null | undefined>,
): string {
  const lines = questions.map((question, index) => {
    const answer = answers[index]?.trim() ?? "";
    return `${question.prompt} ${answer || "skipped"}`;
  });
  if (lines.every((_, index) => !answers[index]?.trim())) {
    return CREATE_WITH_AGENT_COPY.surveySkipped;
  }
  return lines.join("\n");
}

function planningSurveyQuestion(value: unknown): CreateWithAgentSurveyQuestion[] {
  if (!value || typeof value !== "object") {
    return [];
  }
  const question = value as { prompt?: unknown; options?: unknown };
  const prompt = typeof question.prompt === "string" ? question.prompt.trim() : "";
  if (!prompt) {
    return [];
  }
  const options = Array.isArray(question.options)
    ? question.options.flatMap((option) => (typeof option === "string" && option.trim() ? [option.trim()] : []))
    : [];
  if (!options.length) {
    return [];
  }
  return [{ prompt, options }];
}
