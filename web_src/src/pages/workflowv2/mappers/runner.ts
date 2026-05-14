/**
 * Stable entry path for `import "./runner"` (Vite resolves `.ts` before `.tsx`).
 * Implementation with JSX lives in {@link ./runnerImpl}.
 */
export { runnerMapper, RUNNER_STATE_REGISTRY } from "./runnerImpl";
