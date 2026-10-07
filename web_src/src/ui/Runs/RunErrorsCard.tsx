import { AlertTriangle } from "lucide-react";
import { cn } from "@/lib/utils";

export function RunErrorsCard({ errors, className }: { errors: string[]; className?: string }) {
  if (errors.length === 0) {
    return null;
  }

  const title = errors.length === 1 ? "This run has an error" : "This run has errors";

  return (
    <div
      role="alert"
      data-testid="run-errors-card"
      className={cn(
        "rounded-md border-2 border-red-400 bg-red-50 px-3 py-3 text-red-800 dark:border-red-700 dark:bg-red-950/50 dark:text-red-200",
        className,
      )}
    >
      <div className="flex items-start gap-2">
        <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-red-600 dark:text-red-300" aria-hidden />
        <div className="min-w-0 flex-1">
          <p className="text-sm font-semibold">{title}</p>
          {errors.length === 1 ? (
            <p className="mt-1 break-words text-[13px]">{errors[0]}</p>
          ) : (
            <ol className="mt-1.5 list-decimal space-y-1.5 pl-4 text-[13px]">
              {errors.map((message, index) => (
                <li key={`${index}-${message}`} className="break-words">
                  {message}
                </li>
              ))}
            </ol>
          )}
        </div>
      </div>
    </div>
  );
}
