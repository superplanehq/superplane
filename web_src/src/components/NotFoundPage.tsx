import { FileQuestion } from "lucide-react";
import { Button } from "@/components/ui/button";

interface NotFoundPageProps {
  title?: string;
  description?: string;
}

export function NotFoundPage({
  title = "404",
  description = "The page you’re looking for doesn’t exist.",
}: NotFoundPageProps) {
  const handleGoHome = () => {
    window.location.href = "/";
  };

  return (
    <div className="flex flex-col items-center justify-center min-h-[60vh] text-center px-6">
      <div className="flex items-center justify-center w-12 h-12 rounded-md bg-orange-100 text-yellow-700">
        <FileQuestion className="w-5 h-5" />
      </div>
      <h1 className="mt-4 text-3xl font-semibold text-gray-800">{title}</h1>
      <p className="mt-2 text-sm text-gray-500 max-w-md">{description}</p>
      <Button variant="outline" className="mt-6" onClick={handleGoHome}>
        Go Home
      </Button>
    </div>
  );
}
