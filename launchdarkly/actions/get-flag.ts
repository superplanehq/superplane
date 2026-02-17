import { createAction, ActionContext } from '@superplane/framework';
import {
  launchDarklyConnection,
  buildAuthHeaders,
  LD_API_BASE_URL,
  type LaunchDarklyConnectionProps,
} from '../auth';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/**
 * Input properties for the "Get Feature Flag" action.
 */
export interface GetFlagProps {
  /**
   * The key of the LaunchDarkly project that owns the flag.
   *
   * @example "my-application"
   */
  projectKey: string;

  /**
   * The unique key of the feature flag to retrieve.
   *
   * @example "enable-dark-mode"
   */
  flagKey: string;
}

/**
 * Variation value within a feature flag.
 */
export interface FlagVariation {
  _id?: string;
  value: unknown;
  name?: string;
  description?: string;
}

/**
 * Simplified representation of a LaunchDarkly feature flag returned by the
 * REST API.
 *
 * @remarks
 * Only the most commonly-used fields are typed here. The full response may
 * contain many more properties (environment configs, experiments, etc.).
 *
 * @see {@link https://launchdarkly.com/docs/api/feature-flags/get-feature-flag | API – Get Feature Flag}
 */
export interface FeatureFlag {
  /** Human-readable name. */
  name: string;

  /** Unique key used to reference the flag in code. */
  key: string;

  /** The kind of flag (`boolean` or `multivariate`). */
  kind: 'boolean' | 'multivariate';

  /** Internal version counter. */
  _version: number;

  /** Epoch-millisecond timestamp of when the flag was created. */
  creationDate: number;

  /** Whether the flag is temporary. */
  temporary: boolean;

  /** Tags assigned to the flag. */
  tags: string[];

  /** An array of possible variations for the flag. */
  variations: FlagVariation[];

  /** Description of the feature flag. */
  description?: string;

  /** Whether the flag has been archived. */
  archived: boolean;

  /** Whether the flag has been deprecated. */
  deprecated?: boolean;

  /** HAL-style links. */
  _links: Record<string, { href: string; type?: string }>;

  /** Per-environment configuration data (when requested). */
  environments?: Record<string, unknown>;

  /** Catch-all for additional API fields. */
  [key: string]: unknown;
}

/**
 * Output of the "Get Feature Flag" action.
 */
export interface GetFlagOutput {
  /** `true` when the flag was found successfully. */
  found: boolean;

  /** The feature flag data (present when `found` is `true`). */
  flag?: FeatureFlag;

  /** A human-readable status message. */
  message: string;
}

// ---------------------------------------------------------------------------
// Action definition
// ---------------------------------------------------------------------------

/**
 * **Get Feature Flag** – REST API Action (GET)
 *
 * Fetches the details of a single LaunchDarkly feature flag identified by
 * its project key and flag key.
 *
 * A `404` response is handled gracefully – the action will succeed with
 * `found: false` rather than throwing, making it safe to use in conditional
 * workflows.
 *
 * @see {@link https://launchdarkly.com/docs/api/feature-flags/get-feature-flag | API – Get Feature Flag}
 */
export const getFlag = createAction<
  GetFlagProps,
  GetFlagOutput,
  LaunchDarklyConnectionProps
>({
  name: 'Get Feature Flag',
  description:
    'Retrieve the details of a specific LaunchDarkly feature flag by project key and flag key.',
  connection: launchDarklyConnection,

  // ── Prop / input definitions ──────────────────────────────────────────
  props: {
    projectKey: {
      type: 'string',
      label: 'Project Key',
      description:
        'The key of the LaunchDarkly project that contains the flag.',
      required: true,
    },
    flagKey: {
      type: 'string',
      label: 'Flag Key',
      description:
        'The unique key of the feature flag to retrieve.',
      required: true,
    },
  },

  // ── Action handler ────────────────────────────────────────────────────
  /**
   * Execute the GET request against the LaunchDarkly API.
   *
   * @param ctx - Action context provided by the SuperPlane runtime,
   *              including the resolved connection and user-supplied props.
   * @returns The fetched feature flag data or a "not found" result.
   */
  async run(
    ctx: ActionContext<GetFlagProps, LaunchDarklyConnectionProps>,
  ): Promise<GetFlagOutput> {
    const { projectKey, flagKey } = ctx.props;

    // Validate required inputs
    if (!projectKey?.trim()) {
      throw new Error(
        '[LaunchDarkly:getFlag] "projectKey" is required and cannot be empty.',
      );
    }
    if (!flagKey?.trim()) {
      throw new Error(
        '[LaunchDarkly:getFlag] "flagKey" is required and cannot be empty.',
      );
    }

    const url = `${LD_API_BASE_URL}/flags/${encodeURIComponent(projectKey)}/${encodeURIComponent(flagKey)}`;
    const headers = buildAuthHeaders(ctx.connection);

    console.debug(
      `[LaunchDarkly:getFlag] Fetching flag "${flagKey}" from project "${projectKey}"`,
    );

    try {
      const response = await fetch(url, {
        method: 'GET',
        headers,
      });

      // ── 404 – flag does not exist ───────────────────────────────────
      if (response.status === 404) {
        console.info(
          `[LaunchDarkly:getFlag] Flag "${flagKey}" not found (404).`,
        );
        return {
          found: false,
          message: `Feature flag "${flagKey}" was not found in project "${projectKey}".`,
        };
      }

      // ── Other non-success statuses ──────────────────────────────────
      if (!response.ok) {
        const errorBody = await response.text();
        throw new Error(
          `LaunchDarkly API responded with status ${response.status}: ${errorBody}`,
        );
      }

      // ── Success ─────────────────────────────────────────────────────
      const flag = (await response.json()) as FeatureFlag;

      return {
        found: true,
        flag,
        message: `Successfully retrieved feature flag "${flag.key}" (${flag.name}).`,
      };
    } catch (error: unknown) {
      // Re-throw errors we already created above
      if (error instanceof Error && error.message.startsWith('LaunchDarkly API')) {
        throw error;
      }

      const message =
        error instanceof Error ? error.message : String(error);
      throw new Error(
        `[LaunchDarkly:getFlag] Request failed: ${message}`,
      );
    }
  },
});
