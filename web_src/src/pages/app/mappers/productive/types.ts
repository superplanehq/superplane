/** Shapes delivered by Productive.io webhooks and read on productive.onTask nodes. */

export interface ProductiveTaskAttributes {
  title?: string;
  description?: string;
  task_number?: number;
  closed?: boolean;
  created_at?: string;
  updated_at?: string;
  due_date?: string;
}

/** A JSON:API resource, trimmed to the fields the trigger reads. */
export interface ProductiveResource<TAttributes = Record<string, unknown>> {
  id?: string;
  type?: string;
  attributes?: TAttributes;
}

export type ProductiveTask = ProductiveResource<ProductiveTaskAttributes>;

/** Envelope Productive.io POSTs to the webhook URL. */
export interface ProductiveWebhookEvent {
  meta?: { event?: string };
  data?: ProductiveTask;
  included?: ProductiveResource[];
}

export interface ProductiveProject {
  id?: string;
  name?: string;
}

/** Metadata SuperPlane stores on Productive.io nodes during setup. */
export interface ProductiveNodeMetadata {
  project?: ProductiveProject;
}

export interface OnTaskConfiguration {
  project?: string;
  actions?: string[];
}
