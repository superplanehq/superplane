import React from "react";
import { CheckCircle2, CircleAlert, TriangleAlert } from "lucide-react";
import debounce from "lodash.debounce";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { parseCurl, type ParsedCurlRequest } from "@/lib/parseCurl";

type CurlImportConfig = Partial<
  Pick<ParsedCurlRequest, "method" | "url" | "headers" | "queryParams" | "contentType" | "json" | "formData" | "text">
>;

interface CurlImportFieldProps {
  onApply: (config: CurlImportConfig) => void;
  disabled?: boolean;
}

type ValidationState = {
  tone: "success" | "warning" | "error";
  message: string;
};

export const CurlImportField: React.FC<CurlImportFieldProps> = ({ onApply, disabled = false }) => {
  const [value, setValue] = React.useState("");
  const [validation, setValidation] = React.useState<ValidationState | null>(null);

  const parseAndApply = React.useMemo(
    () =>
      debounce((nextValue: string) => {
        const trimmed = nextValue.trim();
        if (trimmed.length === 0) {
          setValidation(null);
          return;
        }

        const result = parseCurl(trimmed);
        if (!result.success || !result.request) {
          setValidation({
            tone: "error",
            message: result.errors[0] || "Could not parse curl command",
          });
          return;
        }

        if (!disabled) {
          onApply({
            method: result.request.method,
            url: result.request.url,
            headers: result.request.headers,
            queryParams: result.request.queryParams,
            contentType: result.request.contentType,
            json: result.request.json,
            formData: result.request.formData,
            text: result.request.text,
          });
        }

        if (result.warnings.length > 0) {
          setValidation({
            tone: "warning",
            message: `Filled what we could. ${result.warnings.join("; ")}`,
          });
          return;
        }

        setValidation({
          tone: "success",
          message: "Parsed successfully. Review imported values below.",
        });
      }, 300),
    [disabled, onApply],
  );

  React.useEffect(() => {
    return () => {
      parseAndApply.cancel();
    };
  }, [parseAndApply]);

  const handleChange = (event: React.ChangeEvent<HTMLTextAreaElement>) => {
    const nextValue = event.target.value ?? "";
    setValue(nextValue);
    parseAndApply(nextValue);
  };

  const describedById = validation ? "curl-import-feedback" : undefined;

  return (
    <div className="space-y-2">
      <Label htmlFor="curl-import-textarea" className="block text-left">
        Import from curl (optional)
      </Label>
      <Textarea
        id="curl-import-textarea"
        value={value}
        onChange={handleChange}
        placeholder='Paste a curl command, e.g. curl -X POST "https://api.example.com" -H "Content-Type: application/json" -d "{\"foo\":\"bar\"}"'
        className="min-h-28"
        disabled={disabled}
        aria-describedby={describedById}
      />
      {validation && (
        <div
          id="curl-import-feedback"
          className={`flex items-start gap-2 text-xs ${
            validation.tone === "error"
              ? "text-red-500 dark:text-red-400"
              : validation.tone === "warning"
                ? "text-amber-600 dark:text-amber-400"
                : "text-green-600 dark:text-green-400"
          }`}
        >
          {validation.tone === "error" ? (
            <CircleAlert className="h-4 w-4 shrink-0 mt-0.5" />
          ) : validation.tone === "warning" ? (
            <TriangleAlert className="h-4 w-4 shrink-0 mt-0.5" />
          ) : (
            <CheckCircle2 className="h-4 w-4 shrink-0 mt-0.5" />
          )}
          <p>{validation.message}</p>
        </div>
      )}
    </div>
  );
};
