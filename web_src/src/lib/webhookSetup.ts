export function webhookSetupInProgress(state: string | undefined): boolean {
  return state === "pending" || state === "provisioning";
}
