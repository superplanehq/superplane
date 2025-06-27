import React, { useState } from 'react';

const NodeGroup = ({ title, icon, logo, children, collapsed = false }) => {
  const [isNodesCollapsed, setIsNodesCollapsed] = useState(collapsed);
  const handleToggle = () => setIsNodesCollapsed(!isNodesCollapsed);

  return (
    <div className={`px-1 ${isNodesCollapsed ? 'collapsed' : 'expanded'}`}>
      <div 
        className={`node-group cursor-pointer py-1 pl1 pr1 rounded-md hover:bg-gray-100 category-trigger flex items-center justify-between relative self-stretch w-full flex-[0_0_auto]`}
        onClick={handleToggle}  
      >
        <div className="flex items-center pl-0 pr-4 relative flex-[0_0_auto]">
          {icon && <i className="material-symbols-outlined f2 gray mr-2">{icon}</i>}
          {logo && <img src={logo} className="sidebar-node-icon mr-2" width={16} />}
          <h2 className={`${!isNodesCollapsed ? 'b': ''} w-fit whitespace-nowrap text-md`}>
            {title}
          </h2>
        </div>
        <div className="ml-auto flex items-center">
          <i 
            className={`material-symbols-outlined f2 gray transition-transform duration-200`}
          >
            {!isNodesCollapsed ? 'keyboard_arrow_down': 'keyboard_arrow_right'}
          </i>
        </div>
      </div>
     
    
        <div className={`max-w-100 categories grid grid-rows-0 transition-[grid-template-rows] duration-250 mt-2`}>
        {!isNodesCollapsed && ( 
          children
        )}
        </div>
     
    </div>
  );
};

export default NodeGroup;
