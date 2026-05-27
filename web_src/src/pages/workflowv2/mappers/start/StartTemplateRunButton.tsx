import React from "react";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";

import type { CustomFieldRendererContext } from "../types";
import { hasManualRunParams } from "./manualRunParams";
import { StartRunParamsForm } from "./runParamsForm";

interface StartTemplate {
  name: string;
  payload: Record<string, unknown>;
}

function payloadForTemplateRun(template: StartTemplate): Record<string, unknown> {
  const p = template.payload;
  if (p && typeof p === "object" && !Array.isArray(p)) {
    return p as Record<string, unknown>;
  }
  return {};
}

export function StartTemplateRunButton({
  template,
  nodeName,
  actions,
}: {
  template: StartTemplate;
  nodeName: string;
  actions: NonNullable<CustomFieldRendererContext["actions"]>;
}) {
  const [isRunning, setIsRunning] = React.useState(false);
  const payload = payloadForTemplateRun(template);
  const parameterized = hasManualRunParams(payload);

  const invokeRun = async (runPayload: Record<string, unknown>) => {
    await actions.invokeNodeTriggerHook("run", {
      template: template.name,
      payload: runPayload,
    });
  };

  const handleDirectRun = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsRunning(true);
    try {
      await invokeRun(payload);
    } catch {
      // Toast handled by invoke hook.
    } finally {
      setIsRunning(false);
    }
  };

  const handleParameterizedRun = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    actions.openModal({
      title: "Run trigger",
      description: (
        <>
          Run template <strong>{template.name}</strong> on node <strong>{nodeName}</strong>.
        </>
      ),
      content: ({ close }) => <StartRunParamsForm templatePayload={payload} onClose={close} onRun={invokeRun} />,
    });
  };

  if (parameterized) {
    return (
      <Button
        size="sm"
        data-testid="start-template-run"
        onClick={handleParameterizedRun}
        className="flex-shrink-0 h-7 py-1 px-2 bg-black text-white hover:bg-black/80"
      >
        Run
      </Button>
    );
  }

  return (
    <LoadingButton
      size="sm"
      data-testid="start-template-run"
      loading={isRunning}
      loadingText="Running..."
      onClick={handleDirectRun}
      className="flex-shrink-0 h-7 py-1 px-2 bg-black text-white hover:bg-black/80"
    >
      Run
    </LoadingButton>
  );
}
