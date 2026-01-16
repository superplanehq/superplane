import { AlertCircle } from "lucide-react";
import { EmptyState } from "@/ui/emptyState";
import { Button } from "@/components/ui/button";

export function ErrorPage() {
  const handleTryAgain = () => {
    window.location.reload();
  };

  const handleGoHome = () => {
    window.location.href = "/";
  };

  return (
    <div className="flex flex-col justify-center items-center min-h-screen bg-gray-50">
      <EmptyState
        icon={AlertCircle}
        title="Something went wrong"
        description="We encountered an unexpected error. Our team has been notified and is working on it."
      />
      <div className="flex gap-2 mt-6">
        <Button onClick={handleTryAgain}>Try Again</Button>
        <Button variant="outline" onClick={handleGoHome}>
          Go Home
        </Button>
      </div>
    </div>
  );
}
