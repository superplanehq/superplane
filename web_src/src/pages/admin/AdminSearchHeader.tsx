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
      <Heading className="text-gray-800 mb-0.5">{title}</Heading>
      <Text className="text-gray-500 text-sm">{subtitle}</Text>
    </div>

    <div className="relative w-72">
      <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
      <input
        type="text"
        placeholder={placeholder}
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
        className="w-full pl-9 pr-3 py-1.5 text-sm border border-slate-200 rounded-md bg-white focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
      />
    </div>
  </div>
);

export default AdminSearchHeader;
