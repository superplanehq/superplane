// GET /api/agents — the fleet roster: every agent the Canvas can call, in
// execution order, with the phase it implements. Discovery endpoint for the
// SuperPlane Canvas builder + the docs in infra/superplane/.

import { NextResponse } from "next/server";
import type { AgentName } from "@/lib/contract";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const FLEET: { agent: AgentName; phase: string; label: string }[] = [
  { agent: "conditioning", phase: "2A", label: "CAD image admissibility" },
  { agent: "classification", phase: "2B", label: "broken-component localization" },
  { agent: "reconstruction", phase: "2C", label: "reconstruct the undamaged intent" },
  { agent: "dimensioning", phase: "2D", label: "mating interfaces + scale chain" },
  { agent: "material-infer", phase: "2E", label: "material + surface inference" },
  { agent: "telemetry", phase: "2F", label: "telemetry track + sensor fusion" },
  { agent: "perception-assemble", phase: "2.x", label: "fuse sensors → PerceptionResult" },
  { agent: "sourcing", phase: "3A", label: "off-the-shelf substitute federation" },
  { agent: "generative-cad", phase: "3B", label: "generative continuation CAD (B1–B8)" },
  { agent: "finalize", phase: "8.3", label: "seal the run, resume production" },
];

export async function GET() {
  return NextResponse.json({ fleet: FLEET, count: FLEET.length });
}
