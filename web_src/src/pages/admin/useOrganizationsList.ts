import { useCallback, useEffect, useRef, useState } from "react";

import type { SortDirection } from "./SortableHeader";

export interface AdminOrganization {
  id: string;
  name: string;
  plan_key?: string;
  plan_name?: string;
  canvas_count: number;
  task_count: number;
  done_task_count: number;
  member_count: number;
  workspace_count: number;
  repository_count: number;
  remaining_credit_cents?: number;
  created_at?: string;
}

export type OrganizationSortField =
  | "canvas_count"
  | "created_at"
  | "done_task_count"
  | "member_count"
  | "name"
  | "plan_name"
  | "remaining_credit_cents"
  | "repository_count"
  | "task_count"
  | "workspace_count";

export const ORGANIZATION_PAGE_SIZE = 50;

interface OrganizationListQuery {
  search: string;
  planKey: string;
  offset: number;
  sortBy: OrganizationSortField;
  sortDirection: SortDirection;
}

interface OrganizationListResponse {
  items: AdminOrganization[];
  pinned: AdminOrganization[];
  total: number;
  matchTotal: number;
}

function listFilterKey({ search, planKey, sortBy, sortDirection }: OrganizationListQuery): string {
  return `${search}\0${planKey}\0${sortBy}\0${sortDirection}`;
}

function pageOffsetForTotal(pageOffset: number, total: number): number {
  if (pageOffset <= 0 || total <= 0) {
    return 0;
  }
  const lastOffset = Math.floor((total - 1) / ORGANIZATION_PAGE_SIZE) * ORGANIZATION_PAGE_SIZE;
  return Math.min(pageOffset, lastOffset);
}

function organizationListResponse(data: {
  items?: AdminOrganization[];
  pinned?: AdminOrganization[];
  total?: number;
  match_total?: number;
}): OrganizationListResponse {
  const total = data.total ?? 0;
  return {
    items: data.items ?? [],
    pinned: data.pinned ?? [],
    total,
    matchTotal: typeof data.match_total === "number" ? data.match_total : total,
  };
}

const LIST_REFRESH_ERROR = "Could not refresh the organization list.";

type ListLoadResult = "applied" | "failed" | "stale";

async function requestOrganizationList(query: OrganizationListQuery): Promise<OrganizationListResponse | null> {
  const params = new URLSearchParams({ limit: String(ORGANIZATION_PAGE_SIZE), offset: String(query.offset) });
  if (query.search) params.set("search", query.search);
  if (query.planKey) params.set("plan_key", query.planKey);
  params.set("sort_by", query.sortBy);
  params.set("sort_direction", query.sortDirection);
  const response = await fetch(`/admin/api/organizations?${params}`, { credentials: "include" });
  if (!response.ok) {
    return null;
  }
  return organizationListResponse(await response.json());
}

export function useOrganizationsList() {
  const [organizations, setOrganizations] = useState<AdminOrganization[]>([]);
  const [pinnedOrganizations, setPinnedOrganizations] = useState<AdminOrganization[]>([]);
  const [total, setTotal] = useState(0);
  const [matchTotal, setMatchTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");
  const [planKey, setPlanKey] = useState("");
  const [loading, setLoading] = useState(true);
  const [sortBy, setSortBy] = useState<OrganizationSortField>("created_at");
  const [sortDirection, setSortDirection] = useState<SortDirection>("desc");
  const [listError, setListError] = useState<string | null>(null);
  const listRequestID = useRef(0);
  const query = { search, planKey, offset, sortBy, sortDirection };
  const appliedFilterKeyRef = useRef(listFilterKey(query));
  const listQueryRef = useRef<OrganizationListQuery>(query);
  const filterKey = listFilterKey(query);
  listQueryRef.current = filterKey === appliedFilterKeyRef.current ? query : { ...query, offset: 0 };

  const applyList = (list: OrganizationListResponse, pageOffset: number) => {
    setOffset(pageOffset);
    setOrganizations(list.items);
    setPinnedOrganizations(list.pinned);
    setTotal(list.total);
    setMatchTotal(list.matchTotal);
  };

  const fetchOrganizations = useCallback(
    async (query: OrganizationListQuery, correctPage = true): Promise<ListLoadResult> => {
      const requestID = ++listRequestID.current;
      setLoading(true);
      try {
        const list = await requestOrganizationList(query);
        if (requestID !== listRequestID.current) {
          return "stale";
        }
        if (!list) {
          setListError(LIST_REFRESH_ERROR);
          return "failed";
        }
        const nextOffset = pageOffsetForTotal(query.offset, list.total);
        if (shouldLoadPage(correctPage, query.offset, nextOffset, list.total)) {
          return fetchOrganizations({ ...query, offset: nextOffset }, false);
        }
        if (requestID !== listRequestID.current) {
          return "stale";
        }
        setListError(null);
        applyList(list, nextOffset);
        return "applied";
      } catch {
        if (requestID !== listRequestID.current) {
          return "stale";
        }
        setListError(LIST_REFRESH_ERROR);
        return "failed";
      } finally {
        if (requestID === listRequestID.current) {
          setLoading(false);
        }
      }
    },
    [],
  );

  useEffect(() => {
    const timeout = setTimeout(() => {
      const query = { search, planKey, offset: 0, sortBy, sortDirection };
      appliedFilterKeyRef.current = listFilterKey(query);
      setOffset(0);
      void fetchOrganizations(query);
    }, 200);
    return () => clearTimeout(timeout);
  }, [search, planKey, sortBy, sortDirection, fetchOrganizations]);

  const pinActions = useOrganizationPinActions(listQueryRef, fetchOrganizations);

  const handleSort = (field: OrganizationSortField) => {
    if (field === sortBy) {
      setSortDirection((prev) => (prev === "asc" ? "desc" : "asc"));
      return;
    }
    setSortBy(field);
    setSortDirection(field === "name" || field === "plan_name" ? "asc" : "desc");
  };

  const changePage = (pageOffset: number) => {
    setOffset(pageOffset);
    void fetchOrganizations({ search, planKey, offset: pageOffset, sortBy, sortDirection });
  };

  return {
    organizations,
    pinnedOrganizations,
    total,
    matchTotal,
    offset,
    search,
    setSearch,
    planKey,
    setPlanKey,
    loading,
    sortBy,
    sortDirection,
    pinError: pinActions.pinError,
    listError,
    pendingPinIDs: pinActions.pendingPinIDs,
    handleSort,
    togglePin: pinActions.togglePin,
    retryList: pinActions.retryList,
    changePage,
  };
}

function shouldLoadPage(correctPage: boolean, pageOffset: number, nextOffset: number, total: number): boolean {
  return correctPage && nextOffset !== pageOffset && total > 0;
}

function useOrganizationPinActions(
  listQueryRef: { current: OrganizationListQuery },
  fetchOrganizations: (query: OrganizationListQuery) => Promise<ListLoadResult>,
) {
  const [pinError, setPinError] = useState<string | null>(null);
  const [pendingPinIDs, setPendingPinIDs] = useState<ReadonlySet<string>>(() => new Set());
  const pendingPinIDsRef = useRef(pendingPinIDs);

  const setPendingPin = (organizationID: string, pending: boolean) => {
    const next = new Set(pendingPinIDsRef.current);
    if (pending) {
      next.add(organizationID);
    } else {
      next.delete(organizationID);
    }
    pendingPinIDsRef.current = next;
    setPendingPinIDs(next);
  };

  const togglePin = async (organization: AdminOrganization, pinned: boolean) => {
    if (pendingPinIDsRef.current.has(organization.id)) {
      return;
    }
    setPendingPin(organization.id, true);
    setPinError(null);
    try {
      const response = await fetch(`/admin/api/organizations/${organization.id}/pin`, {
        method: pinned ? "DELETE" : "PUT",
        credentials: "include",
      });
      if (!response.ok) {
        setPinError(pinned ? "Could not unpin this organization." : "Could not pin this organization.");
        return;
      }
      await fetchOrganizations(listQueryRef.current);
    } catch {
      setPinError(pinned ? "Could not unpin this organization." : "Could not pin this organization.");
    } finally {
      setPendingPin(organization.id, false);
    }
  };

  const retryList = () => {
    void fetchOrganizations(listQueryRef.current);
  };

  return { pinError, pendingPinIDs, togglePin, retryList };
}
