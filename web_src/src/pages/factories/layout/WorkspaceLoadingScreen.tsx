import superplaneIcon from "@/assets/superplane-icon.png";
import { cn } from "@/lib/utils";

import { WORKSPACE_LOADING_TEST_ID } from "@/lib/workspaceLoadingCopy";

import { LOADING_REVEAL_CLASSNAME } from "../lib/loadingReveal";

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
      <img src={superplaneIcon} alt="" width={88} height={88} className="rounded-2xl" />
      <p key={message} className={cn("text-[15px] font-medium text-foreground", LOADING_REVEAL_CLASSNAME)}>
        {message}
      </p>
    </div>
  );
}
