export type OnboardingFactoryTemplate = {
  id: string;
  name: string;
  description: string;
  count: number;
};

export type ResetFactoryTemplateResult = {
  reset: number;
  failures: { canvas_id: string; name: string; error: string }[];
};

export const automationCountLabel = (count: number) => {
  if (count === 1) {
    return "1 automation";
  }

  return `${count} automations`;
};

export function resetFactoryTemplateResultMessage(name: string, result: ResetFactoryTemplateResult): string {
  const failures = result.failures ?? [];
  const failed = failures.length;
  if (result.reset === 0 && failed === 0) {
    return `No ${name} automations to reset.`;
  }

  const resetText = result.reset === 1 ? `Reset 1 ${name} automation.` : `Reset ${result.reset} ${name} automations.`;
  if (failed === 0) {
    return resetText;
  }

  const failText = failed === 1 ? "1 failed" : `${failed} failed`;
  const details = failures
    .map((failure) => {
      const label = failure.name?.trim() || failure.canvas_id;
      const reason = failure.error?.trim() ?? "";
      if (reason === "") {
        return label;
      }
      return `${label} (${reason})`;
    })
    .join("; ");
  return `${resetText} ${failText}: ${details}.`;
}
