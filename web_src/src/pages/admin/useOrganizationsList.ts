import { useCallback, useEffect, useRef, useState } from "react";

import type { SortDirection } from "./SortableHeader";

export interface AdminOrganization {
  id: string;
  name: string;
  canvas_count: number;
  task_count: number;
  done_task_count: number;
  member_count: number;
  created_at?: string;
}

export type OrganizationSortField =
  | "canvas_count"
  | "created_at"
  | "done_task_count"
  | "member_count"
  | "name"
  | "task_count";

export const ORGANIZATION_PAGE_SIZE = 50;

interface OrganizationListQuery {
  search: string;
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

function listFilterKey(search: string, sortBy: OrganizationSortField, sortDirection: SortDirection): string {
  return `${search}\0${sortBy}\0${sortDirection}`;
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

async function requestOrganizationList(
  searchTerm: string,
  pageOffset: number,
  sort: OrganizationSortField,
  direction: SortDirection,
): Promise<OrganizationListResponse | null> {
  const params = new URLSearchParams({ limit: String(ORGANIZATION_PAGE_SIZE), offset: String(pageOffset) });
  if (searchTerm) params.set("search", searchTerm);
  params.set("sort_by", sort);
  params.set("sort_direction", direction);
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
  const [loading, setLoading] = useState(true);
  const [sortBy, setSortBy] = useState<OrganizationSortField>("created_at");
  const [sortDirection, setSortDirection] = useState<SortDirection>("desc");
  const [pinError, setPinError] = useState<string | null>(null);
  const [pendingPinID, setPendingPinID] = useState<string | null>(null);
  const listRequestID = useRef(0);
  const appliedFilterKeyRef = useRef(listFilterKey("", "created_at", "desc"));
  const listQueryRef = useRef<OrganizationListQuery>({ search, offset, sortBy, sortDirection });
  const filterKey = listFilterKey(search, sortBy, sortDirection);
  listQueryRef.current =
    filterKey === appliedFilterKeyRef.current
      ? { search, offset, sortBy, sortDirection }
      : { search, offset: 0, sortBy, sortDirection };

  const applyList = (list: OrganizationListResponse, pageOffset: number) => {
    setOffset(pageOffset);
    setOrganizations(list.items);
    setPinnedOrganizations(list.pinned);
    setTotal(list.total);
    setMatchTotal(list.matchTotal);
  };

  const fetchOrganizations = useCallback(
    async (
      searchTerm: string,
      pageOffset: number,
      sort: OrganizationSortField,
      direction: SortDirection,
      correctPage = true,
    ) => {
      const requestID = ++listRequestID.current;
      setLoading(true);
      try {
        const list = await requestOrganizationList(searchTerm, pageOffset, sort, direction);
        if (requestID !== listRequestID.current || !list) {
          return;
        }
        const nextOffset = pageOffsetForTotal(pageOffset, list.total);
        if (shouldLoadPage(correctPage, pageOffset, nextOffset, list.total)) {
          await fetchOrganizations(searchTerm, nextOffset, sort, direction, false);
          return;
        }
        if (requestID !== listRequestID.current) {
          return;
        }
        applyList(list, nextOffset);
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
      appliedFilterKeyRef.current = listFilterKey(search, sortBy, sortDirection);
      setOffset(0);
      void fetchOrganizations(search, 0, sortBy, sortDirection);
    }, 200);
    return () => clearTimeout(timeout);
  }, [search, sortBy, sortDirection, fetchOrganizations]);

  const handleSort = (field: OrganizationSortField) => {
    if (field === sortBy) {
      setSortDirection((prev) => (prev === "asc" ? "desc" : "asc"));
      return;
    }
    setSortBy(field);
    setSortDirection(field === "name" ? "asc" : "desc");
  };

  const togglePin = async (organization: AdminOrganization, pinned: boolean) => {
    setPinError(null);
    setPendingPinID(organization.id);
    try {
      const response = await fetch(`/admin/api/organizations/${organization.id}/pin`, {
        method: pinned ? "DELETE" : "PUT",
        credentials: "include",
      });
      if (!response.ok) {
        setPinError(pinned ? "Could not unpin this organization." : "Could not pin this organization.");
        return;
      }
      const query = listQueryRef.current;
      await fetchOrganizations(query.search, query.offset, query.sortBy, query.sortDirection);
    } catch {
      setPinError(pinned ? "Could not unpin this organization." : "Could not pin this organization.");
    } finally {
      setPendingPinID(null);
    }
  };

  const changePage = (pageOffset: number) => {
    setOffset(pageOffset);
    void fetchOrganizations(search, pageOffset, sortBy, sortDirection);
  };

  return {
    organizations,
    pinnedOrganizations,
    total,
    matchTotal,
    offset,
    search,
    setSearch,
    loading,
    sortBy,
    sortDirection,
    pinError,
    pendingPinID,
    handleSort,
    togglePin,
    changePage,
  };
}

function shouldLoadPage(correctPage: boolean, pageOffset: number, nextOffset: number, total: number): boolean {
  return correctPage && nextOffset !== pageOffset && total > 0;
}
