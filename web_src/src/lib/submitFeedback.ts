export const FEEDBACK_CATEGORIES = ["bug", "feature", "other"] as const;

export type FeedbackCategory = (typeof FEEDBACK_CATEGORIES)[number];

export const MAX_FEEDBACK_ATTACHMENT_BYTES = 5 * 1024 * 1024;

export const ALLOWED_FEEDBACK_ATTACHMENT_TYPES = [
  "image/png",
  "image/jpeg",
  "image/gif",
  "image/webp",
  "application/pdf",
  "text/plain",
  "text/markdown",
] as const;

export class FeedbackRequestError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "FeedbackRequestError";
  }
}

export function isAllowedFeedbackAttachment(file: File): boolean {
  if (file.size > MAX_FEEDBACK_ATTACHMENT_BYTES) {
    return false;
  }
  return ALLOWED_FEEDBACK_ATTACHMENT_TYPES.includes(file.type as (typeof ALLOWED_FEEDBACK_ATTACHMENT_TYPES)[number]);
}

export async function submitFeedback(params: {
  organizationId: string;
  category: FeedbackCategory;
  details: string;
  pagePath?: string;
  file?: File;
}): Promise<void> {
  const formData = new FormData();
  formData.append("category", params.category);
  formData.append("details", params.details);
  if (params.pagePath) {
    formData.append("page_path", params.pagePath);
  }
  if (params.file) {
    formData.append("file", params.file);
  }

  const response = await fetch("/api/v1/me/feedback", {
    method: "POST",
    headers: {
      "x-organization-id": params.organizationId,
    },
    body: formData,
  });

  if (response.ok) {
    return;
  }

  const payload = (await response.json().catch(() => null)) as { message?: string } | null;
  throw new FeedbackRequestError(payload?.message || "SuperPlane could not send your feedback. Try again.");
}
