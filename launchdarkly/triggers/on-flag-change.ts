import { createTrigger, TriggerHookContext } from '@superplane/framework';
import {
  launchDarklyConnection,
  type LaunchDarklyConnectionProps,
} from '../auth';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/**
 * Known LaunchDarkly webhook event types related to feature flags.
 *
 * @see {@link https://launchdarkly.com/docs/home/flags | LaunchDarkly – Feature Flags}
 */
export type LaunchDarklyFlagEventKind =
  | 'flag.created'
  | 'flag.updated'
  | 'flag.deleted'
  | 'flag.archived'
  | 'flag.restored';

/**
 * Input properties for the "On Feature Flag Change" trigger.
 */
export interface OnFlagChangeProps {
  /**
   * Optional list of event types to listen for.
   *
   * When omitted **all** flag-related events are forwarded.
   * Provide one or more of `flag.created`, `flag.updated`, `flag.deleted`,
   * `flag.archived`, or `flag.restored` to narrow the stream.
   */
  eventTypes?: LaunchDarklyFlagEventKind[];
}

/**
 * Shape of the webhook payload delivered by LaunchDarkly.
 *
 * @remarks
 * Only the most commonly-used top-level fields are typed here.
 * The full payload may contain additional properties depending on
 * the account configuration and event kind.
 */
export interface LaunchDarklyWebhookPayload {
  /** Unique identifier for this webhook delivery. */
  _id: string;

  /** ISO-8601 timestamp of the event. */
  date: number;

  /** The kind / type of the event (e.g. `flag.updated`). */
  kind: string;

  /** Human-readable summary of the change. */
  name: string;

  /** Description of the change. */
  description: string;

  /** Short, human-readable title. */
  shortDescription: string;

  /** The member who triggered the change (if applicable). */
  member?: {
    _id: string;
    email: string;
    firstName?: string;
    lastName?: string;
  };

  /** Title summary of the change suitable for display. */
  titleVerb?: string;

  /** Title of the resource that was changed. */
  title?: string;

  /** The target resource URL within LaunchDarkly. */
  target?: {
    _links?: Record<string, { href: string; type?: string }>;
    name?: string;
    resources?: string[];
  };

  /** Additional properties present on the event. */
  [key: string]: unknown;
}

/**
 * Output emitted by the trigger for each matching event.
 */
export interface OnFlagChangeOutput {
  /** The raw webhook payload from LaunchDarkly. */
  payload: LaunchDarklyWebhookPayload;

  /** The event kind extracted for convenience. */
  eventKind: string;
}

// ---------------------------------------------------------------------------
// Trigger definition
// ---------------------------------------------------------------------------

/**
 * **On Feature Flag Change** – Webhook Trigger
 *
 * Fires whenever LaunchDarkly delivers a webhook event for a feature-flag
 * change. Supports optional filtering by one or more event types so that
 * downstream steps only execute for the changes you care about.
 *
 * ### Setup
 * 1. In LaunchDarkly → **Integrations → Webhooks**, create a new webhook
 *    pointing at the URL provided by SuperPlane for this trigger.
 * 2. Optionally configure a webhook signing secret for payload verification.
 *
 * @see {@link https://launchdarkly.com/docs/home/account/webhooks | LaunchDarkly – Webhooks}
 */
export const onFlagChange = createTrigger<
  OnFlagChangeProps,
  OnFlagChangeOutput,
  LaunchDarklyConnectionProps
>({
  name: 'On Feature Flag Change',
  description:
    'Triggers when a LaunchDarkly feature flag is created, updated, deleted, archived, or restored.',
  connection: launchDarklyConnection,

  // ── Prop / input definitions ──────────────────────────────────────────
  props: {
    eventTypes: {
      type: 'array',
      items: { type: 'string' },
      label: 'Event Types',
      description:
        'Optionally filter by specific event types (e.g. flag.updated, flag.deleted). ' +
        'Leave empty to receive all flag-related events.',
      required: false,
      default: [],
    },
  },

  // ── Trigger type ──────────────────────────────────────────────────────
  type: 'webhook',

  // ── Webhook handler ───────────────────────────────────────────────────
  /**
   * Processes an incoming webhook request from LaunchDarkly.
   *
   * @param ctx - The trigger hook context provided by the SuperPlane runtime,
   *              containing the raw HTTP request body and the user's props.
   * @returns The parsed & optionally filtered trigger output, or `undefined`
   *          if the event should be skipped.
   */
  async onWebhook(
    ctx: TriggerHookContext<OnFlagChangeProps>,
  ): Promise<OnFlagChangeOutput | undefined> {
    try {
      const payload = ctx.body as LaunchDarklyWebhookPayload;

      if (!payload || !payload.kind) {
        console.warn(
          '[LaunchDarkly:onFlagChange] Received webhook with no "kind" field – skipping.',
        );
        return undefined;
      }

      // ── Optional event-type filter ──────────────────────────────────
      const allowedTypes = ctx.props.eventTypes;

      if (allowedTypes && allowedTypes.length > 0) {
        const isAllowed = allowedTypes.some(
          (type) => type === payload.kind,
        );

        if (!isAllowed) {
          console.info(
            `[LaunchDarkly:onFlagChange] Event "${payload.kind}" does not match ` +
              `allowed types [${allowedTypes.join(', ')}] – skipping.`,
          );
          return undefined;
        }
      }

      return {
        payload,
        eventKind: payload.kind,
      };
    } catch (error: unknown) {
      const message =
        error instanceof Error ? error.message : String(error);
      throw new Error(
        `[LaunchDarkly:onFlagChange] Failed to process webhook: ${message}`,
      );
    }
  },
});
