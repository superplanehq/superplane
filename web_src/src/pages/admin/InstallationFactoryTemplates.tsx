import { Button } from "@/components/ui/button";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Text } from "@/components/Text/text";
import { AlertTriangle } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import {
  automationCountLabel,
  resetFactoryTemplateResultMessage,
  type OnboardingFactoryTemplate,
  type ResetFactoryTemplateResult,
} from "./factoryTemplateReset";

const useInstallationFactoryTemplates = () => {
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

  return {
    templates,
    loading,
    loadError,
    confirmTemplate,
    busyID,
    messages,
    errors,
    setConfirmTemplate,
    handleConfirm,
  };
};

export function InstallationFactoryTemplates() {
  const model = useInstallationFactoryTemplates();

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

      {model.loading && model.templates.length === 0 ? (
        <Text className="mt-5 text-sm text-gray-500 dark:text-gray-400">Loading onboarding templates...</Text>
      ) : (
        <>
          {model.loadError ? (
            <Text className="mt-5 text-sm text-red-600 dark:text-red-400">{model.loadError}</Text>
          ) : null}
          {model.templates.length === 0 ? null : (
            <FactoryTemplateList
              templates={model.templates}
              busyID={model.busyID}
              messages={model.messages}
              errors={model.errors}
              onReset={model.setConfirmTemplate}
            />
          )}
        </>
      )}

      <ResetFactoryTemplateDialog
        template={model.confirmTemplate}
        busyID={model.busyID}
        onConfirm={model.handleConfirm}
        onClose={() => model.setConfirmTemplate(null)}
      />
    </section>
  );
}

type FactoryTemplateListProps = {
  templates: OnboardingFactoryTemplate[];
  busyID: string | null;
  messages: Record<string, string>;
  errors: Record<string, string>;
  onReset: (template: OnboardingFactoryTemplate) => void;
};

const FactoryTemplateList = ({ templates, busyID, messages, errors, onReset }: FactoryTemplateListProps) => (
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
            onClick={() => onReset(template)}
            disabled={busy || template.count === 0}
          >
            {`Reset ${template.name}`}
          </Button>
        </li>
      );
    })}
  </ul>
);

type ResetFactoryTemplateDialogProps = {
  template: OnboardingFactoryTemplate | null;
  busyID: string | null;
  onConfirm: (template: OnboardingFactoryTemplate) => Promise<void>;
  onClose: () => void;
};

const ResetFactoryTemplateDialog = ({ template, busyID, onConfirm, onClose }: ResetFactoryTemplateDialogProps) => (
  <Dialog open={template != null} onClose={() => (busyID ? undefined : onClose())} size="md">
    {template ? (
      <>
        <div className="mb-2 flex items-center gap-3">
          <div className="rounded-full bg-red-100 p-2 text-red-600 dark:bg-red-950/40 dark:text-red-300">
            <AlertTriangle size={20} />
          </div>
          <DialogTitle className="text-gray-800 dark:text-gray-100">{`Reset ${template.name} automations`}</DialogTitle>
        </div>
        <DialogDescription className="mt-2 space-y-2 text-sm text-gray-600 dark:text-gray-400">
          <p>
            {`This action replaces every ${template.name} automation on this installation with the current SuperPlane defaults.`}
          </p>
          <p>Custom prompts and graph changes in those automations are lost. Other automations stay the same.</p>
          <p>You cannot undo this action.</p>
        </DialogDescription>
        <DialogActions>
          <Button variant="destructive" onClick={() => void onConfirm(template)} disabled={busyID === template.id}>
            {busyID === template.id ? "Resetting..." : `Reset ${template.name} automations`}
          </Button>
          <Button variant="outline" onClick={onClose} disabled={busyID === template.id}>
            Cancel
          </Button>
        </DialogActions>
      </>
    ) : null}
  </Dialog>
);
