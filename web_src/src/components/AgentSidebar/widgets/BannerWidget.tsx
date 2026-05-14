import { CheckCircle2, XCircle } from "lucide-react";
import { cn } from "@/lib/utils";

interface BannerWidgetProps {
  variant: "success" | "error";
  content: string;
}

export function BannerWidget({ variant, content }: BannerWidgetProps) {
  const isSuccess = variant === "success";
  return (
    <div
      className={cn(
        "my-4 flex items-start gap-2 rounded-lg px-3 py-2 text-sm",
        isSuccess
          ? "bg-green-50 border border-green-200 text-green-900"
          : "bg-red-50 border border-red-200 text-red-900",
      )}
    >
      {isSuccess ? (
        <CheckCircle2 className="size-4 text-green-600 shrink-0 mt-0.5" />
      ) : (
        <XCircle className="size-4 text-red-600 shrink-0 mt-0.5" />
      )}
      <p className="text-xs leading-relaxed">{content}</p>
    </div>
  );
}
