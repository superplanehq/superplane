import superplaneAnimation from "@/assets/superplane-loading.gif";
import superplaneLogo from "@/assets/superplane.svg";
import { cn } from "@/lib/utils";

import { WORKSPACE_LOADING_TEST_ID } from "@/lib/workspaceLoadingCopy";

import { LOADING_REVEAL_CLASSNAME } from "../lib/loadingReveal";

function WorkspaceLoadingMark() {
  return (
    <div className="workspace-loading-mark" aria-hidden>
      <img src={superplaneAnimation} alt="" className="workspace-loading-animation" />
      <img src={superplaneLogo} alt="" className="workspace-loading-static brightness-0 dark:invert" />
    </div>
  );
}

export function WorkspaceLoadingScreen({ message, exiting = false }: { message: string; exiting?: boolean }) {
  return (
    <div
      className={cn(
        "fixed inset-0 z-50 flex min-h-screen flex-col items-center justify-center gap-5 bg-background text-foreground",
        exiting && "workspace-loading-overlay--exit",
      )}
      role="status"
      aria-label={message}
      aria-live="polite"
      aria-busy={!exiting}
      data-testid={WORKSPACE_LOADING_TEST_ID}
    >
      <WorkspaceLoadingMark />
      <p key={message} className={cn("text-[15px] font-medium text-foreground", LOADING_REVEAL_CLASSNAME)}>
        {message}
      </p>
    </div>
  );
}
