import { ArrowLeft, Factory } from "lucide-react";
import { useCallback, useState, type FormEvent } from "react";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

import type { Automation, NewWorkOrderInput, SoftwareFactory } from "./factoryTypes";

interface NewWorkOrderPageProps {
  factory: SoftwareFactory;
  automations: Automation[];
  onCancel: () => void;
  onCreate: (input: NewWorkOrderInput) => void;
}

const emptyWorkOrder: NewWorkOrderInput = { title: "", description: "", automationIds: [] };

export function NewWorkOrderPage({ factory, automations, onCancel, onCreate }: NewWorkOrderPageProps) {
  const [draft, setDraft] = useState<NewWorkOrderInput>(emptyWorkOrder);
  const canCreate =
    draft.title.trim().length > 0 && draft.description.trim().length > 0 && draft.automationIds.length > 0;

  const setAutomationSelected = useCallback((automationId: string, selected: boolean) => {
    setDraft((current) => ({
      ...current,
      automationIds: selected
        ? [...current.automationIds, automationId]
        : current.automationIds.filter((id) => id !== automationId),
    }));
  }, []);

  const createWorkOrder = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      if (!canCreate) return;

      onCreate({
        title: draft.title.trim(),
        description: draft.description.trim(),
        automationIds: draft.automationIds,
      });
    },
    [canCreate, draft, onCreate],
  );

  return (
    <div className="min-h-screen bg-slate-50 text-slate-950 dark:bg-gray-950 dark:text-gray-100">
      <main className="mx-auto w-full max-w-[1040px] px-4 py-6 sm:px-6 sm:py-8 lg:px-10">
        <Button type="button" variant="ghost" size="sm" className="-ml-3" onClick={onCancel}>
          <ArrowLeft aria-hidden />
          {factory.name}
        </Button>

        <header className="mt-5 border-b border-slate-200 pb-7 dark:border-gray-800">
          <div className="mb-2 flex items-center gap-2 text-xs font-medium text-slate-500 dark:text-gray-400">
            <Factory className="size-4" aria-hidden />
            New Work Order
          </div>
          <h1 className="text-2xl font-semibold text-slate-950 dark:text-white">Describe the work</h1>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-600 dark:text-gray-400">
            Capture one concrete software change. The Work Order will remain a draft until it is approved.
          </p>
        </header>

        <form className="mt-8" onSubmit={createWorkOrder}>
          <div className="space-y-3">
            <Label htmlFor="new-work-order-title">Title</Label>
            <Input
              id="new-work-order-title"
              autoFocus
              className="h-10 text-base"
              value={draft.title}
              placeholder="Add refund reconciliation test"
              onChange={(event) => setDraft((current) => ({ ...current, title: event.target.value }))}
            />
          </div>

          <div className="mt-6 space-y-3">
            <div className="flex flex-wrap items-baseline justify-between gap-2">
              <Label htmlFor="new-work-order-description">Description</Label>
              <span className="text-xs text-slate-400 dark:text-gray-500">Expected outcome and constraints</span>
            </div>
            <Textarea
              id="new-work-order-description"
              rows={14}
              className="min-h-80 resize-y text-base leading-7"
              value={draft.description}
              placeholder="Describe what should change, why it matters, and anything the Automation must preserve."
              onChange={(event) => setDraft((current) => ({ ...current, description: event.target.value }))}
            />
          </div>

          <fieldset className="mt-6 border-t border-slate-200 pt-6 dark:border-gray-800">
            <legend className="text-sm font-medium text-slate-800 dark:text-gray-100">Automations</legend>
            <p className="mt-2 text-xs leading-5 text-slate-500 dark:text-gray-400">
              Choose one or more pipelines to process this Work Order after approval.
            </p>

            {automations.length === 0 ? (
              <p className="mt-3 rounded-md border border-dashed border-slate-300 px-4 py-3 text-sm text-slate-500 dark:border-gray-700 dark:text-gray-400">
                No Automations are available.
              </p>
            ) : (
              <div className="mt-3 grid gap-3 sm:grid-cols-2">
                {automations.map((automation) => {
                  const checkboxId = `new-work-order-automation-${automation.id}`;

                  return (
                    <label
                      key={automation.id}
                      htmlFor={checkboxId}
                      className="flex cursor-pointer items-start gap-3 rounded-md border border-slate-200 bg-white px-4 py-3 transition-colors has-[:checked]:border-slate-900 has-[:checked]:bg-slate-50 dark:border-gray-700 dark:bg-gray-900 dark:has-[:checked]:border-gray-300 dark:has-[:checked]:bg-gray-800"
                    >
                      <Checkbox
                        id={checkboxId}
                        checked={draft.automationIds.includes(automation.id)}
                        onChange={(event) => setAutomationSelected(automation.id, event.target.checked)}
                      />
                      <span className="min-w-0">
                        <span className="block text-sm font-medium text-slate-950 dark:text-gray-100">
                          {automation.name}
                        </span>
                        <span className="mt-1 block text-xs leading-5 text-slate-500 dark:text-gray-400">
                          {automation.description}
                        </span>
                      </span>
                    </label>
                  );
                })}
              </div>
            )}
          </fieldset>

          <footer className="mt-8 flex flex-col-reverse gap-3 border-t border-slate-200 pt-6 dark:border-gray-800 sm:flex-row sm:justify-end">
            <Button type="button" variant="outline" onClick={onCancel}>
              Cancel
            </Button>
            <Button type="submit" disabled={!canCreate}>
              Create draft
            </Button>
          </footer>
        </form>
      </main>
    </div>
  );
}
