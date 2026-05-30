// ─────────────────────────────────────────────────────────────────────────
// THE SHARED CONTRACT (types only) — the A↔B seam.
//
// The pivot divides at one seam: the SuperPlane App (Workstream A) calls this
// Next app (Workstream B) over HTTP, and this app fires the Canvas webhook.
// This file is the single source of truth for the wire shapes both sides agree
// on. Person A imports the request/response types to build Canvas nodes; the
// stage endpoints under src/app/api/stages/** + src/app/api/evaluate-gate
// implement them verbatim.
//
// Wire fields are snake_case (so Canvas `if` expressions stay simple and flat);
// the internal Job slices in ./types are camelCase. Each endpoint maps one to
// the other. This module re-uses the canonical enums from ./types so the wire
// contract can never silently drift from the domain model.
// ─────────────────────────────────────────────────────────────────────────

import type {
  BlueprintSourceType,
  DesignDecision,
  GateDecisionType,
  GateName,
  MachineTarget,
  ScaleSource,
} from "./types";

// ───────────────────────── stage endpoints (B implements, A calls) ─────────────────────────
// Every stage endpoint takes exactly this, loads the Job from the store, runs
// the existing agent module(s), persists the Job slice, and returns flat JSON.
export interface StageRequest {
  job_id: string;
}

/** POST /api/stages/perception → the composite-confidence routing fields. */
export interface PerceptionStageResponse {
  job_id: string;
  conf_total: number;
  conf_class: number;
  dimensional_confidence: number;
  reconstruction_confidence: number;
  sensor_fusion_agreement: number | null; // null when no telemetry present
  ambiguous_class: boolean;
  scale_source: ScaleSource | null;
  load_bearing: boolean;
}

/** POST /api/stages/design → the continuation-strategy routing fields. */
export interface DesignStageResponse {
  job_id: string;
  source_type: BlueprintSourceType; // "sourced" | "generated"
  match_score: number | null; // Track A only; null for a generated continuation
  cad_uri_step: string | null;
  cad_uri_stl: string | null;
  load_bearing: boolean;
  // Additive (beyond the flat wire contract): the B1..B8 design trail. The
  // Canvas routes only on the flat fields and ignores this; the UI/audit render
  // it. null on a sourced (Track A) substitute, which has no synthesis trail.
  design_trail: DesignDecision[] | null;
}

/** POST /api/stages/material → printability + structural-feasibility fields. */
export interface MaterialStageResponse {
  job_id: string;
  machine_target: MachineTarget | null; // "fdm" | "sla" | "cnc"
  toolpath_uri: string | null;
  structural_ok: boolean;
  margin_pct: number;
}

/** POST /api/stages/fabricate-report → output-validation (QA) fields. */
export interface FabricateReportResponse {
  job_id: string;
  dimensional_pass: boolean | null;
  structural_pass: boolean | null;
  in_process_anomalies: number;
}

// ───────────────────────── gate evaluation (optional convenience) ─────────────────────────
// A can route purely on the confidence numbers with Canvas `if` nodes, OR call
// this to reuse src/lib/gates/index.ts policy verbatim.
export interface EvaluateGateRequest {
  job_id: string;
  gate: GateName; // composite_confidence | continuation_strategy | printability | output_acceptance
}

export interface EvaluateGateResponse {
  decision: GateDecisionType; // "proceed" | "human_review" | "block"
  reason: string;
  scoped_field: string | null; // the precise field a human must resolve, if any
}

// ───────────────────────── granular agent fleet (B implements, A calls) ─────────────────────────
// The coarse stage endpoints above run a whole phase per call. The agent-fleet
// endpoints under /api/agents/[agent] expose ONE agent per call, so a SuperPlane
// Canvas can fan the perception sensors out in parallel, branch the two design
// tracks, and render the fleet at full granularity. Each takes { job_id }, runs
// exactly one agent, persists its slice (perception sensors persist to an
// in-process scratch the `perception-assemble` agent then composes), and returns
// a small flat result the Canvas routes on. See infra/superplane/.
export type AgentName =
  | "conditioning" //         2A  image admissibility
  | "classification" //       2B  broken-component localization
  | "reconstruction" //       2C  undamaged-intent reconstruction (needs 2B)
  | "dimensioning" //         2D  mating interfaces + scale chain
  | "material-infer" //       2E  material + surface inference
  | "telemetry" //            2F  telemetry track + sensor fusion
  | "perception-assemble" //  2.x compose PerceptionResult from the sensors
  | "sourcing" //             3A  off-the-shelf substitute federation
  | "generative-cad" //       3B  generative continuation CAD (B1–B8)
  | "finalize"; //            8.3 seal the run, mark the line resumed

export interface ConditioningAgentResponse {
  job_id: string;
  admissible: number; // count of admissible images
  images: number; // total images assessed
}
export interface ClassificationAgentResponse {
  job_id: string;
  part_class: string;
  conf_class: number;
  ambiguous_class: boolean;
  additional_parts: number; // other broken components in the same image
  load_bearing: boolean;
}
export interface ReconstructionAgentResponse {
  job_id: string;
  method: string;
  reconstruction_confidence: number;
  failure_class: string | null;
  has_geometry: boolean; // false → confidence below gate → human overlay
}
export interface DimensioningAgentResponse {
  job_id: string;
  scale_source: ScaleSource | null;
  dimensional_confidence: number;
  parameters: number; // count of resolved dimensional parameters
}
export interface MaterialInferAgentResponse {
  job_id: string;
  inferred_material_class: string;
  surface_finish: string;
}
export interface TelemetryAgentResponse {
  job_id: string;
  failure_mode: string | null;
  sensor_fusion_agreement: number | null;
}
/** perception-assemble returns the SAME routing shape as the coarse endpoint. */
export type PerceptionAssembleResponse = PerceptionStageResponse;
export interface SourcingAgentResponse {
  job_id: string;
  matched: boolean; // false → fall through to the generative track (3B)
  source_type: BlueprintSourceType | null;
  match_score: number | null;
  strategy: string | null;
}
export interface GenerativeCadAgentResponse {
  job_id: string;
  source_type: "generated";
  strategy: string;
  load_bearing: boolean;
  design_trail: DesignDecision[] | null; // B1..B8
}
export interface FinalizeAgentResponse {
  job_id: string;
  status: string; // "complete"
  resumed: boolean; // production resumed on the v1 continuation
}

// ───────────────────────── intake → Canvas webhook (A defines, B fires) ─────────────────────────
// POST /api/jobs creates the Job, then POSTs this to SP_INGEST_WEBHOOK with the
// header `X-Webhook-Token: <SP_WEBHOOK_TOKEN>`. (Firing is Workstream B2.)
export interface IngestWebhookPayload {
  job_id: string;
  image_uris: string[];
  telemetry_uri: string | null;
}
