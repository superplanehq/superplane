import { Text } from "@/components/Text/text";
import { Heading } from "@/components/Heading/heading";
import { Search } from "lucide-react";
import React from "react";

interface AdminSearchHeaderProps {
  title: string;
  subtitle: string;
  search: string;
  onSearchChange: (value: string) => void;
  placeholder: string;
}

const AdminSearchHeader: React.FC<AdminSearchHeaderProps> = ({
  title,
  subtitle,
  search,
  onSearchChange,
  placeholder,
}) => (
  <div className="flex items-center justify-between mb-4">
    <div>
      <Heading className="mb-0.5 text-content-primary">{title}</Heading>
      <Text className="text-sm text-content-secondary">{subtitle}</Text>
    </div>

    <div className="relative w-72">
      <Search size={14} className="absolute top-1/2 left-3 -translate-y-1/2 text-content-muted" />
      <input
        type="text"
        placeholder={placeholder}
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
        className="w-full rounded-md border border-edge-default bg-surface-raised py-1.5 pr-3 pl-9 text-sm text-content-primary placeholder:text-content-muted focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
      />
    </div>
  </div>
);

export default AdminSearchHeader;
