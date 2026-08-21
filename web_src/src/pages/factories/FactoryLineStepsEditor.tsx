import type { FactoryApp } from "@/api-client";
import { Fragment } from "react";
import { LineStepAddButton, LineStepArrow, LineStepFlow } from "./FactoryLineStepFlow";
import type { DraftStep } from "./lib/factoryLineFormShared";
import { FactoryLineStepEditor } from "./FactoryLineStepEditor";

interface FactoryLineStepsEditorProps {
  organizationId: string;
  steps: DraftStep[];
  apps: FactoryApp[];
  appById: Map<string, FactoryApp>;
  onStepsChange: (steps: DraftStep[]) => void;
  onAddStep: () => void;
}

export function FactoryLineStepsEditor({
  organizationId,
  steps,
  apps,
  appById,
  onStepsChange,
  onAddStep,
}: FactoryLineStepsEditorProps) {
  return (
    <LineStepFlow variant="editor" className="pt-1">
      {steps.map((step, index) => (
        <Fragment key={index}>
          {index > 0 ? <LineStepArrow /> : null}
          <div className="w-full">
            {steps.length > 1 ? (
              <div className="mb-2 flex justify-end">
                <button
                  type="button"
                  onClick={() => onStepsChange(steps.filter((_, itemIndex) => itemIndex !== index))}
                  className="text-sm font-medium text-gray-500 hover:text-red-600 dark:text-gray-400 dark:hover:text-red-400"
                >
                  Remove
                </button>
              </div>
            ) : null}
            <FactoryLineStepEditor
              organizationId={organizationId}
              index={index}
              step={step}
              apps={apps}
              appById={appById}
              onChange={(updated) =>
                onStepsChange(steps.map((item, itemIndex) => (itemIndex === index ? updated : item)))
              }
            />
          </div>
        </Fragment>
      ))}

      <LineStepAddButton onClick={onAddStep} disabled={apps.length === 0} />
    </LineStepFlow>
  );
}
