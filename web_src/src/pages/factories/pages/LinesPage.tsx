import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
  ArrowLeft,
  Bug,
  Layers,
  Lock,
  MoreHorizontal,
  Plus,
  Rocket,
  Sparkles,
  Workflow,
  type LucideIcon,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { cn } from "@/lib/utils";
import { LineVelocityPanel } from "./LineVelocityPanel";
import { WorkOrderCanvas } from "./WorkOrderCanvas";

const lineDetailTabTriggerClass =
  "rounded-none border-0 border-b-2 border-transparent bg-transparent px-0 pb-2.5 pt-0 text-[13px] text-muted-foreground shadow-none data-[state=active]:border-foreground data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none dark:data-[state=active]:border-foreground dark:data-[state=active]:bg-transparent";

type WorkOrderStatus = "running" | "queued" | "waiting";

type PhaseWorkOrder = {
  id: string;
  title: string;
  status: WorkOrderStatus;
  meta: string;
};

type Phase = {
  id: string;
  name: string;
  summary: string;
  paused?: boolean;
  workOrders: PhaseWorkOrder[];
};

type Line = {
  id: string;
  name: string;
  description: string;
  icon: LucideIcon;
  paused: boolean;
  phases: Phase[];
};

type ConfirmAction = {
  kind: "pause" | "resume";
  target: "line" | "phase";
  lineId: string;
  phaseId?: string;
  name: string;
};

const INITIAL_LINES: Line[] = [
  {
    id: "poc-builder",
    name: "Proof-of-concept builder",
    description: "Spikes and thin vertical slices from idea to a demoable PR.",
    icon: Sparkles,
    paused: false,
    phases: [
      {
        id: "plan",
        name: "Plan",
        summary: "Turn the idea into a bounded plan for the POC.",
        workOrders: [
          { id: "poc-1", title: "Spike agent memory API", status: "running", meta: "4m" },
          { id: "poc-2", title: "Thin slice for Console widget", status: "queued", meta: "next" },
        ],
      },
      {
        id: "build",
        name: "Build",
        summary: "Implement the thinnest path that proves the concept.",
        workOrders: [
          { id: "poc-3", title: "Build console-run widget branch", status: "running", meta: "18m" },
          { id: "poc-4", title: "Bind widget to latest run status", status: "waiting", meta: "needs input" },
          { id: "poc-5", title: "Demo script draft", status: "queued", meta: "next" },
        ],
      },
      {
        id: "demo",
        name: "Demo",
        summary: "Package a reviewable PR and a short demo note.",
        workOrders: [],
      },
    ],
  },
  {
    id: "security-updater",
    name: "Security updater",
    description: "Dependency and advisory fixes that stay in a safe lane.",
    icon: Lock,
    paused: false,
    phases: [
      {
        id: "detect",
        name: "Detect",
        summary: "Watch advisories and open work orders for actionable CVEs.",
        workOrders: [{ id: "sec-1", title: "CVE-2026-114 scan", status: "running", meta: "1m" }],
      },
      {
        id: "patch",
        name: "Patch",
        summary: "Apply upgrades and minimal code changes.",
        workOrders: [
          { id: "sec-2", title: "Bump lodash advisory", status: "queued", meta: "next" },
          { id: "sec-3", title: "OpenSSL transitive bump", status: "waiting", meta: "needs input" },
        ],
      },
      {
        id: "verify",
        name: "Verify",
        summary: "Run security checks and open a PR when clean.",
        workOrders: [{ id: "sec-4", title: "Verify axios upgrade", status: "running", meta: "6m" }],
      },
    ],
  },
  {
    id: "feature-implementor",
    name: "Feature implementor",
    description: "Ship product features through intake, build, and verification.",
    icon: Rocket,
    paused: false,
    phases: [
      {
        id: "intake",
        name: "Intake",
        summary: "Accept a mission or ticket and create a work order.",
        workOrders: [],
      },
      {
        id: "implement",
        name: "Implement",
        summary: "Build against the accepted acceptance criteria.",
        workOrders: [],
      },
      {
        id: "verify",
        name: "Verify",
        summary: "Test, review, and open a PR.",
        workOrders: [],
      },
    ],
  },
  {
    id: "bug-fixer",
    name: "Bug fixer",
    description: "Labeled issues become work orders, get fixed, then verified.",
    icon: Bug,
    paused: false,
    phases: [
      {
        id: "intake",
        name: "Intake",
        summary: "Listen for labeled issues and turn them into work orders.",
        workOrders: [
          { id: "bug-1", title: "Label bug: flaky login", status: "running", meta: "2m" },
          { id: "bug-2", title: "Label bug: timezone drift", status: "queued", meta: "next" },
        ],
      },
      {
        id: "fix",
        name: "Fix",
        summary: "Reproduce, patch, verify, and open a pull request when ready.",
        workOrders: [
          { id: "bug-3", title: "Redact secrets in run payloads", status: "running", meta: "14m" },
          { id: "bug-4", title: "Generate new automation", status: "waiting", meta: "needs input" },
          { id: "bug-5", title: "Null guard on memory browser", status: "queued", meta: "next" },
          { id: "bug-6", title: "Verify code quality", status: "running", meta: "7m" },
        ],
      },
    ],
  },
  {
    id: "docs-writer",
    name: "Docs writer",
    description: "Turn shipped changes into draft docs and reviewable PRs.",
    icon: Layers,
    paused: false,
    phases: [
      {
        id: "collect",
        name: "Collect",
        summary: "Gather merged changes that need documentation.",
        workOrders: [],
      },
      {
        id: "outline",
        name: "Outline",
        summary: "Structure the doc sections before drafting.",
        workOrders: [],
      },
      {
        id: "draft",
        name: "Draft",
        summary: "Write the first pass of docs from the change set.",
        workOrders: [],
      },
      {
        id: "review",
        name: "Review",
        summary: "Open a docs PR for human review.",
        workOrders: [],
      },
      {
        id: "publish",
        name: "Publish",
        summary: "Merge and publish the docs to SuperPlane docs.",
        workOrders: [],
      },
    ],
  },
];

const quietActionClass = "cursor-pointer text-[13px] text-muted-foreground transition-colors hover:text-foreground";

function phaseTick(phase: Phase): WorkOrderStatus | null {
  let running = false;
  let queued = false;
  let waiting = false;
  for (const item of phase.workOrders) {
    if (item.status === "running") running = true;
    else if (item.status === "waiting") waiting = true;
    else queued = true;
  }
  if (waiting) return "waiting";
  if (running) return "running";
  if (queued) return "queued";
  return null;
}

function StageStrip({ phases }: { phases: Phase[] }) {
  return (
    <ol className="mt-3.5 flex w-full items-start">
      {phases.map((phase, index) => {
        const tickKind = phaseTick(phase);
        return (
          <li key={phase.id} className="relative flex min-w-0 flex-1 flex-col items-center text-center">
            {index < phases.length - 1 ? (
              <span
                className="absolute top-[7px] left-[calc(50%+8px)] right-[calc(-50%+8px)] h-px bg-border"
                aria-hidden
              />
            ) : null}
            <span className="relative z-[1] flex h-3.5 items-center justify-center">
              {tickKind ? (
                <span
                  className={cn(
                    "size-2 shrink-0 rounded-full",
                    tickKind === "running" && "bg-[#3b82f6] animate-[status-blink_1.2s_ease-in-out_infinite]",
                    tickKind === "waiting" && "bg-[#f59e0b]",
                    tickKind === "queued" && "bg-[#a3a3a3]",
                  )}
                  aria-hidden
                />
              ) : (
                <span className="size-2 rounded-full bg-[#c4c4c4]" aria-hidden />
              )}
            </span>
            <span className="mt-1.5 max-w-full truncate px-1 text-[12px] leading-tight text-muted-foreground">
              {phase.name}
            </span>
          </li>
        );
      })}
    </ol>
  );
}

function EntityActionsMenu({
  label,
  onEdit,
  onPause,
  onResume,
  onDelete,
  onDuplicate,
  deleteDisabled,
  hoverReveal,
}: {
  label: string;
  onEdit: () => void;
  onPause: () => void;
  onResume: () => void;
  onDelete: () => void;
  onDuplicate: () => void;
  deleteDisabled?: boolean;
  hoverReveal?: boolean;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className={cn(
          "inline-flex size-6 cursor-pointer items-center justify-center rounded-md text-muted-foreground outline-none transition-colors hover:bg-accent hover:text-foreground data-[state=open]:bg-accent data-[state=open]:text-foreground",
          hoverReveal &&
            "opacity-0 group-hover/line:opacity-100 focus-visible:opacity-100 data-[state=open]:opacity-100",
        )}
        aria-label={`Actions for ${label}`}
        onClick={(event) => event.stopPropagation()}
      >
        <MoreHorizontal className="size-3.5" strokeWidth={1.75} />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-40" onClick={(event) => event.stopPropagation()}>
        <DropdownMenuItem className="cursor-pointer text-[13px]" onClick={onEdit}>
          Edit
        </DropdownMenuItem>
        <DropdownMenuItem className="cursor-pointer text-[13px]" onClick={onPause}>
          Pause
        </DropdownMenuItem>
        <DropdownMenuItem className="cursor-pointer text-[13px]" onClick={onResume}>
          Resume
        </DropdownMenuItem>
        <DropdownMenuItem
          className="cursor-pointer text-[13px] text-[#b91c1c] focus:text-[#b91c1c]"
          disabled={deleteDisabled}
          onClick={onDelete}
        >
          Delete
        </DropdownMenuItem>
        <DropdownMenuItem className="cursor-pointer text-[13px]" onClick={onDuplicate}>
          Duplicate
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function PauseResumeDialog({
  confirm,
  onOpenChange,
  onConfirm,
}: {
  confirm: ConfirmAction | null;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  const isPause = confirm?.kind === "pause";
  const noun = confirm?.target === "phase" ? "stage" : "line";

  return (
    <Dialog open={confirm != null} onOpenChange={onOpenChange}>
      <DialogContent className="gap-4 p-5 sm:max-w-sm" showCloseButton={false}>
        <DialogHeader className="gap-1">
          <DialogTitle className="text-[15px] font-semibold tracking-[-0.01em]">
            {isPause ? `Pause ${noun}?` : `Resume ${noun}?`}
          </DialogTitle>
        </DialogHeader>
        <p className="text-[13px] leading-relaxed text-muted-foreground">
          {isPause
            ? `${confirm?.name} will stop accepting new work until you resume it. Work already in progress is not cancelled.`
            : `${confirm?.name} will start accepting work again.`}
        </p>
        <DialogFooter className="gap-2 sm:justify-end">
          <Button type="button" variant="ghost" size="sm" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" size="sm" onClick={onConfirm}>
            {isPause ? "Pause" : "Resume"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function LineCard({
  line,
  emphasized,
  editing,
  onClick,
  onDoneEditing,
  onUpdateLine,
  onUpdatePhase,
  onAddPhase,
  onDeletePhase,
  onEdit,
  onDuplicate,
  onDelete,
  onRequestPause,
  onRequestResume,
}: {
  line: Line;
  emphasized?: boolean;
  editing?: boolean;
  onClick?: () => void;
  onDoneEditing?: () => void;
  onUpdateLine?: (patch: Partial<Pick<Line, "name" | "description">>) => void;
  onUpdatePhase?: (phaseId: string, patch: Partial<Pick<Phase, "name" | "summary">>) => void;
  onAddPhase?: () => void;
  onDeletePhase?: (phaseId: string) => void;
  onEdit?: () => void;
  onDuplicate?: () => void;
  onDelete?: () => void;
  onRequestPause?: () => void;
  onRequestResume?: () => void;
}) {
  const Icon = line.icon;
  const showMenu = onEdit && onDuplicate && onDelete && onRequestPause && onRequestResume;

  const className = cn(
    "group/line w-full rounded-lg border px-3.5 py-3 text-left transition-colors",
    emphasized ? "border-foreground bg-background" : "border-border bg-background",
    onClick && "cursor-pointer hover:border-foreground/25 hover:bg-accent/40",
    line.paused && !emphasized && "opacity-80",
  );

  return (
    <div
      className={className}
      role={onClick ? "button" : undefined}
      tabIndex={onClick ? 0 : undefined}
      onClick={onClick}
      onKeyDown={
        onClick
          ? (event) => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                onClick();
              }
            }
          : undefined
      }
    >
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <Icon className="size-3.5 shrink-0 text-muted-foreground" strokeWidth={1.75} />
            {editing && onUpdateLine ? (
              <input
                value={line.name}
                onChange={(event) => onUpdateLine({ name: event.target.value })}
                onClick={(event) => event.stopPropagation()}
                className="-mx-1 min-w-0 flex-1 bg-transparent px-1 text-[13px] font-medium tracking-[-0.01em] text-foreground outline-none focus:bg-accent/60"
                aria-label="Line name"
              />
            ) : (
              <span className="text-[13px] font-medium tracking-[-0.01em] text-foreground">{line.name}</span>
            )}
            {line.paused ? (
              <span className="rounded-md border border-border bg-accent px-1.5 py-0.5 text-[11px] font-medium tracking-[-0.01em] text-muted-foreground">
                Paused
              </span>
            ) : null}
          </div>
          {editing && onUpdateLine ? (
            <textarea
              value={line.description}
              onChange={(event) => onUpdateLine({ description: event.target.value })}
              onClick={(event) => event.stopPropagation()}
              rows={2}
              className="mt-1 -mx-1 w-[calc(100%+0.5rem)] resize-none bg-transparent px-1 text-[12px] leading-snug text-muted-foreground outline-none focus:bg-accent/60"
              aria-label="Line description"
            />
          ) : (
            <p className="mt-1 text-[12px] leading-snug text-muted-foreground">{line.description}</p>
          )}
        </div>

        <div className="flex shrink-0 items-center gap-2" onClick={(event) => event.stopPropagation()}>
          {emphasized && editing && onDoneEditing ? (
            <button type="button" className={quietActionClass} onClick={onDoneEditing}>
              Done
            </button>
          ) : null}
          {showMenu ? (
            <EntityActionsMenu
              label={line.name}
              onEdit={onEdit}
              onPause={onRequestPause}
              onResume={onRequestResume}
              onDelete={onDelete}
              onDuplicate={onDuplicate}
              hoverReveal
            />
          ) : null}
        </div>
      </div>

      {editing ? (
        <div className="mt-3 space-y-2 border-t border-border pt-3" onClick={(event) => event.stopPropagation()}>
          {line.phases.map((phase, index) => (
            <div key={phase.id} className="group/phase flex items-start gap-2">
              <span className="mt-1.5 flex size-5 shrink-0 items-center justify-center rounded-full bg-accent text-[11px] font-medium text-muted-foreground">
                {index + 1}
              </span>
              <div className="min-w-0 flex-1">
                <input
                  value={phase.name}
                  onChange={(event) => onUpdatePhase?.(phase.id, { name: event.target.value })}
                  className="w-full bg-transparent text-[13px] font-medium tracking-[-0.01em] text-foreground outline-none focus:bg-accent/60"
                  aria-label={`Phase ${index + 1} name`}
                />
                <textarea
                  value={phase.summary}
                  onChange={(event) => onUpdatePhase?.(phase.id, { summary: event.target.value })}
                  rows={2}
                  className="mt-0.5 w-full resize-none bg-transparent text-[12px] leading-snug text-muted-foreground outline-none focus:bg-accent/60"
                  aria-label={`${phase.name || `Phase ${index + 1}`} description`}
                />
              </div>
              {onDeletePhase && line.phases.length > 1 ? (
                <button
                  type="button"
                  className="mt-1.5 cursor-pointer text-[12px] text-muted-foreground opacity-0 transition-opacity group-hover/phase:opacity-100 hover:text-foreground focus-visible:opacity-100"
                  onClick={() => onDeletePhase(phase.id)}
                >
                  Remove
                </button>
              ) : null}
            </div>
          ))}
          {onAddPhase ? (
            <button type="button" onClick={onAddPhase} className={quietActionClass}>
              + Add a stage
            </button>
          ) : null}
        </div>
      ) : (
        <StageStrip phases={line.phases} />
      )}
    </div>
  );
}

function StageBoard({
  phases,
  onWorkOrderClick,
  onEditPhase,
  onRequestPausePhase,
  onRequestResumePhase,
  onDeletePhase,
  onDuplicatePhase,
}: {
  phases: Phase[];
  onWorkOrderClick: (phase: Phase, workOrder: PhaseWorkOrder) => void;
  onEditPhase: (phase: Phase) => void;
  onRequestPausePhase: (phase: Phase) => void;
  onRequestResumePhase: (phase: Phase) => void;
  onDeletePhase: (phase: Phase) => void;
  onDuplicatePhase: (phase: Phase) => void;
}) {
  return (
    <div className="grid w-full gap-3" style={{ gridTemplateColumns: `repeat(${phases.length}, minmax(0, 1fr))` }}>
      {phases.map((phase) => (
        <section
          key={phase.id}
          className={cn(
            "flex min-w-0 flex-col rounded-lg border border-border bg-background",
            phase.paused && "opacity-80",
          )}
          aria-label={`${phase.name} stage`}
        >
          <div className="flex h-9 items-center justify-between gap-2 border-b border-border px-3">
            <div className="flex min-w-0 items-center gap-2">
              <h2 className="truncate text-[12px] font-medium tracking-[-0.01em] text-foreground">{phase.name}</h2>
              {phase.paused ? (
                <span className="shrink-0 rounded-md border border-border bg-accent px-1.5 py-0.5 text-[10px] font-medium tracking-[-0.01em] text-muted-foreground">
                  Paused
                </span>
              ) : null}
            </div>
            <EntityActionsMenu
              label={phase.name}
              onEdit={() => onEditPhase(phase)}
              onPause={() => onRequestPausePhase(phase)}
              onResume={() => onRequestResumePhase(phase)}
              onDelete={() => onDeletePhase(phase)}
              onDuplicate={() => onDuplicatePhase(phase)}
              deleteDisabled={phases.length <= 1}
            />
          </div>
          <ul className="flex min-h-[120px] flex-col gap-2 p-2">
            {phase.workOrders.length === 0 ? (
              <li className="px-2 py-4 text-[12px] text-muted-foreground">No work orders</li>
            ) : (
              phase.workOrders.map((item) => (
                <li key={item.id}>
                  <button
                    type="button"
                    onClick={() => onWorkOrderClick(phase, item)}
                    className="w-full cursor-pointer rounded-md border border-border bg-background px-3 py-2.5 text-left transition-colors hover:border-foreground/25 hover:bg-accent/40"
                  >
                    <div className="text-[13px] font-medium tracking-[-0.01em] text-foreground">{item.title}</div>
                    <div className="mt-1.5 flex items-center gap-1.5 text-[12px] text-muted-foreground">
                      <span
                        className={cn(
                          "size-1.5 shrink-0 rounded-full",
                          item.status === "running" && "bg-[#3b82f6] animate-[status-blink_1.2s_ease-in-out_infinite]",
                          item.status === "waiting" && "bg-[#f59e0b]",
                          item.status === "queued" && "bg-[#a3a3a3]",
                        )}
                        aria-hidden
                      />
                      <span>
                        {item.status === "running" ? "Executing" : item.status === "waiting" ? "Needs input" : "Queued"}{" "}
                        · {item.meta}
                      </span>
                    </div>
                  </button>
                </li>
              ))
            )}
          </ul>
        </section>
      ))}
    </div>
  );
}

export function LinesPage({
  initialSelectedLineId = null,
}: {
  initialSelectedLineId?: string | null;
} = {}) {
  const navigate = useNavigate();
  const { organizationId, factoryId } = useParams<{ organizationId?: string; factoryId?: string }>();
  const [lines, setLines] = useState<Line[]>(INITIAL_LINES);
  const [selectedLineId, setSelectedLineId] = useState<string | null>(initialSelectedLineId);
  const [editing, setEditing] = useState(false);
  const [confirm, setConfirm] = useState<ConfirmAction | null>(null);
  const [openWorkOrder, setOpenWorkOrder] = useState<{
    phase: Phase;
    workOrder: PhaseWorkOrder;
  } | null>(null);

  const selectedLine = selectedLineId == null ? null : (lines.find((line) => line.id === selectedLineId) ?? null);

  function configurePhase(phase: Phase) {
    if (!selectedLine) return;
    const base = organizationId && factoryId ? `/${organizationId}/workspaces/${factoryId}/lines` : "/lines";
    navigate(`${base}/${selectedLine.id}/phases/${phase.id}/configure`, {
      state: {
        lineName: selectedLine.name,
        phaseName: phase.name,
      },
    });
  }

  function updateSelectedLine(patch: Partial<Pick<Line, "name" | "description">>) {
    if (!selectedLineId) return;
    setLines((current) => current.map((line) => (line.id === selectedLineId ? { ...line, ...patch } : line)));
  }

  function updatePhase(phaseId: string, patch: Partial<Pick<Phase, "name" | "summary">>) {
    if (!selectedLineId) return;
    setLines((current) =>
      current.map((line) =>
        line.id === selectedLineId
          ? {
              ...line,
              phases: line.phases.map((phase) => (phase.id === phaseId ? { ...phase, ...patch } : phase)),
            }
          : line,
      ),
    );
  }

  function addPhase() {
    if (!selectedLineId) return;
    const phase: Phase = {
      id: `phase-${Date.now()}`,
      name: "New stage",
      summary: "Describe what this stage does.",
      paused: false,
      workOrders: [],
    };
    setLines((current) =>
      current.map((line) => (line.id === selectedLineId ? { ...line, phases: [...line.phases, phase] } : line)),
    );
  }

  function deletePhase(phaseId: string) {
    if (!selectedLineId) return;
    setLines((current) =>
      current.map((line) =>
        line.id === selectedLineId
          ? {
              ...line,
              phases: line.phases.length <= 1 ? line.phases : line.phases.filter((phase) => phase.id !== phaseId),
            }
          : line,
      ),
    );
  }

  function duplicatePhase(phaseId: string) {
    if (!selectedLineId) return;
    setLines((current) =>
      current.map((line) => {
        if (line.id !== selectedLineId) return line;
        const source = line.phases.find((phase) => phase.id === phaseId);
        if (!source) return line;
        const stamp = Date.now();
        const copy: Phase = {
          ...source,
          id: `${source.id}-${stamp}`,
          name: `${source.name} copy`,
          paused: false,
          workOrders: source.workOrders.map((item) => ({
            ...item,
            id: `${item.id}-${stamp}`,
          })),
        };
        const index = line.phases.findIndex((phase) => phase.id === phaseId);
        const phases = [...line.phases];
        phases.splice(index + 1, 0, copy);
        return { ...line, phases };
      }),
    );
  }

  function editLine(lineId: string) {
    setSelectedLineId(lineId);
    setEditing(true);
    setOpenWorkOrder(null);
  }

  function duplicateLine(lineId: string) {
    const source = lines.find((line) => line.id === lineId);
    if (!source) return;
    const id = `line-${Date.now()}`;
    const copy: Line = {
      ...source,
      id,
      name: `${source.name} copy`,
      paused: false,
      phases: source.phases.map((phase) => ({
        ...phase,
        id: `${phase.id}-${Date.now()}`,
        paused: false,
        workOrders: phase.workOrders.map((item) => ({
          ...item,
          id: `${item.id}-${Date.now()}`,
        })),
      })),
    };
    setLines((current) => [...current, copy]);
    setSelectedLineId(id);
    setEditing(true);
    setOpenWorkOrder(null);
  }

  function deleteLine(lineId: string) {
    setLines((current) => current.filter((line) => line.id !== lineId));
    if (selectedLineId === lineId) {
      setSelectedLineId(null);
      setEditing(false);
      setOpenWorkOrder(null);
    }
  }

  function setLinePaused(lineId: string, paused: boolean) {
    setLines((current) => current.map((line) => (line.id === lineId ? { ...line, paused } : line)));
  }

  function setPhasePaused(lineId: string, phaseId: string, paused: boolean) {
    setLines((current) =>
      current.map((line) =>
        line.id === lineId
          ? {
              ...line,
              phases: line.phases.map((phase) => (phase.id === phaseId ? { ...phase, paused } : phase)),
            }
          : line,
      ),
    );
  }

  function requestPauseLine(line: Line) {
    setConfirm({
      kind: "pause",
      target: "line",
      lineId: line.id,
      name: line.name,
    });
  }

  function requestResumeLine(line: Line) {
    setConfirm({
      kind: "resume",
      target: "line",
      lineId: line.id,
      name: line.name,
    });
  }

  function requestPausePhase(phase: Phase) {
    if (!selectedLineId) return;
    setConfirm({
      kind: "pause",
      target: "phase",
      lineId: selectedLineId,
      phaseId: phase.id,
      name: phase.name,
    });
  }

  function requestResumePhase(phase: Phase) {
    if (!selectedLineId) return;
    setConfirm({
      kind: "resume",
      target: "phase",
      lineId: selectedLineId,
      phaseId: phase.id,
      name: phase.name,
    });
  }

  function confirmPauseResume() {
    if (!confirm) return;
    if (confirm.target === "phase" && confirm.phaseId) {
      setPhasePaused(confirm.lineId, confirm.phaseId, confirm.kind === "pause");
    } else {
      setLinePaused(confirm.lineId, confirm.kind === "pause");
    }
    setConfirm(null);
  }

  function addLine() {
    const id = `line-${Date.now()}`;
    const line: Line = {
      id,
      name: "New line",
      description: "Describe what this factory line does.",
      icon: Workflow,
      paused: false,
      phases: [
        {
          id: `phase-${Date.now()}`,
          name: "New stage",
          summary: "Describe what this stage does.",
          paused: false,
          workOrders: [],
        },
      ],
    };

    setLines((current) => [...current, line]);
    setSelectedLineId(id);
    setEditing(true);
    setOpenWorkOrder(null);
  }

  function lineMenuProps(line: Line) {
    return {
      onEdit: () => editLine(line.id),
      onDuplicate: () => duplicateLine(line.id),
      onDelete: () => deleteLine(line.id),
      onRequestPause: () => requestPauseLine(line),
      onRequestResume: () => requestResumeLine(line),
    };
  }

  if (openWorkOrder && selectedLine) {
    return (
      <div className="absolute inset-0 flex flex-col bg-background">
        <div className="shrink-0 border-b border-border px-5 py-3">
          <button
            type="button"
            onClick={() => setOpenWorkOrder(null)}
            className="mb-2 inline-flex cursor-pointer items-center gap-1.5 text-[13px] text-muted-foreground transition-colors hover:text-foreground"
          >
            <ArrowLeft className="size-3.5" strokeWidth={1.75} />
            {openWorkOrder.phase.name}
          </button>
          <div className="min-w-0">
            <h2 className="text-[15px] font-semibold tracking-[-0.01em] text-foreground">
              {openWorkOrder.workOrder.title}
            </h2>
            <p className="mt-0.5 text-[13px] text-muted-foreground">Canvas · {selectedLine.name}</p>
          </div>
        </div>
        <div className="min-h-0 flex-1">
          <WorkOrderCanvas />
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto w-full max-w-[960px] px-8 py-8">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <h1 className="text-[22px] font-semibold tracking-[-0.02em] text-foreground">Lines</h1>
          <p className="mt-1.5 max-w-2xl text-[13px] text-muted-foreground">
            Factory lines specialize how work moves through the workspace. Each phase is backed by a canvas that runs
            work orders.
          </p>
        </div>
        {selectedLine == null ? (
          <button
            type="button"
            onClick={addLine}
            className="inline-flex h-8 shrink-0 items-center gap-1.5 rounded-md border border-border px-3 text-[13px] text-foreground transition-colors hover:bg-accent"
          >
            <Plus className="size-3.5" strokeWidth={1.75} />
            Add line
          </button>
        ) : null}
      </div>

      {selectedLine ? (
        <div className="mt-6">
          <button
            type="button"
            onClick={() => {
              setSelectedLineId(null);
              setEditing(false);
              setOpenWorkOrder(null);
            }}
            className="mb-3 inline-flex cursor-pointer items-center gap-1.5 text-[13px] text-muted-foreground transition-colors hover:text-foreground"
          >
            <ArrowLeft className="size-3.5" strokeWidth={1.75} />
            All lines
          </button>
          <LineCard
            line={selectedLine}
            emphasized
            editing={editing}
            onDoneEditing={() => setEditing(false)}
            onUpdateLine={updateSelectedLine}
            onUpdatePhase={updatePhase}
            onAddPhase={addPhase}
            onDeletePhase={deletePhase}
            {...lineMenuProps(selectedLine)}
          />
          {!editing ? (
            <Tabs defaultValue="velocity" className="mt-6 gap-0">
              <TabsList className="h-auto w-fit justify-start gap-5 rounded-none border-b border-border bg-transparent p-0">
                <TabsTrigger value="stages" className={lineDetailTabTriggerClass}>
                  Stages
                </TabsTrigger>
                <TabsTrigger value="velocity" className={lineDetailTabTriggerClass}>
                  Velocity
                </TabsTrigger>
              </TabsList>
              <TabsContent value="stages" className="mt-4 outline-none">
                <StageBoard
                  phases={selectedLine.phases}
                  onWorkOrderClick={(phase, workOrder) => setOpenWorkOrder({ phase, workOrder })}
                  onEditPhase={configurePhase}
                  onRequestPausePhase={requestPausePhase}
                  onRequestResumePhase={requestResumePhase}
                  onDeletePhase={(phase) => deletePhase(phase.id)}
                  onDuplicatePhase={(phase) => duplicatePhase(phase.id)}
                />
              </TabsContent>
              <TabsContent value="velocity" className="mt-4 outline-none">
                <LineVelocityPanel stageNames={selectedLine.phases.map((phase) => phase.name)} />
              </TabsContent>
            </Tabs>
          ) : null}
        </div>
      ) : (
        <div className="mt-6 flex flex-col gap-2">
          {lines.map((line) => (
            <LineCard
              key={line.id}
              line={line}
              onClick={() => {
                setSelectedLineId(line.id);
                setEditing(false);
              }}
              {...lineMenuProps(line)}
            />
          ))}
        </div>
      )}

      <PauseResumeDialog
        confirm={confirm}
        onOpenChange={(open) => {
          if (!open) setConfirm(null);
        }}
        onConfirm={confirmPauseResume}
      />
    </div>
  );
}
