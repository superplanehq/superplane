# Next Steps

1. Cleanup the extensions SDK to only allow component blocks to be used as extensions
  We still want to allow triggers and integrations to be exposed later on,
  but supporting only components to start with will make it easier for us chunk the work ahead.

2. Implement InvocationPayload building in Hub.dispatch_job()

3. Write to `jobs` table in NodeExecutor

4. Review if this still makes sense
  Update the packaging pipeline in `pkg/cli/commands/extensions/create_version.go` so runtime bundles target Deno-compatible ESM instead of Node/CommonJS worker bootstrap code.

5. Public registry of extensions.
   What we have right now is the equivalent of a private extension.
   However, it would be really good to have a public registry of extensions.
   Once we have that, we could even start focusing on implementing new integrations
   through extensions available in that registry instead of as part of SuperPlane itself.

6. Support for integrations and triggers as part of extensions
