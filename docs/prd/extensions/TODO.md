# Next Steps

1. Implement the server-side use of uploaded extension bundles.
   Current API calls store extension version metadata, but the runtime-side bundle storage, validation, and execution path still needs to be finished in the main product.

2. Finalize how the server consumes `manifest.json`.
   The CLI already produces `dist/manifest.json`, but the public API now only accepts the tar-gzipped bundle and its digest.

3. Replace manifest parsing DTOs with engine-native types.
   The current server-side bundle processing uses temporary manifest-specific structs for integrations, components, triggers, actions, output channels, and configuration fields.
   We should keep the explicit manifest parsing layer, but convert the stored representation to engine-native types such as `configuration.Field`, `core.Action`, and `core.OutputChannel`.

4. Endpoints exposing available components, triggers and integrations should include the ones coming from extensions.

5. Implement the runtime execution plane.
   The SDK worker protocol exists, but production runtime orchestration, artifact download, worker lifecycle, and sandbox strategy are still open work.

6. Decide whether the watch mode should stay intentionally shallow or become recursive.
   The current CLI watches the entrypoint directory plus `integrations/`, `components/`, and `triggers/`.

7. Start the sandbox-provider abstraction.
   Once the invocation contract is stable, define the provider boundary for:
   - artifact-backed extension startup
   - outbound control-channel lifecycle
   - Cloud Run as the first backend
   - future custom Kubernetes backends

8. Public registry of extensions
  What we have right now is the equivalent of a private extension.
  However, it would be really good to have a public registry of extensions.
  Once we have that, we could even start focusing on implementing new integrations
  through extensions available in that registry instead of as part of SuperPlane itself.
