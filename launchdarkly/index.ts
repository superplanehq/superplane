/**
 * # SuperPlane × LaunchDarkly Integration
 *
 * Provides a set of triggers and actions for interacting with the
 * [LaunchDarkly](https://launchdarkly.com) feature-flag management platform.
 *
 * ## Components
 *
 * | Name                    | Type    | Description                                           |
 * |-------------------------|---------|-------------------------------------------------------|
 * | On Feature Flag Change  | Trigger | Webhook trigger for flag create / update / delete events |
 * | Get Feature Flag        | Action  | Fetch details of a specific feature flag (GET)        |
 * | Delete Feature Flag     | Action  | Permanently remove a feature flag (DELETE)            |
 *
 * ## Authentication
 *
 * All components share a single connection that authenticates via a
 * LaunchDarkly **API access token** (personal or service token).
 *
 * @see {@link https://launchdarkly.com/docs/home/account/api | LaunchDarkly – API Access Tokens}
 * @packageDocumentation
 */

import { createIntegration } from '@superplane/framework';

// ── Connection ──────────────────────────────────────────────────────────────
export {
  launchDarklyConnection,
  buildAuthHeaders,
  LD_API_BASE_URL,
  type LaunchDarklyConnectionProps,
  type LaunchDarklyAuthHeaders,
} from './auth';

// ── Triggers ────────────────────────────────────────────────────────────────
export {
  onFlagChange,
  type OnFlagChangeProps,
  type OnFlagChangeOutput,
  type LaunchDarklyWebhookPayload,
  type LaunchDarklyFlagEventKind,
} from './triggers/on-flag-change';

// ── Actions ─────────────────────────────────────────────────────────────────
export {
  getFlag,
  type GetFlagProps,
  type GetFlagOutput,
  type FeatureFlag,
  type FlagVariation,
} from './actions/get-flag';

export {
  deleteFlag,
  type DeleteFlagProps,
  type DeleteFlagOutput,
} from './actions/delete-flag';

// ── Integration re-exports (for convenience) ────────────────────────────────
import { launchDarklyConnection } from './auth';
import { onFlagChange } from './triggers/on-flag-change';
import { getFlag } from './actions/get-flag';
import { deleteFlag } from './actions/delete-flag';

// ---------------------------------------------------------------------------
// Integration definition
// ---------------------------------------------------------------------------

/**
 * The top-level SuperPlane integration for LaunchDarkly.
 *
 * Register this object with the SuperPlane runtime to expose all included
 * triggers and actions to end-users.
 */
export const launchDarklyIntegration = createIntegration({
  name: 'LaunchDarkly',
  description:
    'Manage and react to feature-flag changes in LaunchDarkly.',
  logoUrl: 'https://app.launchdarkly.com/favicon.ico',

  connection: launchDarklyConnection,

  triggers: [onFlagChange],

  actions: [getFlag, deleteFlag],
});

export default launchDarklyIntegration;
