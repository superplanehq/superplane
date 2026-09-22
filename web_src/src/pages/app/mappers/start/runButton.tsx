import React from "react";
import { LoadingButton } from "@/components/ui/loading-button";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import type { TriggerActionContext } from "../types";
import { StartRunModal } from "./runModal";
import { payloadForTemplateRun, startRunModalTitle, type StartTemplate } from "./templatePayload";

export function StartTemplateRunButton({
  nodeName,
  template,
  actions,
}: {
  nodeName: string;
  template: StartTemplate;
  actions: TriggerActionContext;
}) {
  const [isRunning, setIsRunning] = React.useState(false);

  const handleRun = async (event: React.MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();

    if ((template.parameters?.length ?? 0) > 0) {
      actions.openModal({
        title: startRunModalTitle(nodeName, template.name),
        content: ({ close }) => (
          <StartRunModal
            parameters={template.parameters}
            initialPayload={payloadForTemplateRun(template)}
            onClose={close}
            onRun={async (payload) =>
              actions.invokeNodeTriggerHook("run", {
                template: template.name,
                ...payload,
              })
            }
          />
        ),
      });
      return;
    }

    setIsRunning(true);
    try {
      await actions.invokeNodeTriggerHook("run", {
        template: template.name,
      });
    } catch {
      // The trigger action context reports request errors to the user.
    } finally {
      setIsRunning(false);
    }
  };

  return (
    <LoadingButton
      size="xs"
      data-testid="start-template-run"
      loading={isRunning}
      loadingText="Running..."
      onClick={handleRun}
      className={cn("flex-shrink-0", appDarkModeClasses.primaryAction)}
    >
      Run
    </LoadingButton>
  );
}
