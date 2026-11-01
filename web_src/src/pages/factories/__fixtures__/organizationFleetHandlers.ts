import type { OrganizationsOrganizationRunnerFleetSpec } from "@/api-client";
import type { FactoriesFixture } from "./factoryPageResponses";
import type { FactoriesRoute } from "./handlers";

type Management = NonNullable<FactoriesFixture["organizationFleetManagement"]>;

function fleetResponse(
  management: Management,
  match: RegExpExecArray,
  method: string,
  body: Record<string, unknown> | null,
) {
  const fleets = management.fleets;
  if (match[4] === "capacity") {
    const runners = management.runners.filter((runner) => runner.fleetId === match[3]);
    return {
      json: Object.fromEntries(
        ["pending", "idle", "busy", "terminated"].map((state) => [
          `${state}Runners`,
          String(runners.filter((runner) => runner.state === state).length),
        ]),
      ),
    };
  }
  if (!match[3]) {
    if (method === "POST")
      fleets.push({
        id: String(body?.fleetId),
        enabled: body?.enabled === true,
        runnerVersion: String(body?.runnerVersion),
        spec: body?.spec as OrganizationsOrganizationRunnerFleetSpec,
      });
    return { json: { fleets, fleet: fleets.at(-1) } };
  }
  const fleet = fleets.find((record) => record.id === match[3]);
  if (method === "PATCH" && fleet) Object.assign(fleet, body);
  return fleet ? { json: { fleet } } : { status: 404, json: { code: 5 } };
}

function runnerResponse(
  management: Management,
  match: RegExpExecArray,
  method: string,
  body: Record<string, unknown> | null,
  url: URL,
) {
  const runners = management.runners;
  if (method === "POST") {
    const runner = {
      id: crypto.randomUUID(),
      fleetId: String(body?.fleetId),
      state: "pending",
      ephemeral: body?.ephemeral === true,
    };
    runners.push(runner);
    return {
      json: {
        runner,
        registrationToken: "synthetic-story-registration",
        runnerApiUrl: "https://superplane.example",
        registrationExpiresAt: new Date(
          Date.now() + (management.connection === "expired" ? -1000 : 60000),
        ).toISOString(),
      },
    };
  }
  if (!match[3]) {
    const states = url.searchParams.getAll("states");
    const filtered = runners.filter(
      (runner) =>
        runner.fleetId === url.searchParams.get("fleetId") && (!states.length || states.includes(runner.state ?? "")),
    );
    return { json: { runners: filtered, totalCount: String(filtered.length), hasNextPage: false } };
  }
  const runner = runners.find((record) => record.id === match[3]);
  if (!runner) return { status: 404, json: { code: 5 } };
  if (management.connection === "connected")
    Object.assign(runner, {
      registeredAt: new Date().toISOString(),
      state: "idle",
      hostname: "self-hosted-01",
      os: "linux",
      arch: "amd64",
      ip: "192.0.2.10",
    });
  if (method === "DELETE" && !management.pendingTermination) runner.state = "terminated";
  return { json: { runner } };
}

export function organizationFleetRoutes(fixture: FactoriesFixture): FactoriesRoute[] {
  const management = fixture.organizationFleetManagement ?? { fleets: [], runners: [] };
  return [
    {
      pattern: /^\/api\/v1\/organizations\/([^/]+)\/(fleets|runners)(?:\/([^/]+))?(?:\/(capacity))?$/,
      resolve: (match, method, body, url) => {
        if (management.error) return { status: 503, json: { code: 14 } };
        if (match[2] === "fleets") return fleetResponse(management, match, method, body);
        return runnerResponse(management, match, method, body, url);
      },
    },
  ];
}
