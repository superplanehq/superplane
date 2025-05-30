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
    <div className="flex border-b border-gray-200 bg-white">
      {tabs.map(tab => (
        <button
          key={tab.key}
          className={`flex-1 px-4 py-3 text-sm font-medium transition-colors ${
            activeTab === tab.key
              ? 'text-indigo-600 border-b-2 border-indigo-600 bg-indigo-50'
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