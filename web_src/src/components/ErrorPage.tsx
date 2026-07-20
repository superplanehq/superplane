import { AlertCircle } from "lucide-react";
import { EmptyState } from "@/ui/emptyState";
import { Button } from "@/components/ui/button";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";

export function ErrorPage() {
  const handleTryAgain = () => {
    window.location.reload();
  };

  const handleGoHome = () => {
    window.location.href = "/";
  };

  return (
    <div
      className={cn("flex min-h-screen flex-col items-center justify-center bg-gray-50", appDarkModeClasses.surface)}
    >
      <EmptyState icon={AlertCircle} title="Something went wrong" description="We encountered an unexpected error." />
      <div className="flex gap-2 mt-6">
        <Button onClick={handleTryAgain}>Try Again</Button>
        <Button variant="outline" onClick={handleGoHome}>
          Go Home
        </Button>
      </div>
    </div>
  );
}
