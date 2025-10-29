import Tippy from '@tippyjs/react/headless';
import { ReactElement, useState } from 'react';
import 'tippy.js/dist/tippy.css';


interface ComponentOption {
  name: string;
  blockColor: string;
  onClick: () => void;
  subOptions?: ComponentOption[];
}

interface ComponentSubCategory {
  name: string;
  options: ComponentOption[];
}

interface ComponentTab {
  name: string;
  subCategories: ComponentSubCategory[];
}

interface ComponentLibraryTooltipProps {
  children: React.ReactNode;
  tabs: ComponentTab[];
}

export function ComponentLibraryTooltip({ children, tabs }: ComponentLibraryTooltipProps) {
  const [activeTab, setActiveTab] = useState(0);
  const [selectedOption, setSelectedOption] = useState<ComponentOption | null>(null);

  const handleOptionClick = (option: ComponentOption) => {
    if (option.subOptions && option.subOptions.length > 0) {
      setSelectedOption(option);
    } else {
      option.onClick();
    }
  };

  const handleBackClick = () => {
    setSelectedOption(null);
  };

  const renderSubOptions = () => {
    if (!selectedOption || !selectedOption.subOptions) return null;

    return (
      <div className="p-4">
        <div className="flex items-center mb-3">
          <button
            onClick={handleBackClick}
            className="flex items-center text-gray-600 hover:text-gray-800 text-sm"
          >
            ← Back
          </button>
        </div>
        <h3 className="font-medium text-gray-800 mb-2">{selectedOption.name}</h3>
        <div className="space-y-1">
          {selectedOption.subOptions.map((subOption, index) => (
            <button
              key={index}
              onClick={() => subOption.onClick()}
              className="block w-full text-left px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 rounded"
            >
              {subOption.name}
            </button>
          ))}
        </div>
      </div>
    );
  };

  const renderTabContent = () => {
    if (selectedOption) {
      return renderSubOptions();
    }

    const currentTab = tabs[activeTab];
    if (!currentTab) return null;

    if (currentTab.subCategories.length === 0) {
      return (
        <div className="p-4 w-full">
          <p className="text-gray-600">No items found for this tab.</p>
        </div>
      );
    }

    return (
      <div className="p-4">
        {currentTab.subCategories.map((subCategory, subIndex) => (
          <div key={subIndex} className="mb-4">
            <h3 className="font-medium text-gray-800 mb-2">{subCategory.name}</h3>
            <div className="space-y-1">
              {subCategory.options.map((option, optionIndex) => (
                <button
                  key={optionIndex}
                  onClick={() => handleOptionClick(option)}
                  className="flex items-center justify-between w-full text-left px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 rounded"
                >
                  <div className="flex items-center">
                    <div
                      className="w-3 h-3 rounded mr-2"
                      style={{ backgroundColor: option.blockColor }}
                    />
                    {option.name}
                  </div>
                  {option.subOptions && option.subOptions.length > 0 && (
                    <span className="text-gray-400">→</span>
                  )}
                </button>
              ))}
            </div>
          </div>
        ))}
      </div>
    );
  };

  return (
    <Tippy
      render={() => (
        <div className="bg-white border-2 border-gray-200 rounded-md w-[200px] shadow-lg">
          <div className="flex border-b border-gray-200">
            {tabs.map((tab, index) => (
              <button
                key={index}
                onClick={() => {
                  setActiveTab(index);
                  setSelectedOption(null);
                }}
                className={`px-4 py-2 text-sm font-medium ${activeTab === index
                  ? 'text-blue-600 border-b-2 border-blue-600'
                  : 'text-gray-600 hover:text-gray-800'
                  }`}
              >
                {tab.name}
              </button>
            ))}
          </div>
          {renderTabContent()}
        </div>
      )}
      placement="bottom"
      interactive={true}
      delay={200}
    >
      {children as ReactElement}
    </Tippy>
  );
}