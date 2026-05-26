import React from "react";
import { Play } from "lucide-react";
import { LoadingButton } from "@/components/ui/loading-button";
import { Separator } from "@/components/ui/separator";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import type { NodeInfo, TriggerActionContext } from "../types";
import { parseParams } from "./paramSyntax";
import { StartRunForm } from "./StartRunForm";
import { StartRunSummaryTable } from "./StartRunSummaryTable";

export type StartTemplate = {
  name: string;
  payload: Record<string, unknown>;
};

export type StartConfiguration = {
  templates?: StartTemplate[];
};

export function StartTemplatesField({
  node,
  templates,
  actions,
}: {
  node: NodeInfo;
  templates: StartTemplate[];
  actions: TriggerActionContext;
}) {
  const [runningTemplate, setRunningTemplate] = React.useState<string | null>(null);

  const handleRun = async (template: StartTemplate) => {
    const payload = payloadForTemplateRun(template);
    let defs;
    try {
      defs = parseParams(payload);
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Invalid template parameters");
      return;
    }

    if (defs.length === 0) {
      setRunningTemplate(template.name);
      try {
        await actions.invokeNodeTriggerHook("run", { template: template.name });
      } finally {
        setRunningTemplate(null);
      }
      return;
    }

    const nodeName = node.name || "Unnamed trigger";

    actions.openModal({
      title: "Run trigger",
      content: ({ close }) => (
        <div className="space-y-4">
          <StartRunSummaryTable nodeName={nodeName} templateName={template.name} />
          <Separator />
          <StartRunForm
            defs={defs}
            onClose={close}
            onRun={async (params) => {
              await actions.invokeNodeTriggerHook("run", {
                template: template.name,
                params,
              });
            }}
          />
        </div>
      ),
    });
  };

  return (
    <div className="px-2 py-1.5 flex flex-col gap-1.5">
      {templates.map((template, index) => {
        const isRunning = runningTemplate === template.name;
        return (
          <div key={index} className="flex items-center justify-between min-w-0">
            <div className="flex items-center min-w-0 flex-1">
              <div className="w-4 h-4 mr-2 flex-shrink-0">
                <Play size={16} className="text-gray-500" />
              </div>
              <span className="text-[13px] font-medium font-inter text-gray-500 truncate">{template.name}</span>
            </div>
            <LoadingButton
              size="sm"
              data-testid="start-template-run"
              loading={isRunning}
              loadingText="Running..."
              disabled={runningTemplate !== null && !isRunning}
              onClick={(event) => {
                event.preventDefault();
                event.stopPropagation();
                void handleRun(template).catch((error) => {
                  showErrorToast(getApiErrorMessage(error, "failed to run trigger"));
                });
              }}
              className="flex-shrink-0 h-7 py-1 px-2 bg-black text-white hover:bg-black/80"
            >
              Run
            </LoadingButton>
          </div>
        );
      })}
    </div>
  );
}

function payloadForTemplateRun(template: StartTemplate): Record<string, unknown> {
  const payload = template.payload;
  if (payload && typeof payload === "object" && !Array.isArray(payload)) {
    return payload;
  }
  return {};
}
