export function stagingCommitSuccessToast(factoryContext: boolean): string {
  if (factoryContext) {
    return "Changes saved";
  }
  return "Changes committed";
}

export function stagingResetSuccessToast(factoryContext: boolean): string {
  if (factoryContext) {
    return "Changes discarded";
  }
  return "Reverted to last commit";
}
