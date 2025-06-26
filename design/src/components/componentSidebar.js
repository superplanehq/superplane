import React, { useState } from "react";
import SidebarNode from "./sidebarNode";
import NodeGroup from "./NodeGroup";
import { categoriesList } from "./categoriesList";

const ComponentSidebar = ({ onAddNode, onDragStart }) => {
  const [isOpen, setIsOpen] = useState(true);

  const handleToggle = () => setIsOpen(!isOpen);

  return (
    <div className={`relative h-full ${isOpen ? 'w-[300px]' : 'w-0'} bg-transparent transition-[width] duration-300 ease-linear`}>
      <div className="absolute z-50 top-[1px] bottom-[0px]">
        <button
          className={`open-sidebar top-[0px] left-[4px] absolute z-40 !m-2 flex btn-secondary items-center gap-2 rounded-md border border-secondary-hover bg-white fill-foreground stroke-foreground py-2 px-4 text-primary shadow transition-all duration-300 ${isOpen ? 'pointer-events-none opacity-0 -translate-x-full' : 'pointer-events-all opacity-100 translate-x-0'}`}
          onClick={handleToggle}
        >
          <span className="f4 lh-0 b mr-1">Components</span>
          <i className="material-symbols-outlined f2 gray -scale-x-100">menu_open</i>
        </button>

        <div className={`flex h-full items-start z-50 w-[300px] transition-all duration-300 ${isOpen ? 'opacity-1 -translate-x-0' : 'opacity-0 -translate-x-full'}`}>
          <div className="flex h-full w-full flex-col sidebar-body bg-white relative self-stretch overflow-hidden">
            <div className="flex h-full w-full flex-col items-start py-0 relative flex-[0_0_auto] border-r [border-right-style:solid] border-[#0000001a]">
              <div className="flex flex-col w-100 items-start pt-0 pb-0 px-0 relative flex-[0_0_auto] border-0 border-none">
                <div className="flex items-center justify-between pt-4 pb-4 px-4 w-100 relative">
                  <h2 className="f4 lh-0 b mr-1 mb-0">Components</h2>
                  <button href="#" id="sidebar-toggle" className="!relative !w-6 !h-6" onClick={handleToggle}>
                    <i className="material-symbols-outlined f2 gray">menu_open</i>
                  </button>
                </div>
                <div className="my-2 px-4 w-100">
                  <input type="text" className="form-control w-100 mb-4" placeholder="Search…" />
                </div>
              </div>
              <div className="flex min-h-0 flex-1 flex-col overflow-auto w-full">
                <div className="relative flex w-full min-w-0 flex-col px-2">
                  <div className="flex items-center text-sm uppercase b gray px-1 pt2 mb-2">
                    <i className="material-symbols-outlined f2 gray mr-1 hidden">merge</i>
                    Stages
                  </div>
                  <div className="pl1 pr2">
                    <SidebarNode
                      key={"deployment-gates"}
                      icon="rocket_launch"
                      title={"Pre-deployment"}
                      onAddNode={onAddNode}
                      onDragStart={onDragStart}
                    />
                    <SidebarNode
                      key={"deployment-gates"}
                      icon="rocket_launch"
                      title={"Staging"}
                      onAddNode={onAddNode}
                      onDragStart={onDragStart}
                    />
                    <SidebarNode
                      key={"deployment-gates"}
                      icon="rocket_launch"
                      title={"Production"}
                      onAddNode={onAddNode}
                      onDragStart={onDragStart}
                    />
                    <SidebarNode
                      key={"deployment-gates"}
                      icon="rocket_launch"
                      title={"Something else"}
                      onAddNode={onAddNode}
                      onDragStart={onDragStart}
                    />
                  </div>
                  <div className="flex items-center text-sm uppercase b gray px-1 pt2 mb-2">
                    <i className="material-symbols-outlined f2 gray mr-1 hidden">merge</i>
                    Deployment gates
                  </div>
                  <div className="pl1 pr2">
                    <SidebarNode
                      key={"deployment-gates"}
                      icon="local_police"
                      title={"Manual approval"}
                      onAddNode={onAddNode}
                      onDragStart={onDragStart}
                    />
                    <SidebarNode
                      key={"deployment-gates"}
                      icon="local_police"
                      title={"Schedule restriction"}
                      onAddNode={onAddNode}
                      onDragStart={onDragStart}
                    />
                    <SidebarNode
                      key={"deployment-gates"}
                      icon="local_police"
                      title={"Deployment window"}
                      onAddNode={onAddNode}
                      onDragStart={onDragStart}
                    />
                    <SidebarNode
                      key={"deployment-gates"}
                      icon="local_police"
                      title={"Incident pauses"}
                      onAddNode={onAddNode}
                      onDragStart={onDragStart}
                    />
                  </div>
                {categoriesList.map((category) => (
                  <div key={category.category_name}>
                    <div className="flex items-center text-sm uppercase b gray px-1 pt2 mb-2">
                      <i className="material-symbols-outlined f2 gray mr-1 hidden">merge</i>
                      {category.category_name}
                    </div>
                    {category.tools.map((tool) => (
                      <NodeGroup
                        key={tool.name}
                        title={tool.name}
                        logo={tool.logo}
                        collapsed={true}
                      >
                        <SidebarNode
                          key={tool.name}
                          logo={tool.logo}
                          title={"Listen to " + tool.name}
                        />
                        {tool.events.length > 0 && (
                        <div className="relative w-fit whitespace-nowrap text-sm b gray px-2 mb-2">
                        Bundle
                        </div>
                        )}
                        {tool.events.map((item, key) => (
                          <SidebarNode
                            key={key}
                            logo={tool.logo}
                            title={item}
                            icon="bolt"
                            icon_filled={true}
                            onAddNode={onAddNode}
                            onDragStart={onDragStart}
                          />
                        ))}
                        {tool.actions.map((item, key) => (
                          <SidebarNode
                            key={key}
                            logo={tool.logo}
                            title={item}
                            onAddNode={onAddNode}
                            onDragStart={onDragStart}
                          />
                        ))}
                      </NodeGroup>
                    ))}
                  </div>
                ))}
                 
                   <div className="flex items-center text-sm uppercase b gray px-1 pt2 mb-2">
                      <i className="material-symbols-outlined f2 gray mr-1 hidden">merge</i>
                      Custom components
                    </div>
                    <div className="pl1 pr2">
                    <SidebarNode
                      key={"deployment-gates"}
                      icon="data_object"
                      title={"My custom component 1"}
                      onAddNode={onAddNode}
                      onDragStart={onDragStart}
                    />
                    <SidebarNode
                      key={"deployment-gates"}
                      icon="data_object"
                      title={"My custom component 2"}
                      onAddNode={onAddNode}
                      onDragStart={onDragStart}
                    />
                    <SidebarNode
                      key={"deployment-gates"}
                      icon="data_object"
                      title={"My custom component 3"}
                      onAddNode={onAddNode}
                      onDragStart={onDragStart}
                    />
                   </div>
                 
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ComponentSidebar;