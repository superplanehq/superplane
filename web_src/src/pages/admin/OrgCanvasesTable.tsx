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
    <div>
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <Palette size={16} className="text-gray-600" />
          <Heading level={2} className="text-gray-800 text-base">
            Canvases ({total})
          </Heading>
        </div>
        <div className="relative w-56">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            placeholder="Search canvases..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-9 pr-3 py-1.5 text-sm border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
      </div>

      {canvases.length === 0 ? (
        <Text className="text-gray-500 text-sm">
          {search ? "No canvases match your search." : "No canvases in this organization."}
        </Text>
      ) : (
        <>
          <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-100">
                  <th className="text-left px-4 py-2.5 text-gray-500 font-medium">Name</th>
                  <th className="text-left px-4 py-2.5 text-gray-500 font-medium">Description</th>
                </tr>
              </thead>
              <tbody>
                {canvases.map((canvas) => (
                  <tr key={canvas.id} className="border-b border-slate-50 last:border-0">
                    <td className="px-4 py-2.5 text-gray-800 font-medium">{canvas.name}</td>
                    <td className="px-4 py-2.5 text-gray-500">{canvas.description || "—"}</td>
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
