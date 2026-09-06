import type { FactoryApp } from "@/api-client";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useCanvas } from "@/hooks/useCanvasData";
import { cn } from "@/lib/utils";
import { useMemo } from "react";
import { listTriggerNodes } from "./lib/factoryCanvasTriggers";
import type { DraftStep } from "./lib/factoryLineFormShared";
import { LineStepEditorShell } from "./FactoryLineStepFlow";

const stepFieldClassName = "w-full min-w-0";

interface FactoryLineStepEditorProps {
  organizationId: string;
  index: number;
  step: DraftStep;
  apps: FactoryApp[];
  appById: Map<string, FactoryApp>;
  onChange: (step: DraftStep) => void;
}

export function FactoryLineStepEditor({
  organizationId,
  index,
  step,
  apps,
  appById,
  onChange,
}: FactoryLineStepEditorProps) {
  const { data: canvas, isLoading: canvasLoading } = useCanvas(organizationId, step.appId, {
    enabled: Boolean(step.appId),
  });
  const triggers = useMemo(() => listTriggerNodes(canvas), [canvas]);

  return (
    <LineStepEditorShell>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className={cn("space-y-2", stepFieldClassName)}>
          <Label htmlFor={`factory-line-step-app-${index}`}>Automation</Label>
          <Select value={step.appId} onValueChange={(appId) => onChange({ ...step, appId, entrypoint: "" })}>
            <SelectTrigger id={`factory-line-step-app-${index}`} className={stepFieldClassName}>
              <SelectValue placeholder="Select automation" />
            </SelectTrigger>
            <SelectContent>
              {apps.map((app) => (
                <SelectItem key={app.id} value={app.id ?? ""}>
                  {app.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className={cn("space-y-2", stepFieldClassName)}>
          <Label htmlFor={`factory-line-step-entrypoint-${index}`}>Trigger</Label>
          <Select
            value={step.entrypoint}
            onValueChange={(entrypoint) => onChange({ ...step, entrypoint })}
            disabled={!step.appId || canvasLoading}
          >
            <SelectTrigger id={`factory-line-step-entrypoint-${index}`} className={stepFieldClassName}>
              <SelectValue placeholder={canvasLoading ? "Loading triggers…" : "Select trigger"} />
            </SelectTrigger>
            <SelectContent>
              {triggers.map((trigger) => (
                <SelectItem key={trigger.id} value={trigger.id ?? ""}>
                  {trigger.name || trigger.id}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {step.appId && !canvasLoading && triggers.length === 0 ? (
            <p className="text-xs text-amber-700 dark:text-amber-300">
              {appById.get(step.appId)?.name ?? "This automation"} has no triggers yet.
            </p>
          ) : null}
        </div>

        <div className={cn("space-y-2", stepFieldClassName)}>
          <Label htmlFor={`factory-line-step-max-parallelism-${index}`}>Max parallelism</Label>
          <Input
            id={`factory-line-step-max-parallelism-${index}`}
            className={stepFieldClassName}
            type="number"
            min={1}
            value={step.maxParallelism}
            onChange={(event) => onChange({ ...step, maxParallelism: event.target.value })}
            placeholder="10"
          />
          <p className="text-xs text-muted-foreground">
            Maximum number of tasks to be processed in parallel at this step.
          </p>
        </div>
      </div>
    </LineStepEditorShell>
  );
}
