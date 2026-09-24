/** Keep setup mounted after this visit marks onboarding complete. */
export function holdSetupAfterThisVisitCompletes(startedIncomplete: boolean, isSetupRoute: boolean) {
  return startedIncomplete && isSetupRoute;
}
