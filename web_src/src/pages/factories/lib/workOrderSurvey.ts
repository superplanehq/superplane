import type { FactoriesWorkOrder } from "@/api-client";

export type WorkOrderSurveyQuestionView = {
  id: string;
  prompt: string;
  options: string[];
  allowFreeText: boolean;
};

export type WorkOrderSurveyView = {
  id: string;
  status: string;
  canvasRunId?: string;
  questions: WorkOrderSurveyQuestionView[];
  expiresAt?: string;
};

export type WorkOrderSurveyAnswerInput = {
  id: string;
  value: string;
};

export function workOrderPendingSurvey(order: FactoriesWorkOrder | null | undefined): WorkOrderSurveyView | undefined {
  if (!order) {
    return undefined;
  }
  const raw = order.pendingSurvey;
  if (!raw?.id || raw.status !== "pending" || !raw.questions?.length) {
    return undefined;
  }

  const questions = raw.questions
    .map((question) => {
      const id = question.id?.trim() ?? "";
      const prompt = question.prompt?.trim() ?? "";
      if (!id || !prompt) {
        return undefined;
      }
      return {
        id,
        prompt,
        options: question.options ?? [],
        allowFreeText: Boolean(question.allowFreeText) || (question.options?.length ?? 0) === 0,
      };
    })
    .filter((question): question is WorkOrderSurveyQuestionView => Boolean(question));

  if (questions.length === 0) {
    return undefined;
  }

  return {
    id: raw.id,
    status: raw.status,
    canvasRunId: raw.canvasRunId,
    questions,
    expiresAt: raw.expiresAt,
  };
}
