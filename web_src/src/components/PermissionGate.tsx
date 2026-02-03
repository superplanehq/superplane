import React from "react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { usePermissions } from "@/contexts/PermissionsContext";
import { NotFoundPage } from "@/components/NotFoundPage";
import { cn } from "@/lib/utils";

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

export function RequirePermission({ resource, action, children }: RequirePermissionProps) {
  const { canAct, isLoading } = usePermissions();

  if (isLoading) {
    return (
      <div className="flex justify-center items-center min-h-[40vh]">
        <p className="text-gray-500">Checking permissions...</p>
      </div>
    );
  }

  if (!canAct(resource, action)) {
    return <NotFoundPage />;
  }

  return <>{children}</>;
}
