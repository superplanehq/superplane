import { Text } from "@/components/Text/text";
import { Heading } from "@/components/Heading/heading";
import { Palette, Search } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import AdminPagination from "./AdminPagination";

interface Canvas {
  id: string;
  name: string;
  description: string;
}

interface PaginatedResponse<T> {
  items: T[];
  total: number;
}

const PAGE_SIZE = 50;

export function OrgCanvasesTable({ orgId }: { orgId: string }) {
  const [canvases, setCanvases] = useState<Canvas[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");

  const fetchCanvases = useCallback(
    async (s: string, o: number) => {
      const params = new URLSearchParams({ limit: String(PAGE_SIZE), offset: String(o) });
      if (s) params.set("search", s);
      const res = await fetch(`/admin/api/organizations/${orgId}/canvases?${params}`, { credentials: "include" });
      if (res.ok) {
        const data: PaginatedResponse<Canvas> = await res.json();
        setCanvases(data.items);
        setTotal(data.total);
      }
    },
    [orgId],
  );

  useEffect(() => {
    const t = setTimeout(() => {
      setOffset(0);
      fetchCanvases(search, 0);
    }, 200);
    return () => clearTimeout(t);
  }, [search, fetchCanvases]);

  return (
    <div className="mb-8">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <Palette size={16} className="text-content-secondary" />
          <Heading level={2} className="text-base text-content-primary">
            Canvases ({total})
          </Heading>
        </div>
        <div className="relative w-56">
          <Search size={14} className="absolute top-1/2 left-3 -translate-y-1/2 text-content-muted" />
          <input
            type="text"
            placeholder="Search canvases..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-md border border-edge-default bg-surface-raised py-1.5 pr-3 pl-9 text-sm text-content-primary placeholder:text-content-muted focus:ring-1 focus:ring-blue-500 focus:outline-none"
          />
        </div>
      </div>

      {canvases.length === 0 ? (
        <Text className="text-sm text-content-secondary">
          {search ? "No canvases match your search." : "No canvases in this organization."}
        </Text>
      ) : (
        <>
          <div className="overflow-hidden rounded-md bg-surface-raised shadow-sm outline outline-edge-subtle">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-edge-default">
                  <th className="px-4 py-2.5 text-left font-medium text-content-secondary">Name</th>
                  <th className="px-4 py-2.5 text-left font-medium text-content-secondary">Description</th>
                </tr>
              </thead>
              <tbody>
                {canvases.map((canvas) => (
                  <tr key={canvas.id} className="border-b border-edge-subtle last:border-0">
                    <td className="px-4 py-2.5 font-medium text-content-primary">{canvas.name}</td>
                    <td className="px-4 py-2.5 text-content-secondary">{canvas.description || "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <AdminPagination
            offset={offset}
            total={total}
            pageSize={PAGE_SIZE}
            onPageChange={(o) => {
              setOffset(o);
              fetchCanvases(search, o);
            }}
          />
        </>
      )}
    </div>
  );
}
