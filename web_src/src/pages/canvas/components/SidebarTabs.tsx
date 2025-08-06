interface Tab {
  key: string;
  label: string;
}

interface SidebarTabsProps {
  tabs: Tab[];
  activeTab: string;
  onTabChange: (tabKey: string) => void;
}

export const SidebarTabs = ({ tabs, activeTab, onTabChange }: SidebarTabsProps) => {
  return (
    <div className="flex border-b border-gray-200 bg-white w-full">
      {tabs.map(tab => (
        <button
          key={tab.key}
          className={`cursor-pointer px-5 py-1 text-sm font-semibold transition-colors text-color-[var(--dark-indigo)] ${activeTab === tab.key
            ? 'text-[var(--indigo)] border-b-2 border-color-[var(--indigo)] font-bold'
            : 'text-gray-500 hover:text-gray-700 hover:bg-gray-50'
            }`}
          onClick={() => onTabChange(tab.key)}
        >
          {tab.label}
        </button>
      ))}
    </div>
  );
};