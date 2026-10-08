# Bitbucket Forge synchronization

SuperPlane records installations from authenticated Forge lifecycle and
scheduled deliveries. Onboarding checks each recorded installation for the
connected Bitbucket account's workspace and repository access.

The scheduled remote trigger runs every five minutes. It supplies a fresh
system token and recreates a missing installation record after a successful
delivery. Keep the configured remote URL reachable. A failed delivery delays
discovery until a later successful delivery.

Installation records remain in PostgreSQL when a token expires. Uninstall
clears credentials and marks the installation as removed. Delayed deliveries
cannot restore an installation after uninstall.

## Activate the schedule

1. Confirm the manifest's remote URL points to the intended SuperPlane instance.
2. Run these commands from `forge/bitbucket/`:

   ```sh
   forge lint
   forge deploy --environment production
   forge install list
   ```

3. If Forge creates a major version, upgrade the existing installations before testing the new schedule.
4. Keep SuperPlane and its public tunnel running during local testing.
5. Connect Bitbucket in onboarding. Leave the page open while the installation synchronizes.

Changing a scheduled trigger resets its schedule. Forge starts invocations
approximately five minutes after deployment. Delivery timing can vary.
See the [scheduled trigger documentation](https://developer.atlassian.com/platform/forge/manifest-reference/modules/scheduled-trigger/).

## Monitor deliveries

With OpenTelemetry metrics enabled, SuperPlane exports:

- `bitbucket.forge.deliveries`, with `outcome` values `accepted`, `rejected`,
  and `failed`. `failed` means encryption or database storage failed.
  `rejected` means configuration, authentication, or body validation rejected the request.
- `bitbucket.forge.installations.stale`: the number of installations with
  missing or expired credentials, or no delivery for more than fifteen minutes.
  Removed installations do not count. The gauge returns to zero after recovery.

Configure an alert when the stale gauge stays above zero or failed deliveries
increase. Existing structured logs include the installation ID for encryption
and storage failures. Credentials never appear in these metrics or logs.

Monitoring covers installations already recorded by SuperPlane. It cannot
identify an installation that has never delivered to this database.
