import { ShieldOff } from "lucide-react";
import { Button } from "@/components/ui/button";

interface PermissionDeniedPageProps {
  resource?: string;
  action?: string;
  description?: string;
}

export function PermissionDeniedPage({ resource, action, description }: PermissionDeniedPageProps) {
  const requiredPermission = resource && action ? `${resource}.${action}` : undefined;
  const message =
    description ??
    (requiredPermission
      ? `You do not have permission to open this page. Required permission: ${requiredPermission}.`
      : "You do not have permission to open this page.");

  const handleGoHome = () => {
    window.location.href = "/";
  };

  return (
    <div
      className="flex flex-col items-center justify-center min-h-[60vh] text-center px-6"
      data-testid="permission-denied-page"
    >
      <div className="flex items-center justify-center w-12 h-12 rounded-md bg-orange-100 text-yellow-700">
        <ShieldOff className="w-5 h-5" />
      </div>
      <h1 className="mt-4 text-3xl font-semibold text-gray-800">Permission denied</h1>
      <p className="mt-2 text-sm text-gray-500 max-w-md">{message}</p>
      <Button variant="outline" className="mt-6" onClick={handleGoHome}>
        Go Home
      </Button>
    </div>
  );
}
