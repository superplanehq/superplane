// ─────────────────────────────────────────────────────────────────────────
// THE AGENT FLEET — one agent per call.  POST /api/agents/{agent}
//
// The coarse /api/stages/* endpoints run a whole phase per call. These expose
// ONE agent each, so a SuperPlane Canvas can drive the *exact process* at full
// granularity: fan the six perception sensors out in parallel, fuse them, branch
// the two design tracks, and gate between every phase. Each handler:
//   1. loads the Job from the store (shared with the worker + coarse endpoints),
//   2. runs exactly ONE agent module from src/agents/** (no domain logic here),
//   3. persists its slice — perception sensors stash into an in-process scratch
//      keyed by jobId; `perception-assemble` composes the PerceptionResult from
//      it exactly as the worker does and writes job.perception,
//   4. returns small flat JSON the Canvas routes on (see src/lib/contract.ts).
//
// Runs at zero credentials: every agent has a deterministic fallback, so the
// whole fleet executes end-to-end offline. See infra/superplane/ for the Canvas
// that wires these into nodes, gates, and the resume-production terminal.
// ─────────────────────────────────────────────────────────────────────────

import { NextResponse } from "next/server";
import type {
  Dimensions,
  IdentifiedPart,
  ImageAdmissibility,
  Job,
  MaterialClass,
  PerceptionResult,
  ReconstructedGeometry,
  ScaleSource,
} from "@/lib/types";
import type {
  AgentName,
  ClassificationAgentResponse,
  ConditioningAgentResponse,
  DimensioningAgentResponse,
  FinalizeAgentResponse,
  GenerativeCadAgentResponse,
  MaterialInferAgentResponse,
  PerceptionAssembleResponse,
  ReconstructionAgentResponse,
  SourcingAgentResponse,
  TelemetryAgentResponse,
} from "@/lib/contract";
import { appendAudit, upsertJob } from "@/lib/store";
import { createAuditEntry } from "@/lib/audit";
import { runConditioning } from "@/agents/perception/conditioning";
import { runClassification } from "@/agents/perception/classification";
import { runReconstruction } from "@/agents/perception/reconstruction";
import { runDimensioning } from "@/agents/perception/dimensioning";
import { runMaterialInference } from "@/agents/perception/material";
import { runTelemetry } from "@/agents/perception/telemetry";
import { runSourcing } from "@/agents/design/sourcing";
import { runGenerativeCad } from "@/agents/design/generative-cad";
import { compositeConfidence, errResponse, isLoadBearing, resolveJob } from "../../stages/_lib";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

// ───────────────────────── perception scratch (in-process) ─────────────────────────
// The six perception sensors run as independent nodes, so their outputs are
// stashed here keyed by jobId until `perception-assemble` composes them into the
// canonical PerceptionResult. Pinned to globalThis so every Next.js route bundle
// shares one map (the store/worker/superplane pattern). Cleared on assemble.
interface PerceptionScratch {
  admissibility?: ImageAdmissibility[];
  identifiedPart?: IdentifiedPart;
  reconstructedGeometry?: ReconstructedGeometry;
  dimensions?: Dimensions;
  scaleSource?: ScaleSource | null;
  dimensionalConfidence?: number;
  inferredMaterialClass?: MaterialClass;
  surfaceFinish?: string;
  failureMode?: string | null;
  sensorFusionAgreement?: number | null;
}
const g = globalThis as unknown as {
  __kineticosScratch?: Map<string, PerceptionScratch>;
};
const scratchStore: Map<string, PerceptionScratch> =
  g.__kineticosScratch ?? (g.__kineticosScratch = new Map());

function scratch(jobId: string): PerceptionScratch {
  let s = scratchStore.get(jobId);
  if (!s) {
    s = {};
    scratchStore.set(jobId, s);
  }
  return s;
}

// Ordered for the GET discovery view + the Canvas layout.
const AGENT_ORDER: AgentName[] = [
  "conditioning",
  "classification",
  "reconstruction",
  "dimensioning",
  "material-infer",
  "telemetry",
  "perception-assemble",
  "sourcing",
  "generative-cad",
  "finalize",
];

// ───────────────────────── the dispatch ─────────────────────────
type Handler = (job: Job) => Promise<NextResponse>;

const HANDLERS: Record<AgentName, Handler> = {
  // 2A — image admissibility
  async conditioning(job) {
    const admissibility = (await runConditioning(job)).data;
    scratch(job.jobId).admissibility = admissibility;
    const body: ConditioningAgentResponse = {
      job_id: job.jobId,
      admissible: admissibility.filter((a) => a.admissible).length,
      images: admissibility.length,
    };
    return NextResponse.json(body);
  },

  // 2B — broken-component localization (drives every later stage)
  async classification(job) {
    const identified = (await runClassification(job)).data;
    scratch(job.jobId).identifiedPart = identified;
    const body: ClassificationAgentResponse = {
      job_id: job.jobId,
      part_class: identified.partClass,
      conf_class: identified.confClass,
      ambiguous_class: identified.ambiguousClass,
      additional_parts: identified.additionalParts?.length ?? 0,
      load_bearing: isLoadBearing(identified.partClass),
    };
    return NextResponse.json(body);
  },

  // 2C — reconstruct the undamaged intent (needs 2B's identified part)
  async reconstruction(job) {
    const identified = scratch(job.jobId).identifiedPart;
    if (!identified) {
      return errResponse(409, "classification (2B) must run before reconstruction (2C)");
    }
    const reconstructed = (await runReconstruction(job, identified)).data;
    scratch(job.jobId).reconstructedGeometry = reconstructed;
    const body: ReconstructionAgentResponse = {
      job_id: job.jobId,
      method: reconstructed.method,
      reconstruction_confidence: reconstructed.reconstructionConfidence,
      failure_class: reconstructed.failureClass,
      has_geometry: reconstructed.uri !== null,
    };
    return NextResponse.json(body);
  },

  // 2D — mating interfaces + terminal scale chain
  async dimensioning(job) {
    const dim = (await runDimensioning(job)).data;
    const s = scratch(job.jobId);
    s.dimensions = dim.dimensions;
    s.scaleSource = dim.scaleSource;
    s.dimensionalConfidence = dim.dimensionalConfidence;
    const body: DimensioningAgentResponse = {
      job_id: job.jobId,
      scale_source: dim.scaleSource,
      dimensional_confidence: dim.dimensionalConfidence,
      parameters: Object.keys(dim.dimensions).length,
    };
    return NextResponse.json(body);
  },

  // 2E — material + surface inference
  async "material-infer"(job) {
    const material = (await runMaterialInference(job)).data;
    const s = scratch(job.jobId);
    s.inferredMaterialClass = material.inferredMaterialClass;
    s.surfaceFinish = material.surfaceFinish;
    const body: MaterialInferAgentResponse = {
      job_id: job.jobId,
      inferred_material_class: material.inferredMaterialClass,
      surface_finish: material.surfaceFinish,
    };
    return NextResponse.json(body);
  },

  // 2F — telemetry track + sensor fusion
  async telemetry(job) {
    const tele = (await runTelemetry(job)).data;
    const s = scratch(job.jobId);
    s.failureMode = tele.failureMode;
    s.sensorFusionAgreement = tele.sensorFusionAgreement;
    const body: TelemetryAgentResponse = {
      job_id: job.jobId,
      failure_mode: tele.failureMode,
      sensor_fusion_agreement: tele.sensorFusionAgreement,
    };
    return NextResponse.json(body);
  },

  // 2.x — fuse the six sensors into the canonical PerceptionResult (worker parity)
  async "perception-assemble"(job) {
    const s = scratch(job.jobId);
    if (!s.identifiedPart || !s.reconstructedGeometry || !s.dimensions || !s.admissibility) {
      return errResponse(
        409,
        "run conditioning, classification, reconstruction and dimensioning before perception-assemble",
      );
    }
    const perception: PerceptionResult = {
      admissibility: s.admissibility,
      identifiedPart: s.identifiedPart,
      reconstructedGeometry: s.reconstructedGeometry,
      dimensions: s.dimensions,
      scaleSource: s.scaleSource ?? null,
      dimensionalConfidence: s.dimensionalConfidence ?? 0,
      inferredMaterialClass: s.inferredMaterialClass ?? "unknown",
      surfaceFinish: s.surfaceFinish ?? "unknown",
      failureMode: s.failureMode ?? null,
      sensorFusionAgreement: s.sensorFusionAgreement ?? null,
      confTotal: compositeConfidence(
        s.identifiedPart.confClass,
        s.dimensionalConfidence ?? 0,
        s.reconstructedGeometry.reconstructionConfidence,
        s.sensorFusionAgreement ?? null,
      ),
    };
    job.perception = perception;
    await upsertJob(job);
    scratchStore.delete(job.jobId);
    await safeAudit(job.jobId, "perception-fleet", "assemble", `composite confidence ${perception.confTotal.toFixed(2)}`);

    const body: PerceptionAssembleResponse = {
      job_id: job.jobId,
      conf_total: perception.confTotal,
      conf_class: perception.identifiedPart.confClass,
      dimensional_confidence: perception.dimensionalConfidence,
      reconstruction_confidence: perception.reconstructedGeometry.reconstructionConfidence,
      sensor_fusion_agreement: perception.sensorFusionAgreement,
      ambiguous_class: perception.identifiedPart.ambiguousClass,
      scale_source: perception.scaleSource,
      load_bearing: isLoadBearing(perception.identifiedPart.partClass),
    };
    return NextResponse.json(body);
  },

  // 3A — off-the-shelf substitute federation (null → fall through to 3B)
  async sourcing(job) {
    if (!job.perception) return errResponse(409, "perception must run before sourcing (3A)");
    const blueprint = (await runSourcing(job)).data;
    if (blueprint) {
      job.blueprint = blueprint;
      await upsertJob(job);
    }
    const body: SourcingAgentResponse = {
      job_id: job.jobId,
      matched: blueprint !== null,
      source_type: blueprint?.sourceType ?? null,
      match_score: blueprint?.matchScore ?? null,
      strategy: blueprint?.strategy ?? null,
    };
    return NextResponse.json(body);
  },

  // 3B — generative continuation CAD, B1..B8 (the new 3D-printable file)
  async "generative-cad"(job) {
    if (!job.perception) return errResponse(409, "perception must run before generative-cad (3B)");
    const blueprint = (await runGenerativeCad(job)).data;
    job.blueprint = blueprint;
    await upsertJob(job);
    const body: GenerativeCadAgentResponse = {
      job_id: job.jobId,
      source_type: "generated",
      strategy: blueprint.strategy,
      load_bearing: isLoadBearing(job.perception.identifiedPart.partClass),
      design_trail: blueprint.designTrail,
    };
    return NextResponse.json(body);
  },

  // 8.3 — seal the run; the line is resumed on the accepted v1 continuation
  async finalize(job) {
    job.status = "complete";
    job.pendingGate = null;
    job.auditTrail.push(
      createAuditEntry(
        "fleet",
        "complete",
        "continuation accepted — audit sealed, production resumed on the v1 CAD output",
      ),
    );
    await upsertJob(job);
    const body: FinalizeAgentResponse = {
      job_id: job.jobId,
      status: job.status,
      resumed: true,
    };
    return NextResponse.json(body);
  },
};

async function safeAudit(jobId: string, actor: string, action: string, detail: string): Promise<void> {
  try {
    await appendAudit(jobId, createAuditEntry(actor, action, detail));
  } catch {
    // never fail the fleet on an audit write
  }
}

// ───────────────────────── route handlers ─────────────────────────
export async function POST(
  req: Request,
  { params }: { params: Promise<{ agent: string }> },
) {
  const { agent } = await params;
  const handler = HANDLERS[agent as AgentName];
  if (!handler) {
    return errResponse(404, `unknown agent "${agent}" — one of: ${AGENT_ORDER.join(", ")}`);
  }

  let body: { job_id?: unknown };
  try {
    body = (await req.json()) as { job_id?: unknown };
  } catch {
    return errResponse(400, "invalid JSON body");
  }
  const resolved = await resolveJob(body.job_id);
  if ("error" in resolved) return resolved.error;

  return handler(resolved.job);
}
