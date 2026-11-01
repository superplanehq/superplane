import { useEffect, useRef, useState } from "react";
import {
  organizationsCreateFleetRunner,
  organizationsDescribeFleetRunner,
  type OrganizationsCreateFleetRunnerResponse,
  type SuperplaneOrganizationsRunner,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

type RegistrationStatus = "ready" | "creating" | "waiting" | "connected" | "expired" | "terminated";
type RegistrationContext = { organizationId: string; fleetId: string; refresh: () => void };

function registrationOutcome(runner: SuperplaneOrganizationsRunner | undefined, expiresAt: string): RegistrationStatus {
  if (runner?.registeredAt) return "connected";
  if (runner?.state === "terminated") return "terminated";
  if (runner?.state === "pending" && Date.now() >= Date.parse(expiresAt)) return "expired";
  return "waiting";
}

function useRegistrationPoll(
  context: RegistrationContext,
  registration: OrganizationsCreateFleetRunnerResponse | null,
  finish: (status: RegistrationStatus) => void,
  setError: (error: string) => void,
  retry: number,
) {
  const { organizationId, fleetId } = context;
  const callbacks = useRef({ finish, setError });
  callbacks.current = { finish, setError };
  useEffect(() => {
    if (!registration) return;
    let stopped = false;
    let timer: ReturnType<typeof setTimeout>;
    let abort: AbortController | undefined;
    const poll = async () => {
      if (stopped) return;
      if (document.visibilityState === "hidden") {
        timer = setTimeout(poll, 5000);
        return;
      }
      abort = new AbortController();
      try {
        const response = await organizationsDescribeFleetRunner(
          withOrganizationHeader({
            organizationId,
            path: { id: organizationId, runnerId: registration.runner!.id! },
            query: { fleetId },
            signal: abort.signal,
          }),
        );
        if (stopped) return;
        const outcome = registrationOutcome(response.data.runner, registration.registrationExpiresAt!);
        callbacks.current.setError("");
        if (outcome !== "waiting") {
          callbacks.current.finish(outcome);
          return;
        }
      } catch {
        if (stopped) return;
        callbacks.current.setError("Could not check registration. Retry the status check.");
      }
      if (!stopped) timer = setTimeout(poll, 5000);
    };
    void poll();
    return () => {
      stopped = true;
      clearTimeout(timer);
      abort?.abort();
    };
  }, [organizationId, fleetId, registration, retry]);
}

// Secret-bearing output belongs only to the mounted connection view, never a query or mutation cache.
export function useRunnerRegistration(context: RegistrationContext) {
  const [mode, setMode] = useState("ephemeral");
  const [registration, setRegistration] = useState<OrganizationsCreateFleetRunnerResponse | null>(null);
  const [status, setStatus] = useState<RegistrationStatus>("ready");
  const [error, setError] = useState("");
  const [retry, setRetry] = useState(0);
  const attempt = useRef({ key: crypto.randomUUID(), ephemeral: true });
  const request = useRef<AbortController | null>(null);
  const refreshRef = useRef(context.refresh);
  refreshRef.current = context.refresh;
  useEffect(() => () => request.current?.abort(), []);
  useEffect(() => {
    if (!registration?.registrationExpiresAt) return;
    const remaining = Date.parse(registration.registrationExpiresAt) - Date.now();
    if (remaining <= 0) return;
    const timer = setTimeout(() => setRetry((value) => value + 1), remaining);
    return () => clearTimeout(timer);
  }, [registration]);
  useRegistrationPoll(
    context,
    registration,
    (outcome) => {
      setRegistration(null);
      setStatus(outcome);
      refreshRef.current();
    },
    setError,
    retry,
  );

  const create = async (replacement = false) => {
    if (status === "creating") return;
    if (replacement) attempt.current = { key: crypto.randomUUID(), ephemeral: mode === "ephemeral" };
    request.current?.abort();
    const abort = new AbortController();
    request.current = abort;
    setRegistration(null);
    setError("");
    setStatus("creating");
    try {
      const response = await organizationsCreateFleetRunner(
        withOrganizationHeader({
          organizationId: context.organizationId,
          path: { id: context.organizationId },
          body: { fleetId: context.fleetId, ephemeral: attempt.current.ephemeral, idempotencyKey: attempt.current.key },
          signal: abort.signal,
        }),
      );
      if (abort.signal.aborted) return;
      const output = response.data;
      if (!output.runner?.id || !output.registrationToken || !output.runnerApiUrl || !output.registrationExpiresAt)
        throw new Error("Incomplete registration");
      setRegistration(output);
      setStatus("waiting");
      refreshRef.current();
    } catch {
      if (abort.signal.aborted) return;
      setStatus("ready");
      setError("Could not create registration. Check fleet access and retry.");
      refreshRef.current();
    }
  };
  return {
    mode,
    registration,
    status,
    error,
    setError,
    create,
    retryStatus: () => setRetry((value) => value + 1),
    chooseMode: (value: string) => {
      setMode(value);
      attempt.current = { key: crypto.randomUUID(), ephemeral: value === "ephemeral" };
    },
  };
}
