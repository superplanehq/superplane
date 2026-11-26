import { WEBHOOK_INTAKE_FIELDS_COPY } from "./webhookIntakeSetupCopy";

export interface WebhookIntakeFieldPaths {
  title: string;
  description: string;
  link: string;
}

export type WebhookIntakeFieldPathErrors = Partial<Record<keyof WebhookIntakeFieldPaths, string>>;

export const DEFAULT_WEBHOOK_INTAKE_FIELD_PATHS: WebhookIntakeFieldPaths = {
  title: "title",
  description: "description",
  link: "url",
};

// Must match validWebhookIntakePath in pkg/grpc/actions/factories/webhook_intake.go.
const FIELD_PATH = /^[A-Za-z0-9_-]+(\.[A-Za-z0-9_-]+)*$/;
const FIELD_PATH_MAX_LENGTH = 200;

export function validWebhookIntakeFieldPath(path: string): boolean {
  const value = path.trim();
  return value.length > 0 && value.length <= FIELD_PATH_MAX_LENGTH && FIELD_PATH.test(value);
}

const EXAMPLE_VALUES: WebhookIntakeFieldPaths = {
  title: "Checkout fails for saved cards",
  description: "Customers see an error when they pay with a saved card.",
  link: "https://tracker.example.com/items/42",
};

export function webhookIntakeFieldPathErrors(paths: WebhookIntakeFieldPaths): WebhookIntakeFieldPathErrors {
  const errors: WebhookIntakeFieldPathErrors = {};
  for (const field of ["title", "description", "link"] as const) {
    const path = paths[field].trim();
    if (!path) {
      if (field === "title") errors.title = WEBHOOK_INTAKE_FIELDS_COPY.titleRequired;
      continue;
    }
    if (!validWebhookIntakeFieldPath(path)) {
      errors[field] = WEBHOOK_INTAKE_FIELDS_COPY.pathInvalid;
    }
  }
  return errors;
}

/** A request body that fills every mapped field, so the user can check the paths. */
export function webhookIntakeExampleBody(paths: WebhookIntakeFieldPaths): string {
  // Objects without a prototype, so a segment such as __proto__ stays an own key.
  const body: Record<string, unknown> = Object.create(null);
  for (const field of ["title", "description", "link"] as const) {
    const path = paths[field].trim();
    if (!path || !FIELD_PATH.test(path)) continue;
    const segments = path.split(".");
    let parent = body;
    for (const segment of segments.slice(0, -1)) {
      const child = parent[segment];
      parent[segment] = child && typeof child === "object" ? child : Object.create(null);
      parent = parent[segment] as Record<string, unknown>;
    }
    parent[segments[segments.length - 1]] = EXAMPLE_VALUES[field];
  }
  return JSON.stringify(body, null, 2);
}
