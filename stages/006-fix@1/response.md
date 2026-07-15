All checks pass. Let me summarize the fix:

## Summary

**Root cause:** Vite's `import-analysis` plugin resolves module file paths at *transform time* — before Vitest's runtime mock intercepts can intervene. Because `@/api-client/sdk.gen` is a generated file (gitignored, absent in the verify environment), Vite threw `"Failed to resolve import"` when transforming any module that imports from it (`useIntegrations.ts`, `useSecrets.ts`, `RepositoryFileFieldRenderer.tsx`, etc.). The test file `index.spec.tsx` transitively reaches these modules through `InstallProgressPanel → InstallIntegrationsSection → useIntegrations`.

**Fix:** Added a custom Vite plugin (`generatedApiClientStub`) to `vitest.config.ts` with `enforce: "pre"`. The plugin intercepts any `resolveId` call whose path matches `api-client/(sdk|types).gen` and returns a virtual module ID. When Vite then calls `load()` for that virtual ID, it returns an empty ESM module (`export {};`). This satisfies Vite's import-analysis without requiring the generated files to exist on disk.

The plugin only runs during Vitest (not during the production Vite build) since the generated files exist in real dev/build environments after `make pb.gen`.