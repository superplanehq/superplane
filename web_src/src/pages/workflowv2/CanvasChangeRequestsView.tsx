import {
  CanvasesCanvasChangeRequest,
  CanvasesCanvasChangeRequestApprovalConfig,
  CanvasesCanvasVersion,
} from "@/api-client";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { ArrowLeft } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import {
  buildInitials,
  ChangeRequestDescriptionCard,
  formatTimestamp as formatDisplayTimestamp,
  summarizeNodeDiff,
  VersionNodeDiffAccordion,
} from "./VersionNodeDiff";
import { Avatar, AvatarFallback, AvatarImage } from "@/ui/avatar";
import { CanvasChangeRequestConflictResolver } from "./CanvasChangeRequestConflictResolver";

type ChangeRequestFilter = "all" | "open" | "rejected" | "published";

interface CanvasChangeRequestsViewProps {
  changeRequests: CanvasesCanvasChangeRequest[];
  canvasVersions?: CanvasesCanvasVersion[];
  selectedChangeRequestId?: string;
  canUpdateCanvas: boolean;
  currentUserId?: string;
  actionPending: boolean;
  resolvePending: boolean;
  liveCanvasVersion?: CanvasesCanvasVersion;
  changeRequestApprovalConfig?: CanvasesCanvasChangeRequestApprovalConfig;
  ownerProfilesByID?: Map<string, { name: string; avatarUrl?: string }>;
  roleDisplayNamesByName?: Map<string, string>;
  canvasName: string;
  canvasDescription?: string;
  onSelectChangeRequest: (changeRequestId: string) => void;
  onApprove: (changeRequestId: string) => Promise<void>;
  onUnapprove: (changeRequestId: string) => Promise<void>;
  onPublish: (changeRequestId: string) => Promise<void>;
  onReject: (changeRequestId: string) => Promise<void>;
  onReopen: (changeRequestId: string) => Promise<void>;
  onResolve: (data: {
    changeRequestId: string;
    nodes: Record<string, unknown>[];
    edges: Record<string, unknown>[];
  }) => Promise<void>;
}

function normalizeStatus(status?: string): "open" | "published" | "rejected" | "unknown" {
  const value = (status || "").toLowerCase();
  if (value.includes("open")) return "open";
  if (value.includes("publish")) return "published";
  if (value.includes("reject")) return "rejected";
  return "unknown";
}

function formatStatusLabel(status: ReturnType<typeof normalizeStatus>): string {
  if (status === "open") return "Open";
  if (status === "published") return "Published";
  if (status === "rejected") return "Rejected";
  return "Unknown";
}

function statusBadgeVariant(
  status: ReturnType<typeof normalizeStatus>,
): "default" | "secondary" | "destructive" | "outline" {
  if (status === "published") return "default";
  if (status === "rejected") return "destructive";
  return "outline";
}

function normalizeApprovalState(state?: string): "approved" | "rejected" | "unapproved" | "unknown" {
  const value = (state || "").toLowerCase();
  if (value.includes("unapproved")) return "unapproved";
  if (value.includes("approved")) return "approved";
  if (value.includes("rejected")) return "rejected";
  return "unknown";
}

function resolveUserDisplay(
  userRef: { id?: string; name?: string } | undefined,
  profilesByID?: Map<string, { name: string; avatarUrl?: string }>,
): { name: string; avatarUrl?: string; id?: string } {
  const userID = userRef?.id || "";
  const profile = userID ? profilesByID?.get(userID) : undefined;
  return {
    id: userID || undefined,
    name: userRef?.name || profile?.name || "Unknown user",
    avatarUrl: profile?.avatarUrl,
  };
}

function isChangeRequestConflicted(changeRequest?: CanvasesCanvasChangeRequest): boolean {
  if (!changeRequest) {
    return false;
  }

  if (typeof changeRequest.metadata?.isConflicted === "boolean") {
    return changeRequest.metadata.isConflicted;
  }

  return (changeRequest.diff?.conflictingNodeIds || []).length > 0;
}

function formatTimestamp(value?: string): string {
  return formatDisplayTimestamp(value) || "unknown time";
}

export function CanvasChangeRequestsView({
  changeRequests,
  canvasVersions = [],
  selectedChangeRequestId,
  canUpdateCanvas,
  currentUserId,
  actionPending,
  resolvePending,
  liveCanvasVersion,
  changeRequestApprovalConfig,
  ownerProfilesByID,
  roleDisplayNamesByName,
  canvasName,
  canvasDescription,
  onSelectChangeRequest,
  onApprove,
  onUnapprove,
  onPublish,
  onReject,
  onReopen,
  onResolve,
}: CanvasChangeRequestsViewProps) {
  const [filter, setFilter] = useState<ChangeRequestFilter>("open");
  const [resolvingChangeRequestID, setResolvingChangeRequestID] = useState("");
  const [showDetailView, setShowDetailView] = useState(Boolean(selectedChangeRequestId));

  const filteredRequests = useMemo(() => {
    if (filter === "all") {
      return changeRequests;
    }
    return changeRequests.filter((item) => normalizeStatus(item.metadata?.status) === filter);
  }, [changeRequests, filter]);

  const selectedChangeRequest = useMemo(() => {
    const selected = changeRequests.find((item) => item.metadata?.id === selectedChangeRequestId);
    if (selected) {
      return selected;
    }
    return filteredRequests[0];
  }, [changeRequests, filteredRequests, selectedChangeRequestId]);

  useEffect(() => {
    if (!selectedChangeRequestId) {
      setShowDetailView(false);
    }
  }, [selectedChangeRequestId]);

  const resolvingChangeRequest = useMemo(
    () => changeRequests.find((changeRequest) => changeRequest.metadata?.id === resolvingChangeRequestID),
    [changeRequests, resolvingChangeRequestID],
  );

  useEffect(() => {
    if (!resolvingChangeRequestID) {
      return;
    }

    const stillExists = changeRequests.some((changeRequest) => changeRequest.metadata?.id === resolvingChangeRequestID);
    if (!stillExists) {
      setResolvingChangeRequestID("");
    }
  }, [changeRequests, resolvingChangeRequestID]);

  const selectedStatus = normalizeStatus(selectedChangeRequest?.metadata?.status);
  const selectedChangeRequestIdSafe = selectedChangeRequest?.metadata?.id || "";
  const conflictingNodeIds = selectedChangeRequest?.diff?.conflictingNodeIds || [];
  const canvasVersionsByID = useMemo(() => {
    const result = new Map<string, CanvasesCanvasVersion>();
    canvasVersions.forEach((version) => {
      const id = version.metadata?.id || "";
      if (!id) {
        return;
      }
      result.set(id, version);
    });
    if (liveCanvasVersion?.metadata?.id) {
      result.set(liveCanvasVersion.metadata.id, liveCanvasVersion);
    }
    return result;
  }, [canvasVersions, liveCanvasVersion]);
  const selectedBasedOnVersion = useMemo(() => {
    const basedOnVersionID = selectedChangeRequest?.metadata?.basedOnVersionId || "";
    if (!basedOnVersionID) {
      return undefined;
    }

    return canvasVersionsByID.get(basedOnVersionID);
  }, [canvasVersionsByID, selectedChangeRequest?.metadata?.basedOnVersionId]);
  const selectedPublishedPreviousVersion = useMemo(() => {
    if (selectedStatus !== "published") {
      return undefined;
    }

    const selectedVersionID =
      selectedChangeRequest?.version?.metadata?.id || selectedChangeRequest?.metadata?.versionId || "";
    if (!selectedVersionID) {
      return undefined;
    }

    const selectedVersionIndex = canvasVersions.findIndex((version) => version.metadata?.id === selectedVersionID);
    if (selectedVersionIndex < 0) {
      return undefined;
    }

    return canvasVersions[selectedVersionIndex + 1];
  }, [
    canvasVersions,
    selectedChangeRequest?.metadata?.versionId,
    selectedChangeRequest?.version?.metadata?.id,
    selectedStatus,
  ]);
  const selectedApprovals = selectedChangeRequest?.approvals || [];
  const activeApprovals = useMemo(
    () => selectedApprovals.filter((approval) => !approval.invalidatedAt),
    [selectedApprovals],
  );
  const selectedHasConflicts = isChangeRequestConflicted(selectedChangeRequest);
  const selectedDiffSummary = useMemo(
    () =>
      summarizeNodeDiff(
        selectedChangeRequest?.version,
        selectedBasedOnVersion || selectedPublishedPreviousVersion || liveCanvasVersion,
      ),
    [selectedBasedOnVersion, selectedChangeRequest?.version, selectedPublishedPreviousVersion, liveCanvasVersion],
  );
  const selectedConflictingNodeIDSet = useMemo(() => new Set(conflictingNodeIds), [conflictingNodeIds]);
  const requestedBy = useMemo(
    () => resolveUserDisplay(selectedChangeRequest?.metadata?.owner, ownerProfilesByID),
    [selectedChangeRequest?.metadata?.owner, ownerProfilesByID],
  );
  const requiredApprovalsCount = useMemo(() => {
    const configuredCount = changeRequestApprovalConfig?.items?.length || 0;
    return configuredCount > 0 ? configuredCount : 1;
  }, [changeRequestApprovalConfig?.items]);
  const activeApprovedCount = useMemo(
    () => activeApprovals.filter((approval) => normalizeApprovalState(approval.state) === "approved").length,
    [activeApprovals],
  );
  const hasCurrentUserActiveApproval = useMemo(() => {
    if (!currentUserId) {
      return false;
    }

    return activeApprovals.some(
      (approval) => normalizeApprovalState(approval.state) === "approved" && approval.actor?.id === currentUserId,
    );
  }, [activeApprovals, currentUserId]);
  const approvalRequirementsSatisfied = activeApprovedCount >= requiredApprovalsCount;
  const activityItems = useMemo(() => {
    const openedAt = formatTimestamp(selectedChangeRequest?.metadata?.createdAt);
    const items: Array<{
      id: string;
      title: string;
      detail: string;
      timestamp: string;
      tone: "slate" | "emerald" | "rose" | "amber";
      invalidated?: boolean;
      actor?: {
        name: string;
        avatarUrl?: string;
      };
    }> = [];

    items.push({
      id: "opened",
      title: "Opened",
      detail: "opened this change request.",
      timestamp: openedAt || "unknown time",
      tone: "slate",
      actor: {
        name: requestedBy.name,
        avatarUrl: requestedBy.avatarUrl,
      },
    });

    selectedApprovals.forEach((approval, index) => {
      const state = normalizeApprovalState(approval.state);
      if (state === "unknown") {
        return;
      }

      const actor = resolveUserDisplay(approval.actor, ownerProfilesByID);
      const approverType = approval.approver?.type || "";
      const roleName = approval.approver?.roleName || "";
      const roleDisplayName = roleDisplayNamesByName?.get(roleName) || roleName;
      let detail = "updated approval state.";
      let title = "Approval Updated";
      let tone: "slate" | "emerald" | "rose" | "amber" = "slate";
      let invalidated = false;

      if (state === "approved") {
        title = "Approved";
        detail = "approved this change request.";
        tone = "emerald";
      } else if (state === "rejected") {
        title = "Rejected";
        detail = "rejected this change request.";
        tone = "rose";
      } else if (state === "unapproved") {
        title = "Unapproved";
        detail = "removed their approval.";
        tone = "slate";
      }

      if (approverType === "TYPE_ROLE" && roleName) {
        detail = `${detail} (role: ${roleDisplayName})`;
      }
      if (approval.invalidatedAt && state === "approved") {
        invalidated = true;
      }
      if (approval.invalidatedAt && state === "approved") {
        tone = "amber";
      }

      items.push({
        id: `approval-${approval.createdAt || index}-${state}`,
        title,
        detail,
        timestamp: formatTimestamp(approval.createdAt) || "unknown time",
        tone,
        invalidated,
        actor: {
          name: actor.name,
          avatarUrl: actor.avatarUrl,
        },
      });
    });

    if (selectedStatus === "published") {
      const publishedAt = formatTimestamp(
        selectedChangeRequest?.metadata?.publishedAt || selectedChangeRequest?.metadata?.updatedAt,
      );
      items.push({
        id: "published",
        title: "Published",
        detail: "This change request was published to live.",
        timestamp: publishedAt || "unknown time",
        tone: "emerald",
      });
    }

    return items;
  }, [
    selectedChangeRequest?.metadata?.createdAt,
    selectedChangeRequest?.metadata?.publishedAt,
    selectedChangeRequest?.metadata?.updatedAt,
    selectedStatus,
    selectedApprovals,
    ownerProfilesByID,
    roleDisplayNamesByName,
    requestedBy.avatarUrl,
    requestedBy.name,
  ]);

  const canApprove =
    canUpdateCanvas && selectedStatus === "open" && !selectedHasConflicts && !hasCurrentUserActiveApproval;
  const canUnapprove = canUpdateCanvas && selectedStatus === "open" && hasCurrentUserActiveApproval;
  const canPublish =
    canUpdateCanvas && selectedStatus === "open" && !selectedHasConflicts && approvalRequirementsSatisfied;
  const canReject = canUpdateCanvas && selectedStatus === "open";
  const canReopen = canUpdateCanvas && selectedStatus === "rejected";
  const hasChangeRequestID = !!selectedChangeRequestIdSafe;
  const showPublishAction = canPublish && !actionPending && hasChangeRequestID;
  const showApproveAction = canApprove && !actionPending && hasChangeRequestID;
  const showUnapproveAction = canUnapprove && !actionPending && hasChangeRequestID;
  const showRejectAction = canReject && !actionPending && hasChangeRequestID;
  const showReopenAction = canReopen && !actionPending && hasChangeRequestID;
  const hasReviewActions =
    showPublishAction || showApproveAction || showUnapproveAction || showRejectAction || showReopenAction;
  const showReviewActionsCard = selectedStatus !== "published" && hasReviewActions;
  const canResolveConflicts =
    canUpdateCanvas &&
    selectedStatus === "open" &&
    selectedHasConflicts &&
    !!selectedChangeRequest?.version?.spec?.nodes &&
    !!selectedChangeRequest?.version?.spec?.edges;
  const showConflictResolutionCard = selectedHasConflicts;
  const hasSidebarContent = showReviewActionsCard || showConflictResolutionCard;

  if (resolvingChangeRequest) {
    return (
      <CanvasChangeRequestConflictResolver
        liveCanvasVersion={liveCanvasVersion}
        changeRequest={resolvingChangeRequest}
        canvasName={canvasName}
        canvasDescription={canvasDescription}
        isSubmitting={resolvePending}
        onBack={() => setResolvingChangeRequestID("")}
        onSubmit={async (data) => {
          await onResolve(data);
          setResolvingChangeRequestID("");
        }}
      />
    );
  }

  if (!showDetailView) {
    return (
      <div className="h-full overflow-auto bg-slate-50">
        <div className="mx-auto max-w-6xl space-y-4 p-5 md:p-7">
          <section className="rounded-xl border border-slate-200 bg-white">
            <div className="border-b border-slate-200 px-4 py-3">
              <div className="flex items-center justify-between gap-2">
                <div>
                  <p className="text-base font-semibold text-slate-900">Change Requests</p>
                  <p className="text-xs text-slate-600">Select a request to open it in a dedicated PR view.</p>
                </div>
                <Badge variant="outline">{changeRequests.length}</Badge>
              </div>
            </div>
            <div className="space-y-3 p-4">
              <Tabs value={filter} onValueChange={(value) => setFilter(value as ChangeRequestFilter)}>
                <TabsList className="grid w-full grid-cols-4">
                  <TabsTrigger value="open">Open</TabsTrigger>
                  <TabsTrigger value="rejected">Rejected</TabsTrigger>
                  <TabsTrigger value="published">Published</TabsTrigger>
                  <TabsTrigger value="all">All</TabsTrigger>
                </TabsList>
              </Tabs>

              <div className="max-h-[500px] overflow-auto rounded-md border border-slate-200 bg-white">
                {filteredRequests.length === 0 ? (
                  <p className="p-3 text-sm text-slate-600">No change requests in this filter.</p>
                ) : (
                  filteredRequests.map((item) => {
                    const itemId = item.metadata?.id || "";
                    const itemStatus = normalizeStatus(item.metadata?.status);
                    const conflictCount = item.diff?.conflictingNodeIds?.length || 0;
                    const hasConflicts = isChangeRequestConflicted(item);
                    const itemChangedCount = item.diff?.changedNodeIds?.length || 0;

                    return (
                      <button
                        key={itemId}
                        type="button"
                        className={cn(
                          "w-full border-b p-3 text-left last:border-b-0 hover:bg-slate-50",
                          hasConflicts ? "border-red-200 bg-red-50/40" : "border-slate-200",
                        )}
                        onClick={() => {
                          onSelectChangeRequest(itemId);
                          setShowDetailView(true);
                        }}
                      >
                        <div className="flex items-center justify-between gap-2">
                          <p className="truncate text-sm font-semibold text-slate-900">
                            {item.metadata?.title?.trim() || "Untitled change request"}
                          </p>
                          <div className="flex items-center gap-2">
                            {hasConflicts ? (
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Badge variant="destructive" className="uppercase">
                                    Conflicted
                                  </Badge>
                                </TooltipTrigger>
                                <TooltipContent>
                                  This change request is conflicted. It cannot be approved until conflicts are resolved.
                                </TooltipContent>
                              </Tooltip>
                            ) : null}
                            <Badge variant={statusBadgeVariant(itemStatus)}>{formatStatusLabel(itemStatus)}</Badge>
                          </div>
                        </div>
                        <div className="mt-1 flex flex-wrap items-center gap-3 text-xs text-slate-600">
                          <span>changed nodes: {itemChangedCount}</span>
                          <span className={hasConflicts ? "font-semibold text-red-700" : "text-emerald-700"}>
                            conflicts: {conflictCount}
                          </span>
                          <span>updated: {formatTimestamp(item.metadata?.updatedAt)}</span>
                        </div>
                      </button>
                    );
                  })
                )}
              </div>
            </div>
          </section>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full overflow-auto bg-slate-50">
      <div className="mx-auto max-w-6xl space-y-4 p-5 md:p-7">
        {!selectedChangeRequest ? (
          <section className="rounded-xl border border-slate-200 bg-white p-4">
            <Button variant="ghost" size="sm" className="px-1" onClick={() => setShowDetailView(false)}>
              <ArrowLeft className="h-4 w-4" />
              Back to Change Requests
            </Button>
            <p className="mt-2 text-sm text-slate-600">This change request is no longer available.</p>
          </section>
        ) : (
          <>
            <section className="rounded-xl border border-slate-200 bg-white p-4">
              <Button variant="ghost" size="sm" className="mb-3 px-1" onClick={() => setShowDetailView(false)}>
                <ArrowLeft className="h-4 w-4" />
                Back to Change Requests
              </Button>
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="space-y-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <h3 className="text-lg font-semibold text-slate-900">
                      {selectedChangeRequest.metadata?.title?.trim() || "Untitled change request"}
                    </h3>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {selectedHasConflicts ? (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Badge variant="destructive" className="uppercase">
                          Conflicted
                        </Badge>
                      </TooltipTrigger>
                      <TooltipContent>
                        This change request is conflicted. It cannot be approved until conflicts are resolved.
                      </TooltipContent>
                    </Tooltip>
                  ) : null}
                  <Badge variant={statusBadgeVariant(selectedStatus)}>{formatStatusLabel(selectedStatus)}</Badge>
                </div>
              </div>

              <div className="mt-4">
                <ChangeRequestDescriptionCard
                  ownerName={requestedBy.name}
                  ownerAvatarUrl={requestedBy.avatarUrl}
                  timestamp={formatTimestamp(selectedChangeRequest.metadata?.createdAt)}
                  actionLabel="requested changes"
                  content={selectedChangeRequest.metadata?.description?.trim() || "No description provided."}
                />
              </div>
            </section>

            <div className={cn("grid gap-4", hasSidebarContent ? "lg:grid-cols-[1fr_280px]" : "lg:grid-cols-1")}>
              <section className="space-y-4">
                <div className="rounded-xl border border-slate-200 bg-white p-4">
                  <p className="text-sm font-semibold text-slate-900">Summary</p>
                  <div className="mt-4">
                    <VersionNodeDiffAccordion
                      summary={selectedDiffSummary}
                      conflictingNodeIDs={selectedConflictingNodeIDSet}
                      emptyMessage="No node-level differences found."
                    />
                  </div>
                </div>

                <div className="rounded-xl border border-slate-200 bg-white p-4">
                  <p className="text-sm font-semibold text-slate-900">Activity</p>
                  <ol className="mt-3 space-y-3">
                    {activityItems.map((item, index) => (
                      <li key={item.id} className="relative flex items-start gap-3">
                        <div className="relative flex w-3 justify-center">
                          {index < activityItems.length - 1 ? (
                            <span className="absolute left-1/2 top-4 h-[calc(100%+1.5rem)] w-px -translate-x-1/2 bg-slate-200" />
                          ) : null}
                          <span
                            className={cn(
                              "mt-1 h-2.5 w-2.5 rounded-full",
                              item.tone === "emerald"
                                ? "bg-emerald-500"
                                : item.tone === "rose"
                                  ? "bg-rose-500"
                                  : item.tone === "amber"
                                    ? "bg-amber-500"
                                    : "bg-slate-400",
                            )}
                          />
                        </div>
                        <div className="min-w-0">
                          <p className="text-sm font-medium text-slate-900">
                            {item.title}
                            {item.invalidated ? (
                              <span className="ml-1 text-xs text-amber-600">(invalidated)</span>
                            ) : null}
                            <span className="text-xs font-normal text-slate-500">· {item.timestamp}</span>
                          </p>
                          {item.actor ? (
                            <p className="flex items-center gap-1.5 text-xs text-slate-600 mt-1">
                              <Avatar className="h-4 w-4">
                                <AvatarImage src={item.actor.avatarUrl} alt={item.actor.name} />
                                <AvatarFallback className="text-[8px] font-medium">
                                  {buildInitials(item.actor.name)}
                                </AvatarFallback>
                              </Avatar>
                              <span className="font-bold text-slate-900">{item.actor.name}</span>
                              <span>{item.detail}</span>
                            </p>
                          ) : (
                            <p className="text-xs text-slate-600">{item.detail}</p>
                          )}
                        </div>
                      </li>
                    ))}
                  </ol>
                </div>
              </section>

              {hasSidebarContent ? (
                <aside className="space-y-3">
                  {showReviewActionsCard ? (
                    <div className="rounded-xl border border-slate-200 bg-white p-4">
                      <p className="text-sm font-semibold text-slate-900">Review Actions</p>
                      <p className="mt-1 text-xs text-slate-600">
                        Active approvals: {activeApprovedCount}/{requiredApprovalsCount}
                      </p>
                      <div className="mt-3 space-y-2">
                        {showPublishAction ? (
                          <Button
                            className="w-full justify-center"
                            onClick={() => onPublish(selectedChangeRequestIdSafe)}
                          >
                            Publish
                          </Button>
                        ) : null}
                        {showApproveAction ? (
                          <Button
                            className="w-full justify-center"
                            variant="secondary"
                            onClick={() => onApprove(selectedChangeRequestIdSafe)}
                          >
                            Approve
                          </Button>
                        ) : null}
                        {showUnapproveAction ? (
                          <Button
                            className="w-full justify-center"
                            variant="outline"
                            onClick={() => onUnapprove(selectedChangeRequestIdSafe)}
                          >
                            Unapprove
                          </Button>
                        ) : null}
                        {showRejectAction ? (
                          <Button
                            className="w-full justify-center"
                            variant="destructive"
                            onClick={() => onReject(selectedChangeRequestIdSafe)}
                          >
                            Reject
                          </Button>
                        ) : null}
                        {showReopenAction ? (
                          <Button
                            className="w-full justify-center"
                            variant="outline"
                            onClick={() => onReopen(selectedChangeRequestIdSafe)}
                          >
                            Reopen
                          </Button>
                        ) : null}
                      </div>
                    </div>
                  ) : null}

                  {showConflictResolutionCard ? (
                    <div className="rounded-xl border border-slate-200 bg-white p-4">
                      <p className="text-sm font-semibold text-slate-900">Conflict Resolution</p>
                      <p className="mt-1 text-xs text-slate-600">
                        Conflicts found in this request. Open resolver to merge node changes.
                      </p>
                      <Button
                        className="mt-3 w-full justify-center"
                        variant="secondary"
                        onClick={() => setResolvingChangeRequestID(selectedChangeRequestIdSafe)}
                        disabled={!canResolveConflicts || resolvePending}
                      >
                        Resolve Conflicts
                      </Button>
                    </div>
                  ) : null}
                </aside>
              ) : null}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
