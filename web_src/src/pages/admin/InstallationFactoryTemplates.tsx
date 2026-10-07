import { Button } from "@/components/ui/button";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Text } from "@/components/Text/text";
import { AlertTriangle } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

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

const automationCountLabel = (count: number) => {
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

export function InstallationFactoryTemplates() {
  const [templates, setTemplates] = useState<OnboardingFactoryTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [confirmTemplate, setConfirmTemplate] = useState<OnboardingFactoryTemplate | null>(null);
  const [busyID, setBusyID] = useState<string | null>(null);
  const [messages, setMessages] = useState<Record<string, string>>({});
  const [errors, setErrors] = useState<Record<string, string>>({});

  const loadTemplates = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const response = await fetch("/admin/api/installation/factory-templates", { credentials: "include" });
      if (!response.ok) {
        throw new Error("load failed");
      }
      const payload = (await response.json()) as { templates?: OnboardingFactoryTemplate[] };
      if (!Array.isArray(payload.templates)) {
        throw new Error("load failed");
      }
      setTemplates(payload.templates);
    } catch {
      setLoadError("Could not load onboarding templates.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadTemplates();
  }, [loadTemplates]);

  const handleConfirm = async (template: OnboardingFactoryTemplate) => {
    setBusyID(template.id);
    setErrors((current) => ({ ...current, [template.id]: "" }));
    setMessages((current) => ({ ...current, [template.id]: "" }));
    try {
      const response = await fetch(`/admin/api/installation/factory-templates/${template.id}/reset`, {
        method: "POST",
        credentials: "include",
      });
      if (!response.ok) {
        throw new Error("reset failed");
      }
      const result = (await response.json()) as ResetFactoryTemplateResult;
      setConfirmTemplate(null);
      setMessages((current) => ({
        ...current,
        [template.id]: resetFactoryTemplateResultMessage(template.name, result),
      }));
      await loadTemplates();
    } catch {
      setConfirmTemplate(null);
      setErrors((current) => ({
        ...current,
        [template.id]: `Could not reset ${template.name} automations.`,
      }));
    } finally {
      setBusyID(null);
    }
  };

  return (
    <section className="border-t border-slate-200 py-6 dark:border-gray-700/70">
      <div className="max-w-2xl">
        <p className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
          Factory templates
        </p>
        <h2 className="mt-1 text-base font-semibold text-gray-900 dark:text-gray-100">Onboarding templates</h2>
        <Text className="mt-2 text-sm text-gray-600 dark:text-gray-400">
          Reset SuperPlane onboarding automations to the current defaults. Reset applies to every organization that has
          that template.
        </Text>
      </div>

      {loading && templates.length === 0 ? (
        <Text className="mt-5 text-sm text-gray-500 dark:text-gray-400">Loading onboarding templates...</Text>
      ) : loadError ? (
        <Text className="mt-5 text-sm text-red-600 dark:text-red-400">{loadError}</Text>
      ) : (
        <ul className="mt-6 divide-y divide-slate-200 dark:divide-gray-700/70">
          {templates.map((template) => {
            const busy = busyID === template.id;
            return (
              <li key={template.id} className="flex flex-col gap-3 py-4 sm:flex-row sm:items-start sm:justify-between">
                <div className="min-w-0">
                  <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{template.name}</p>
                  <Text className="mt-1 text-sm text-gray-600 dark:text-gray-400">{template.description}</Text>
                  <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {automationCountLabel(template.count)}
                  </Text>
                  {messages[template.id] ? (
                    <Text className="mt-2 text-sm text-gray-600 dark:text-gray-400">{messages[template.id]}</Text>
                  ) : null}
                  {errors[template.id] ? (
                    <Text className="mt-2 text-sm text-red-600 dark:text-red-400">{errors[template.id]}</Text>
                  ) : null}
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="shrink-0"
                  onClick={() => setConfirmTemplate(template)}
                  disabled={busy || template.count === 0}
                >
                  {`Reset ${template.name}`}
                </Button>
              </li>
            );
          })}
        </ul>
      )}

      <Dialog open={confirmTemplate != null} onClose={() => (busyID ? undefined : setConfirmTemplate(null))} size="md">
        {confirmTemplate ? (
          <>
            <div className="mb-2 flex items-center gap-3">
              <div className="rounded-full bg-red-100 p-2 text-red-600 dark:bg-red-950/40 dark:text-red-300">
                <AlertTriangle size={20} />
              </div>
              <DialogTitle className="text-gray-800 dark:text-gray-100">
                {`Reset ${confirmTemplate.name} automations`}
              </DialogTitle>
            </div>
            <DialogDescription className="mt-2 space-y-2 text-sm text-gray-600 dark:text-gray-400">
              <p>
                {`This action replaces every ${confirmTemplate.name} automation on this installation with the current SuperPlane defaults.`}
              </p>
              <p>Custom prompts and graph changes in those automations are lost. Other automations stay the same.</p>
              <p>You cannot undo this action.</p>
            </DialogDescription>
            <DialogActions>
              <Button
                variant="destructive"
                onClick={() => void handleConfirm(confirmTemplate)}
                disabled={busyID === confirmTemplate.id}
              >
                {busyID === confirmTemplate.id ? "Resetting..." : `Reset ${confirmTemplate.name} automations`}
              </Button>
              <Button
                variant="outline"
                onClick={() => setConfirmTemplate(null)}
                disabled={busyID === confirmTemplate.id}
              >
                Cancel
              </Button>
            </DialogActions>
          </>
        ) : null}
      </Dialog>
    </section>
  );
}
