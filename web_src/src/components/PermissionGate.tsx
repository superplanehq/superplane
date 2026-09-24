import React from "react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { usePermissions } from "@/contexts/usePermissions";
import { PermissionDeniedPage } from "@/components/PermissionDeniedPage";
import { cn } from "@/lib/utils";
import { useWorkspaceLoading } from "@/hooks/useWorkspaceLoading";
import { WORKSPACE_LOADING_COPY } from "@/lib/workspaceLoadingCopy";

interface PermissionTooltipProps {
  allowed: boolean;
  message: string;
  children: React.ReactNode;
  className?: string;
}

export function PermissionTooltip({ allowed, message, children, className }: PermissionTooltipProps) {
  if (allowed) return <>{children}</>;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div className={cn("inline-flex", className)}>
          <div className="pointer-events-none opacity-60 w-full">{children}</div>
        </div>
      </TooltipTrigger>
      <TooltipContent side="top">{message}</TooltipContent>
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

export function RequirePermission({ resource, action, children }: RequirePermissionProps) {
  const { canAct, isLoading } = usePermissions();

  if (isLoading) {
    return <PermissionLoading />;
  }

  if (!canAct(resource, action)) {
    return <PermissionDeniedPage resource={resource} action={action} />;
  }

  return <>{children}</>;
}

interface RequireAnyPermissionProps {
  checks: Array<{ resource: string; action: string }>;
  children: React.ReactNode;
}

export function RequireAnyPermission({ checks, children }: RequireAnyPermissionProps) {
  const { canAct, isLoading } = usePermissions();

  if (isLoading) {
    return <PermissionLoading />;
  }

  if (!checks.some((check) => canAct(check.resource, check.action))) {
    return <PermissionDeniedPage />;
  }

  return <>{children}</>;
}
