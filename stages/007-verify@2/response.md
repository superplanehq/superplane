All checks passed:

- `make format.js` — all files unchanged (exit 0)
- `canvasAppPreferencePresentation.spec.ts` — 1 test passed
- `index.spec.tsx` — 23 tests passed (previously failing due to missing `@/api-client/sdk.gen`; the fix in `vitest.config.ts` resolved it)

```json
{"outcome": "succeeded", "context_updates": {"verify.summary": "format.js passed; canvasAppPreferencePresentation.spec.ts (1 test) and index.spec.tsx (23 tests) all passed with the generatedApiClientStub plugin in vitest.config.ts resolving the previously-failing sdk.gen import"}}
```