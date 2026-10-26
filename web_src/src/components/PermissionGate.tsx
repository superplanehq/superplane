import React from "react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { usePermissions } from "@/contexts/usePermissions";
import { PermissionDeniedPage } from "@/components/PermissionDeniedPage";
import { cn } from "@/lib/utils";
import { useWorkspaceLoading } from "@/hooks/useWorkspaceLoading";
import { useIsSupportSession } from "@/hooks/useSupportSession";
import { SUPPORT_SESSION_READ_ONLY_MESSAGE } from "@/lib/supportSession";
import { WORKSPACE_LOADING_COPY } from "@/lib/workspaceLoadingCopy";

interface PermissionTooltipProps {
  allowed: boolean;
  message: string;
  children: React.ReactNode;
  className?: string;
}

export function PermissionTooltip({ allowed, message, children, className }: PermissionTooltipProps) {
  const inSupportSession = useIsSupportSession();
  if (allowed) return <>{children}</>;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div className={cn("inline-flex", className)}>
          <div className="pointer-events-none opacity-60 w-full">{children}</div>
        </div>
      </TooltipTrigger>
      <TooltipContent side="top">{inSupportSession ? SUPPORT_SESSION_READ_ONLY_MESSAGE : message}</TooltipContent>
    </Tooltip>
  );
}

interface RequirePermissionProps {
  resource: string;
  action: string;
  children: React.ReactNode;
}

function PermissionLoading() {
  const overlayHandles = useWorkspaceLoading(WORKSPACE_LOADING_COPY.access, true);
  if (overlayHandles) {
    return null;
  }
  return (
    <div className="min-h-screen flex items-center justify-center">
      <p className="text-gray-500">Checking permissions...</p>
    </div>
  );
}

// Route gates use the real permission so a read-only support session can
// still open pages whose actions it cannot use.
export function RequirePermission({ resource, action, children }: RequirePermissionProps) {
  const { hasPermission, isLoading } = usePermissions();

  if (isLoading) {
    return <PermissionLoading />;
  }

  if (!hasPermission(resource, action)) {
    return <PermissionDeniedPage resource={resource} action={action} />;
  }

  return <>{children}</>;
}

interface RequireAnyPermissionProps {
  checks: Array<{ resource: string; action: string }>;
  children: React.ReactNode;
}

export function RequireAnyPermission({ checks, children }: RequireAnyPermissionProps) {
  const { hasPermission, isLoading } = usePermissions();

  if (isLoading) {
    return <PermissionLoading />;
  }

  if (!checks.some((check) => hasPermission(check.resource, check.action))) {
    return <PermissionDeniedPage />;
  }

  return <>{children}</>;
}
