import React from 'react';

const SidebarNode = ({
  title,
  logo,
  icon,
  icon_filled = false,
  onAddNode,
  onDragStart
}) => (
  <div className="cursor-grab rounded-md flex items-center node pl2 pr2 py-3 relative bg-gray-100 hover:bg-gray-200 mb-2" onDragStart={(e) => onDragStart(e, 'githubIntegration')} draggable="true">
    
      {logo && (
        <img src={logo} className="sidebar-node-icon mr-2" width="20px"/>
      )}
      
    
    <div className="min-w-0 flex relative self-stretch flex-1 truncate">
        <div className='flex items-center min-w-0 w-full pr2'>
          {icon && (
            <i className={`material-symbols-outlined ${icon_filled ? 'fill' : 'outlined'} text-xl dark-gray leading-none`}>{icon}</i>
          )}
          <h3 className="relative first-letter:uppercase black-90 f3 mb-0 tracking-[0] leading-none mb-0 f6 gray overflow-hidden ml2 text-ellipsis">
            {title}
          </h3>
        </div>
         
    </div>
    <div className="flex items-center">
        <button className="add-node hidden material-symbols-outlined f3 gray mr2 hover:bg-gray-100 br2" onClick={() => onAddNode('githubIntegration', { x: 0, y: 0 })}>add</button>
        <button className="drag-node cursor-grab material-symbols-outlined f3 gray">drag_indicator</button>
    </div>
</div>
  
);

export default SidebarNode;
