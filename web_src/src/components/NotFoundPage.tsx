import { Cloud, FileQuestion, Plane } from "lucide-react";
import { Button } from "@/components/ui/button";
import "./NotFoundPage.css";

interface NotFoundPageProps {
  title?: string;
  description?: string;
  actionLabel?: string;
  showFlightAnimation?: boolean;
}

export function NotFoundPage({
  title = "404",
  description = "The page you’re looking for doesn’t exist.",
  actionLabel = "Go Home",
  showFlightAnimation = false,
}: NotFoundPageProps) {
  return (
    <div className="flex flex-col items-center justify-center min-h-[60vh] text-center px-6">
      {showFlightAnimation ? (
        <FlightPathIllustration />
      ) : (
        <div className="flex items-center justify-center w-12 h-12 rounded-md bg-orange-100 text-yellow-700">
          <FileQuestion className="w-5 h-5" />
        </div>
      )}
      <h1 className="mt-4 text-3xl font-semibold text-gray-800">{title}</h1>
      <p className="mt-2 text-sm text-gray-500 max-w-md">{description}</p>
      <Button asChild variant="outline" className="mt-6">
        <a href="/" aria-label="Return to the SuperPlane home page">
          {actionLabel}
        </a>
      </Button>
    </div>
  );
}

function FlightPathIllustration() {
  return (
    <div
      className="relative h-20 w-full max-w-80 overflow-hidden text-orange-500"
      aria-hidden="true"
      data-testid="flight-path-illustration"
    >
      <Cloud className="absolute left-8 top-8 h-5 w-5 text-slate-200" fill="currentColor" />
      <Cloud className="absolute right-7 top-2 h-7 w-7 text-slate-100" fill="currentColor" />
      <svg className="absolute inset-0 h-full w-full" viewBox="0 0 320 80" fill="none">
        <path
          d="M8 58 C72 58 72 20 136 20 C200 20 200 58 312 34"
          className="stroke-slate-300"
          strokeWidth="1.5"
          strokeDasharray="5 7"
          strokeLinecap="round"
        />
      </svg>
      <Plane className="not-found-page__plane absolute left-0 top-8 h-7 w-7 fill-orange-100" />
    </div>
  );
}
