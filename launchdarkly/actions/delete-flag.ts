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
 * Input properties for the "Delete Feature Flag" action.
 */
export interface DeleteFlagProps {
  /**
   * The key of the LaunchDarkly project that owns the flag.
   *
   * @example "my-application"
   */
  projectKey: string;

  /**
   * The unique key of the feature flag to delete.
   *
   * @example "enable-dark-mode"
   */
  flagKey: string;
}

/**
 * Output of the "Delete Feature Flag" action.
 */
export interface DeleteFlagOutput {
  /** `true` when the flag was successfully deleted. */
  deleted: boolean;

  /** A human-readable status message. */
  message: string;

  /** HTTP status code returned by the API. */
  statusCode: number;
}

// ---------------------------------------------------------------------------
// Action definition
// ---------------------------------------------------------------------------

/**
 * **Delete Feature Flag** – REST API Action (DELETE)
 *
 * Permanently deletes a LaunchDarkly feature flag in **all environments**.
 *
 * > ⚠️ **Use with caution** – this action cannot be undone. Only delete
 * > flags that your application no longer references.
 *
 * A `404` response is handled gracefully, returning `deleted: false` with
 * a descriptive message rather than throwing.
 *
 * @see {@link https://launchdarkly.com/docs/api/feature-flags/delete-feature-flag | API – Delete Feature Flag}
 */
export const deleteFlag = createAction<
  DeleteFlagProps,
  DeleteFlagOutput,
  LaunchDarklyConnectionProps
>({
  name: 'Delete Feature Flag',
  description:
    'Permanently delete a LaunchDarkly feature flag from all environments. Use with caution.',
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
        'The unique key of the feature flag to delete.',
      required: true,
    },
  },

  // ── Action handler ────────────────────────────────────────────────────
  /**
   * Execute the DELETE request against the LaunchDarkly API.
   *
   * @param ctx - Action context provided by the SuperPlane runtime,
   *              including the resolved connection and user-supplied props.
   * @returns A result indicating whether the flag was successfully deleted.
   */
  async run(
    ctx: ActionContext<DeleteFlagProps, LaunchDarklyConnectionProps>,
  ): Promise<DeleteFlagOutput> {
    const { projectKey, flagKey } = ctx.props;

    // Validate required inputs
    if (!projectKey?.trim()) {
      throw new Error(
        '[LaunchDarkly:deleteFlag] "projectKey" is required and cannot be empty.',
      );
    }
    if (!flagKey?.trim()) {
      throw new Error(
        '[LaunchDarkly:deleteFlag] "flagKey" is required and cannot be empty.',
      );
    }

    const url = `${LD_API_BASE_URL}/flags/${encodeURIComponent(projectKey)}/${encodeURIComponent(flagKey)}`;
    const headers = buildAuthHeaders(ctx.connection);

    console.debug(
      `[LaunchDarkly:deleteFlag] Deleting flag "${flagKey}" from project "${projectKey}"`,
    );

    try {
      const response = await fetch(url, {
        method: 'DELETE',
        headers,
      });

      // ── 204 – successful deletion (no content) ─────────────────────
      if (response.status === 204) {
        console.info(
          `[LaunchDarkly:deleteFlag] Flag "${flagKey}" successfully deleted (204).`,
        );
        return {
          deleted: true,
          message: `Feature flag "${flagKey}" was successfully deleted from project "${projectKey}".`,
          statusCode: 204,
        };
      }

      // ── 404 – flag does not exist ──────────────────────────────────
      if (response.status === 404) {
        console.info(
          `[LaunchDarkly:deleteFlag] Flag "${flagKey}" not found (404).`,
        );
        return {
          deleted: false,
          message: `Feature flag "${flagKey}" was not found in project "${projectKey}". It may have already been deleted.`,
          statusCode: 404,
        };
      }

      // ── 409 – conflict (e.g. flag is in use by an experiment) ──────
      if (response.status === 409) {
        const errorBody = await response.text();
        throw new Error(
          `Cannot delete flag "${flagKey}": conflict (409). ` +
            `The flag may be in use by an active experiment or approval request. ` +
            `Details: ${errorBody}`,
        );
      }

      // ── Other non-success statuses ─────────────────────────────────
      if (!response.ok) {
        const errorBody = await response.text();
        throw new Error(
          `LaunchDarkly API responded with status ${response.status}: ${errorBody}`,
        );
      }

      // ── Unexpected success status (e.g. 200) – still treat as OK ───
      return {
        deleted: true,
        message: `Feature flag "${flagKey}" deletion returned status ${response.status}.`,
        statusCode: response.status,
      };
    } catch (error: unknown) {
      // Re-throw errors we already created above
      if (
        error instanceof Error &&
        (error.message.startsWith('LaunchDarkly API') ||
          error.message.startsWith('Cannot delete flag'))
      ) {
        throw error;
      }

      const message =
        error instanceof Error ? error.message : String(error);
      throw new Error(
        `[LaunchDarkly:deleteFlag] Request failed: ${message}`,
      );
    }
  },
});
