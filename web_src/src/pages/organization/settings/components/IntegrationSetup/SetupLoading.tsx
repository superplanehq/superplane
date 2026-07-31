import { Loader2 } from "lucide-react";

export function SetupLoading() {
  return (
    <div className="flex justify-center items-center gap-2 py-16 text-content-secondary">
      <Loader2 className="h-6 w-6 animate-spin" aria-hidden />
      <span className="text-sm">Loading integration metadata...</span>
    </div>
  );
}
