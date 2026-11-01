export function fleetCapacityResponse(failed: boolean) {
  if (failed) return Response.json({ code: 14 }, { status: 503 });
  return Response.json({ pendingRunners: "4", idleRunners: "1", busyRunners: "2", terminatedRunners: "5" });
}
