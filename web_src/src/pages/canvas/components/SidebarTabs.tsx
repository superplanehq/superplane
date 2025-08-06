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
    <div className="flex border-b border-gray-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 w-full">
      {tabs.map(tab => (
        <button
          key={tab.key}
          className={`cursor-pointer px-5 py-1 text-sm font-semibold transition-colors ${activeTab === tab.key
            ? 'text-blue-600 dark:text-blue-400 border-b-2 border-blue-600 dark:border-blue-400 font-bold'
            : 'text-gray-500 dark:text-zinc-400 hover:text-gray-700 dark:hover:text-zinc-200 hover:bg-gray-50 dark:hover:bg-zinc-800'
            }`}
          onClick={() => onTabChange(tab.key)}
        >
          {tab.label}
        </button>
      ))}
    </div>
  );
};