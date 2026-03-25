import { Text } from "@/components/Text/text";
import React from "react";

interface AdminPaginationProps {
  offset: number;
  total: number;
  pageSize: number;
  onPageChange: (newOffset: number) => void;
}

const AdminPagination: React.FC<AdminPaginationProps> = ({ offset, total, pageSize, onPageChange }) => {
  const totalPages = Math.ceil(total / pageSize);
  const currentPage = Math.floor(offset / pageSize) + 1;

  if (totalPages <= 1) return null;

  return (
    <div className="flex items-center justify-between mt-4 text-sm text-gray-500">
      <Text>
        Showing {offset + 1}–{Math.min(offset + pageSize, total)} of {total}
      </Text>
      <div className="flex gap-2">
        <button
          onClick={() => onPageChange(offset - pageSize)}
          disabled={offset === 0}
          className="px-3 py-1 rounded border border-slate-200 bg-white hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed text-xs"
        >
          Previous
        </button>
        <button
          onClick={() => onPageChange(offset + pageSize)}
          disabled={currentPage >= totalPages}
          className="px-3 py-1 rounded border border-slate-200 bg-white hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed text-xs"
        >
          Next
        </button>
      </div>
    </div>
  );
};

export default AdminPagination;
