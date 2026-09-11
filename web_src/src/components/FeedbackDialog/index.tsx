import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import {
  FeedbackRequestError,
  isAllowedFeedbackAttachment,
  submitFeedback,
  type FeedbackCategory,
} from "@/lib/submitFeedback";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { Bug, Lightbulb, MessageSquare, Paperclip, X } from "lucide-react";
import { useEffect, useId, useRef, useState, type ChangeEvent, type FormEvent } from "react";
import { useLocation } from "react-router";

const CATEGORY_OPTIONS: Array<{
  value: FeedbackCategory;
  label: string;
  description: string;
  Icon: typeof Bug;
}> = [
  {
    value: "bug",
    label: "Report a bug",
    description: "Something does not work as expected.",
    Icon: Bug,
  },
  {
    value: "feature",
    label: "Request a feature",
    description: "Suggest a new capability or improvement.",
    Icon: Lightbulb,
  },
  {
    value: "other",
    label: "Something else",
    description: "Ask a question or share other feedback.",
    Icon: MessageSquare,
  },
];

interface FeedbackDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId?: string;
  initialCategory?: FeedbackCategory;
}

export function FeedbackDialog({ open, onOpenChange, organizationId, initialCategory }: FeedbackDialogProps) {
  const location = useLocation();
  const detailsId = useId();
  const [category, setCategory] = useState<FeedbackCategory | undefined>(initialCategory);
  const [details, setDetails] = useState("");
  const [file, setFile] = useState<File | undefined>();
  const [fileError, setFileError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (!open) {
      return;
    }
    setCategory(initialCategory);
    setDetails("");
    setFile(undefined);
    setFileError("");
    setIsSubmitting(false);
  }, [open, initialCategory]);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!organizationId || !category || !details.trim() || isSubmitting) {
      return;
    }

    setIsSubmitting(true);
    try {
      await submitFeedback({
        organizationId,
        category,
        details: details.trim(),
        pagePath: `${location.pathname}${location.search}`,
        file,
      });
      showSuccessToast("Feedback sent.");
      onOpenChange(false);
    } catch (error) {
      const message =
        error instanceof FeedbackRequestError ? error.message : "SuperPlane could not send your feedback. Try again.";
      showErrorToast(message);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl" showCloseButton={!isSubmitting}>
        <DialogHeader>
          <DialogTitle>Send feedback</DialogTitle>
          <DialogDescription>Choose a category and describe the issue or idea.</DialogDescription>
        </DialogHeader>
        <form className="space-y-4" onSubmit={(event) => void handleSubmit(event)}>
          <FeedbackCategoryPicker category={category} onChange={setCategory} disabled={isSubmitting} />
          <div className="space-y-2">
            <Label htmlFor={detailsId}>Details</Label>
            <Textarea
              id={detailsId}
              value={details}
              onChange={(event) => setDetails(event.target.value)}
              placeholder="Describe the problem or idea."
              disabled={isSubmitting}
              data-testid="feedback-details"
              className="min-h-28"
            />
          </div>
          <FeedbackAttachmentField
            file={file}
            fileError={fileError}
            disabled={isSubmitting}
            onFileChange={setFile}
            onFileError={setFileError}
          />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={isSubmitting}>
              Cancel
            </Button>
            <LoadingButton
              type="submit"
              disabled={!organizationId || !category || !details.trim()}
              loading={isSubmitting}
              loadingText="Sending..."
              data-testid="feedback-send"
            >
              Send
            </LoadingButton>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function FeedbackCategoryPicker({
  category,
  onChange,
  disabled,
}: {
  category?: FeedbackCategory;
  onChange: (category: FeedbackCategory) => void;
  disabled: boolean;
}) {
  return (
    <fieldset className="space-y-2" disabled={disabled}>
      <legend className="text-sm font-medium text-gray-800 dark:text-gray-100">Category</legend>
      <div className="grid gap-2 sm:grid-cols-3">
        {CATEGORY_OPTIONS.map((option) => {
          const selected = category === option.value;
          const OptionIcon = option.Icon;
          return (
            <Button
              key={option.value}
              type="button"
              variant="outline"
              aria-pressed={selected}
              disabled={disabled}
              data-testid={`feedback-category-${option.value}`}
              onClick={() => onChange(option.value)}
              className={cn(
                "h-auto w-full flex-col items-start gap-1 rounded-md px-3 py-2 text-left whitespace-normal font-normal",
                selected
                  ? "border-sky-500 bg-sky-50 dark:border-sky-400 dark:bg-sky-950/40"
                  : "border-slate-950/20 hover:bg-slate-50 dark:border-gray-700/70 dark:hover:bg-gray-800",
              )}
            >
              <OptionIcon className="h-4 w-4 text-gray-600 dark:text-gray-300" aria-hidden />
              <span className="text-sm font-medium text-gray-800 dark:text-gray-100">{option.label}</span>
              <span className="text-[13px] text-gray-500 dark:text-gray-400">{option.description}</span>
            </Button>
          );
        })}
      </div>
    </fieldset>
  );
}

function FeedbackAttachmentField({
  file,
  fileError,
  disabled,
  onFileChange,
  onFileError,
}: {
  file?: File;
  fileError: string;
  disabled: boolean;
  onFileChange: (file: File | undefined) => void;
  onFileError: (message: string) => void;
}) {
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const handleFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    const nextFile = event.target.files?.[0];
    event.target.value = "";
    if (!nextFile) {
      return;
    }
    if (!isAllowedFeedbackAttachment(nextFile)) {
      onFileError("Attach a PNG, JPEG, GIF, WebP, PDF, or text file that is 5 MB or smaller.");
      return;
    }
    onFileError("");
    onFileChange(nextFile);
  };

  return (
    <div className="space-y-2">
      <Input
        ref={fileInputRef}
        type="file"
        accept="image/png,image/jpeg,image/gif,image/webp,application/pdf,text/plain,text/markdown,.md"
        className="sr-only"
        onChange={handleFileChange}
        data-testid="feedback-file-input"
      />
      {file ? (
        <div className="flex items-center gap-2 rounded-md border border-slate-950/20 px-3 py-2 text-sm dark:border-gray-700/70">
          <Paperclip className="h-4 w-4 shrink-0 text-gray-500" aria-hidden />
          <span className="min-w-0 flex-1 truncate text-gray-800 dark:text-gray-100">{file.name}</span>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            className="rounded-md text-gray-500 hover:bg-slate-100 hover:text-gray-800 dark:hover:bg-gray-800 dark:hover:text-gray-100"
            onClick={() => onFileChange(undefined)}
            disabled={disabled}
            aria-label="Remove attached file"
          >
            <X className="h-4 w-4" aria-hidden />
          </Button>
        </div>
      ) : (
        <Button
          type="button"
          variant="outline"
          onClick={() => fileInputRef.current?.click()}
          disabled={disabled}
          data-testid="feedback-attach-file"
        >
          <Paperclip className="h-4 w-4" aria-hidden />
          Attach a file
        </Button>
      )}
      {fileError ? (
        <p className="text-sm text-red-600 dark:text-red-400" role="alert">
          {fileError}
        </p>
      ) : null}
    </div>
  );
}
