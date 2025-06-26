import React, { useState, useCallback, useRef, useEffect } from 'react';
import * as htmlToImage from 'html-to-image';
import rocket from './images/logos/rocket.svg';
import semaphore from './images/semaphore-logo-sign-black.svg';
import kubernetes from './images/logos/kubernetes.svg';
import RunItem from './components/RunItem';
import MessageItem from './components/MessageItem';
import github from './images/icn-github.svg';
import s3 from './images/logos/aws-cloudformation.svg';
import ReactFlow, {
  Controls,
  Background,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  addEdge,
  MarkerType,
  ConnectionLineType,
  Handle,
  Position,
} from 'reactflow';
import 'reactflow/dist/style.css';
import icnCommit from './images/icn-commit.svg';
import profileImg from './images/profile.jpg';
import faviconPinned from './images/favicon-pinned.svg';
import Tippy from '@tippyjs/react';
import 'tippy.js/dist/tippy.css';
import CustomBarHandle from './CustomBarHandle';
import ComponentSidebar from './components/componentSidebar';
import Navigation from './components/navigation';

const DeploymentCardStage = React.memo(({ data, selected, onIconAction, id, onDelete }) => {
  const [showOverlay, setShowOverlay] = React.useState(false);
  
  const handleAction = React.useCallback((action) => {
    if (action === 'code') setShowOverlay(true);
    if (onIconAction) onIconAction(action);
  }, [onIconAction]);
  
  const handleDelete = React.useCallback(() => {
    if (onDelete) onDelete(id);
  }, [onDelete, id]);
  
  // Use a fixed width to prevent resize observer loops and add white shadow
  const nodeStyle = React.useMemo(() => ({
    width: data.style?.width || 320,
    boxShadow: '0 4px 12px rgba(128,128,128,0.20)', // White shadow
  }), [data.style?.width]);

  return (
    <div className={`bg-white br2 ba bw1  ${selected ? 'b--indigo' : 'b--lighter-gray'} relative`} style={nodeStyle}>
      {/* Icon block above node when selected */}
      {selected && (
        <div className="absolute -top-12 left-1/2 -translate-x-1/2 flex gap-2 bg-white shadow-gray-lg br2 px-2 py-1 border z-10">
        <Tippy content="Start a run for this stage" placement="top">
        <button className="hover:bg-gray-100 text-black-60 px-2 py-1 br2 leading-none" title="Start Run" onClick={() => handleAction('run')}>
        <span className="material-icons" style={{fontSize:20}}>play_arrow</span>
        </button>
        </Tippy>
        <Tippy content="View code for this stage" placement="top">
        <button className="hover:bg-gray-100 text-black-60 px-2 py-1 br2 leading-none" title="View Code" onClick={() => handleAction('code')}>
        <span className="material-icons" style={{fontSize:20}}>code</span>
        </button>
        </Tippy>
        <Tippy content="Edit triggers for this stage" placement="top">
        <button className="hover:bg-gray-100 text-black-60 px-2 py-1 br2 leading-none" title="Edit Triggers" onClick={() => handleAction('edit')}>
        <span className="material-icons" style={{fontSize:20}}>bolt</span>
        </button>
        </Tippy>
        <Tippy content="Delete this stage" placement="top">
        <button className="hover:bg-red-100 hover:text-red-600 text-black-60 px-2 py-1 br2 leading-none" title="Delete Stage" onClick={handleDelete}>
        <span className="material-icons" style={{fontSize:20}}>delete</span>
        </button>
        </Tippy>
        <Tippy content="More actions" placement="top">
        <button className="hover:bg-gray-100 text-black-60 px-2 py-1 br2 leading-none" title="More Actions" onClick={() => handleAction('run')}>
        <span className="material-icons" style={{fontSize:20}}>more_vert</span>
        </button>
        </Tippy>
  
        </div>
      )}
      {/* Modal overlay for View Code */}
      <OverlayModal open={showOverlay} onClose={() => setShowOverlay(false)}>
        <h2 style={{ fontSize: 22, fontWeight: 700, marginBottom: 16 }}>Stage Code</h2>
        <div style={{ color: '#444', fontSize: 16, lineHeight: 1.7 }}>
        Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse et urna fringilla, tincidunt nulla nec, dictum erat. Etiam euismod, justo id facilisis dictum, urna massa dictum erat, eget dictum urna massa id justo. Praesent nec facilisis urna. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas.
        </div>
      </OverlayModal>
      <div className="pa3 flex justify-between">
          <div className="flex items-center">
          <span className="material-symbols-outlined mr1">rocket_launch</span>
              <p className="mb0 b ml1">{data.label}</p>
          </div>
          <div className='flex items-center'>
            <Tippy content="Healthy. Last check run 2 hours ago" placement="top">
            <span className="br-pill bg-green w-[12px] h-[12px] ba bw1  b--lightest-green"></span>
            </Tippy>
            </div>
          {(data.hasHealthCheck && data.healthCheckStatus === 'healthy') ? (
            <div className='flex items-center'>
            <Tippy content="Healthy. Last check run 2 hours ago" placement="top">
            <span className="br-pill bg-green w-[12px] h-[12px] ba bw1  b--lightest-green"></span>
            </Tippy>
            </div>
          ) : null}
          {(data.hasHealthCheck && data.healthCheckStatus === 'blocked') ? (
            <div className='flex items-center'>
            <Tippy content="Blocked. Last check run 2 hours ago" placement="top">
            <span className="br-pill bg-red w-[12px] h-[12px] ba bw1  b--lightest-red"></span>
            </Tippy>
            </div>
          ) : null}
          {(data.hasHealthCheck && data.healthCheckStatus === 'warning') ? (
            <div className='flex items-center'>
            <Tippy content="Warning. Last check run 2 hours ago" placement="top">
            <span className="br-pill bg-yellow w-[12px] h-[12px] ba bw1  b--lightest-yellow"></span>
            </Tippy>
            </div>
          ) : null}
      </div>
      
      <div className={`pa3 ${data.status === 'Passed' ? 'bg-washed-green b--green' : data.status === 'Failed' ? 'bg-washed-red b--red' : data.status === 'Running' ? 'bg-washed-blue b--blue' : data.status === 'Queued' ? 'bg-washed-yellow b--yellow' : 'bg-washed-green b--green'} w-full bt min-w-0 text-ellipsis overflow-hidden`}>
      <div className="flex items-center w-full justify-between">
        <div className="ttu f7 mt0 mb2">Last run</div>
        <div className="f6 black-60 text-xs">{data.timestamp}</div>
      </div>
  
        <div className="">
            <div className='flex items-center mb1'>
                  {(() => {
                      switch (data.status.toLowerCase()) {
                        case 'passed':
                          return <span className="material-symbols-outlined fill green f1 mr1">check_circle</span>
                        case 'failed':
                          return <span className="material-symbols-outlined fill red f1 mr1">cancel</span>
                        case 'queued':
                          return <span className="material-symbols-outlined fill orange f1 mr1">queue</span>
                        case 'running':
                          return <span className="br-pill bg-blue w-[22px] h-[22px] b--lightest-blue text-center mr2"><span className="white f4 mr1 job-log-working"></span></span>
                        default:
                          return null
                      }
                  })()}
                  <img alt="Favicon" className="h1 w1 mr2" src={semaphore}/>
                <a href="#" className="min-w-0 fw6 font-normal flex items-center underline-hover truncate">
                 BUG-213 When clicking on the...
                </a>
            </div>
            
            <div className='flex items-center'>
              <div className='hidden'>
                    {(() => {
                      switch (data.status.toLowerCase()) {
                        case 'passed':
                          return <span className="material-symbols-outlined fill green f1 mr1">check_circle</span>
                        case 'failed':
                          return <span className="material-symbols-outlined fill red f1 mr1">cancel</span>
                        case 'queued':
                          return <span className="material-symbols-outlined fill orange f1 mr1">queue</span>
                        case 'running':
                          return <span className="blue f1 mr1 job-log-working"></span>
                        default:
                          return null
                      }
                    })()}
                

                
            </div>
                <div className="flex items-center mt1 hidden">
                    <span className="material-symbols-outlined f6">input</span>
                    <span className="ml1 text-xs">Inputs</span>
                </div>
                <div className="flex flex-wrap gap-1 mt2">
                <span className="bg-black-05 text-gray-700 text-xs px-2 pt-0.5 pb-0.5 rounded-full mr2">
                code: {data.labels && data.labels[0] ? data.labels[0] : '—'}
                </span>
                <span className="bg-black-05 text-black-70 text-xs px-2 pt-0.5 pb-0.5 rounded-full mr2 b ba b--black-10">
                image: {data.labels && data.labels[1] ? data.labels[1] : '—'}
                </span>
                <span className="bg-black-05 text-gray-700 text-xs px-2 pt-0.5 pb-0.5 rounded-full mr2">
                terraform: {data.labels && data.labels[2] ? data.labels[2] : '—'}
                </span>
                <span className="bg-black-05 text-gray-700 text-xs px-2 pt-0.5 pb-0.5 rounded-full mr2">
                type: {data.labels && data.labels[3] ? data.labels[3] : '—'}
                </span>
                </div>
                <div className="text-xs mt1 hidden">
                    <p>code: 1042a82</p>
                    <p className='dark-green b'>image: v.4.1.3</p>
                    <p>terraform: v.2.9.2</p>
                    <p>type: community</p>
                </div>

             
            </div>
        </div>
      </div>
      <div className="pa3 pt2 pb0 w-full">
        <div className="ttu f7 mb1">QUEUE</div>
        <div className="w-full">
        <div className="min-w-0 text-ellipsis overflow-hidden">
              
               {data.queue.length > 0 ? (
                  <>
                  {data.queue.map((item, idx) => (
                    <div className='flex items-center w-full  p-2 bg-gray-100 br2 mt1'>
                    <Tippy content="Need manual approval" placement="top">
                      <div className="br-100 black bg-lightest-orange dark-orange w-[24px] h-[24px] mr2 flex items-center justify-center">
                       
                        <i className="material-symbols-outlined f3">how_to_reg</i>
                      </div>
                    </Tippy>
                    <img alt="Favicon" className="h1 w1 mr2" src={semaphore}/>
                    <a href="#" className="min-w-0 fw6 text-sm font-normal flex items-center underline-hover">
                    <div className='truncate'>{item}</div>
                    </a>
                    </div>
                  ))}
                  </>
                ) : (
                  <div className="text-sm text-gray-500 italic">No items in queue</div>
                )}
               
                
              
              <div className='hidden text-align-right'>
                <a className='link-blue text-xs'>View all</a>
              </div>
                
              
          </div>
        </div>
      </div>
         <CustomBarHandle type="target" position={Position.Left} />
         <CustomBarHandle type="source" position={Position.Right} />
     

     
      <div className="p-4 hidden">
        <div className="flex justify-between items-center mb-3">
          <span className={`status-badge ${data.status ? data.status.toLowerCase() : ''}`}>
            {data.status}
          </span>
          <button className="text-gray-500 hover:text-gray-700" onClick={handleDelete}>
            <span className="material-symbols-outlined">delete</span>
          </button>
        </div>

        <h3 className="font-semibold text-gray-900 mb-2">{data.name}</h3>
        <p className="text-gray-600 text-sm">{data.description}</p>
      </div>
    </div>
  );
});
// Custom stage component for the deployment card
const DeploymentCardStage2 = React.memo(({ data, selected, onIconAction, id, onDelete }) => {
  const [showOverlay, setShowOverlay] = React.useState(false);
  const handleAction = React.useCallback((action) => {
    if (action === 'code') setShowOverlay(true);
    if (onIconAction) onIconAction(action);
  }, [onIconAction]);
  
  const handleDelete = React.useCallback(() => {
    if (onDelete) onDelete(id);
  }, [onDelete, id]);
  
  // Use a fixed width to prevent resize observer loops and add white shadow
  const nodeStyle = React.useMemo(() => ({
    width: data.style?.width || 320,
    boxShadow: '0 4px 12px rgba(128,128,128,0.20)', // White shadow
  }), [data.style?.width]);
  
  return (
    <div className={`bg-white roundedg border ${selected ? 'ring-2 ring-blue-500' : 'border-gray-200'} relative`} style={nodeStyle}>
    
    {/* Icon block above node when selected */}
    {selected && (
      <div className="absolute -top-10 left-1/2 -translate-x-1/2 flex gap-2 bg-white shadow-gray-lg br4 px-3 py-2 border z-10">
      <Tippy content="Delete this stage" placement="top">
      <button className="hover:bg-red-100 text-red-600 p-2 br4" title="Delete Stage" onClick={handleDelete}>
      <span className="material-icons" style={{fontSize:20}}>delete</span>
      </button>
      </Tippy>
      <Tippy content="View code for this stage" placement="top">
      <button className="hover:bg-gray-100 p-2 br4" title="View Code" onClick={() => handleAction('code')}>
      <span className="material-icons" style={{fontSize:20}}>code</span>
      </button>
      </Tippy>
      <Tippy content="Edit triggers for this stage" placement="top">
      <button className="hover:bg-gray-100 p-2 br4" title="Edit Triggers" onClick={() => handleAction('edit')}>
      <span className="material-icons" style={{fontSize:20}}>bolt</span>
      </button>
      </Tippy>
      <Tippy content="Start a run for this stage" placement="top">
      <button className="hover:bg-gray-100 p-2 br4" title="Start Run" onClick={() => handleAction('run')}>
      <span className="material-icons" style={{fontSize:20}}>play_arrow</span>
      </button>
      </Tippy>

      </div>
    )}
    {/* Modal overlay for View Code */}
    <OverlayModal open={showOverlay} onClose={() => setShowOverlay(false)}>
    <h2 style={{ fontSize: 22, fontWeight: 700, marginBottom: 16 }}>Stage Code</h2>
    <div style={{ color: '#444', fontSize: 16, lineHeight: 1.7 }}>
    Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse et urna fringilla, tincidunt nulla nec, dictum erat. Etiam euismod, justo id facilisis dictum, urna massa dictum erat, eget dictum urna massa id justo. Praesent nec facilisis urna. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas.
    </div>
    </OverlayModal>
    {/* Custom Node Header */}
    <div className="flex items-center px-3 py-2 border-b bg-gray-50 rounded-tg">
    <span className="flex items-center justify-center w-8 h-8 bg-gray-100 rounded-full mr-2">
    <span className="material-symbols-outlined text-lg">{data.icon}</span>
    </span>
    <span className="font-bold text-gray-900 flex-1"></span>
    
    {/* Example action button (menu) */}
    <button className="ml-2 p-1 rounded hover:bg-gray-200 transition" title="More actions">
    <span className="material-symbols-outlined text-gray-500">more_vert</span>
    </button>
    </div>

    <div className="pa3 flex justify-between bg-white">
        <div className="flex items-center">
            <div className="d-inline-block mr-2 w-[24px]">
              <img src={rocket}/>
            </div>
            <p className="mb0 b ml1">{data.label}</p>
        </div>

        <div className="button-group">
            <i className="material-icons" style={{fontSize:20}}>check_circle</i>
        </div>
    </div>

    <div className="p-4">
    <div className="flex justify-between items-center mb-3">
    <span className={`status-badge ${data.status ? data.status.toLowerCase() : ''}`}>{data.status}</span>
    <span className="text-xs text-gray-500">{data.timestamp}</span>
    </div>
    <div className="flex flex-wrap gap-1 mb-3">
    <span className="pipeline-badge bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2">
    code: {data.labels && data.labels[0] ? data.labels[0] : '—'}
    </span>
    <span className="pipeline-badge bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2">
    image: {data.labels && data.labels[1] ? data.labels[1] : '—'}
    </span>
    <span className="pipeline-badge bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2">
    terraform: {data.labels && data.labels[2] ? data.labels[2] : '—'}
    </span>
    <span className="pipeline-badge bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2">
    type: {data.labels && data.labels[3] ? data.labels[3] : '—'}
    </span>
    </div>
    </div>
    <div className="border-t border-gray-200 p-4">
    <h4 className="text-sm font-medium text-gray-700 mb-2">Run Queue</h4>
    {data.queue.length > 0 ? (
      <>
      {data.queue.map((item, idx) => (
        <div key={idx} className="flex items-center p-2 bg-gray-50 rounded mb-1">
        <div className={`material-symbols-outlined v-mid ${data.queueIconClass || 'purple'} b`}>
        {data.queueIcon || 'flaky'}
        </div>
        <span className="text-sm ml2">{item}</span>
        </div>
      ))}
      </>
    ) : (
      <div className="text-sm text-gray-500 italic">No items in queue</div>
    )}
    </div>
    <CustomBarHandle type="target" position={Position.Left} />
    <CustomBarHandle type="source" position={Position.Right} />
    </div>
  );
});

// Custom integration component for GitHub repository
const GitHubIntegration = ({ data, selected }) => {
  // Select header color and icon based on integrationType
  const isKubernetes = data.integrationType === 'kubernetes';
  const isS3 = data.repoName === 'buckets/my-app-data';
  // Add white shadow style
  const nodeStyle = {
    boxShadow: '0 4px 12px rgba(128,128,128,0.20)' // White shadow
  };
  
  return (
    <div className={`bg-white roundedg border ${selected ? 'ring-2 ring-blue-500' : 'border-gray-200'}`} style={nodeStyle}>
  
    <div className='pa3 flex justify-between bb b--lightest-gray'>
    <div className="flex items-center"><div className="d-inline-block mr-2 w-[24px]">
    {isKubernetes ? (
      <img src={kubernetes}/>
    ) : isS3 ? (
      <img src={s3}/>
    ) : (
      <img src={github}/>
    )}
    </div>
    <p className="mb0 b ml1">Sync Cluster</p>
    </div>
    </div>
    <div className={`flex items-center bg-white black rounded-tg hidden`}>
    <span className={` font-semibold text-base ${isKubernetes ? 'white' : isS3 ? 'black' : 'white'}`}>
    {isKubernetes ? 'prod-cluster' : data.repoName}
    </span>
    </div>
    <div className="repo-info">
    <div className="mb-2">
    <a href={data.repoUrl} className="link dark-indigo underline-hover flex items-center">
    {data.repoName}
    </a>
    </div>
    <div className="flex items-center w-full justify-between">
        <div className="ttu f7">Events</div>
      </div>
    </div>
    <div className="w-full p-3 pt-0">
    
    <div className='flex items-center w-full p-2 bg-gray-100 br2 mb1'>
    <Tippy content="Need manual approval" placement="top">
      <i className="material-symbols-outlined f3 fill br-100 black bg-washed-green black-60 p2 mr2">bolt</i>
    </Tippy>
    <a href="#" className="min-w-0 fw6 text-sm font-normal flex items-center underline-hover">
    <div className='truncate'>https://hooks.semaphoreci.com/semaphore/semaphore/semaphore</div>
    </a>
    </div>
    <div className='flex items-center w-full p-2 bg-gray-100 br2 mb1'>
    <Tippy content="Need manual approval" placement="top">
      <i className="material-symbols-outlined f3 fill br-100 black bg-washed-green black-60 p2 mr2">bolt</i>
    </Tippy>
    <a href="#" className="min-w-0 fw6 text-sm font-normal flex items-center underline-hover">
    <div className='truncate'>https://hooks.semaphoreci.com/semaphore/semaphore/semaphore</div>
    </a>
    </div>
    <div className='flex items-center w-full p-2 bg-gray-100 br2 mb1'>
    <Tippy content="Need manual approval" placement="top">
      <i className="material-symbols-outlined f3 fill br-100 black bg-washed-green black-60 p2 mr2">bolt</i>
    </Tippy>
    <a href="#" className="min-w-0 fw6 text-sm font-normal flex items-center underline-hover">
    <div className='truncate'>https://hooks.semaphoreci.com/semaphore/semaphore/semaphore</div>
    </a>
    </div>
   
    
    </div>
    <Handle 
    type="source" 
    position={Position.Right} 
    style={{ background: isKubernetes ? '#2563eb' : '#000', width: 10, height: 10 }} 
    />
    </div>
  );
};

// Sidebar component to display selected stage details
const Sidebar = React.memo(({ selectedStage, onClose }) => {
  const [activeTab, setActiveTab] = useState('general');
  const [viewMode, setViewMode] = useState('form');
  const [width, setWidth] = useState(600);
  const isDragging = useRef(false);
  const sidebarRef = useRef(null);
  const animationFrameRef = useRef(null);
  const [isMsgDragStart, setIsMsgDragStart] = useState(false);
 
  const handleMsgDragStart = (e) => {
    e.stopPropagation();
    setIsMsgDragStart(!isMsgDragStart);
  };
  // Sidebar tab definitions - memoized to prevent unnecessary re-renders
  const tabs = React.useMemo(() => [
    //{ key: 'runs', label: 'Runs' },
    { key: 'general', label: 'Activity' },
    { key: 'history', label: 'History' },
    //{ key: 'queue', label: 'Queue' },
    { key: 'settings', label: 'Settings' },
  ], []);
  
  // Cleanup function for animation frame and event listeners
  React.useEffect(() => {
    return () => {
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current);
      }
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };
  }, []);
  
  // Handle mouse down on resize handle - memoized to prevent recreation on each render
  const handleMouseDown = React.useCallback((e) => {
    isDragging.current = true;
    document.body.style.cursor = 'ew-resize';
    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  }, []);
  
  // Handle mouse move during resize - memoized with dependencies
  const handleMouseMove = React.useCallback((e) => {
    if (!isDragging.current) return;
    // Cancel any pending animation frame to prevent queuing multiple updates
    if (animationFrameRef.current) {
      cancelAnimationFrame(animationFrameRef.current);
    }
    
    // Schedule width update in next animation frame to prevent layout thrashing
    animationFrameRef.current = requestAnimationFrame(() => {
      const newWidth = Math.max(300, Math.min(800, window.innerWidth - e.clientX));
      setWidth(newWidth);
      animationFrameRef.current = null;
    });
  }, []);
  console.log(selectedStage);
  // Handle mouse up to stop resizing - memoized to prevent recreation
  const handleMouseUp = React.useCallback(() => {
    isDragging.current = false;
    document.body.style.cursor = '';
    document.removeEventListener('mousemove', handleMouseMove);
    document.removeEventListener('mouseup', handleMouseUp);
  }, []);
  
  // Render the appropriate content based on the active tab
  const renderTabContent = () => {
    // View mode toggle
    const viewModeToggle = (
      <div className="flex items-center justify-between bb b--black-20 pb2">
        <div className="flex items-center button-group">
          <button
            className={`px-3 py-1 rounded btn-small ${viewMode === 'form' ? 'shadow-[inset_0_1px_0_rgba(0,0,0,.05),inset_0_500px_0_0_var(--washed-gray),0_0_0_1px_var(--black-20)]' : 'ba bg-white b--black-20'}`}
            onClick={() => setViewMode('form')}
          >
            Form
          </button>
          <button
            className={`px-3 py-1 rounded btn-small ${viewMode === 'yaml' ? 'shadow-[inset_0_1px_0_rgba(0,0,0,.05),inset_0_500px_0_0_var(--washed-gray),0_0_0_1px_var(--black-20)]' : 'ba bg-white b--black-20'}`}
            onClick={() => setViewMode('yaml')}
          >
            Yaml
          </button>
        </div>
      </div>
    );

    switch (activeTab) {
      case 'general':
      return (
        <div className="pv2 ph3">
          <div className='flex items-center justify-between'>
            <h2 className="f7 ttu">Recent runs</h2>
            <button className="btn btn-link dark-indigo btn-small px-0">View all</button>
          </div>
          
          {/* Latest Run */}
          <RunItem
            status={selectedStage.data.status}
            commitTitle="Run #2"
            commitHash="1045a77"
            imageVersion="v.1.2.1"
            extraTags="+2 more"
            timestamp="8 minutes ago"
            date="Today"
            isHightlighted={true}
          />
          {selectedStage.data.status != 'Passed' && (
          
            <RunItem
              status="Passed"
              commitTitle="Run #1"
              commitHash="1045a77"
              imageVersion="v.1.2.0"
              extraTags="+2 more"
              timestamp="11 minutes ago"
              date="Today"
            />
          )}
         
          
          
          
          
          {/* Queue Section */}
          <div className='flex items-center justify-between'>
            <div className="ttu f7 mb1 mt3">QUEUE (3)</div>
            <button className={`btn btn-link dark-indigo btn-small px-0 ${isMsgDragStart ? "hidden" : "flex"}`} onClick={handleMsgDragStart}>Manage queue</button>
            <div className={`text-xs gray ml3-m ml0 tr ${isMsgDragStart ? "flex" : "hidden"}`}>
              <button className="btn btn-link dark-indigo btn-small px-0 mr3">Save</button>
              <button className="btn btn-link dark-indigo btn-small px-0" onClick={handleMsgDragStart}>Cancel</button>
            </div>
          </div>
          <MessageItem
            commitHash="1045a77"
            imageVersion="v.1.2.3"
            extraTags="+3 more"
            timestamp="8 minutes ago"
            date="Today"
            isDragStart={isMsgDragStart}
          />
          <MessageItem
            commitHash="1045a77"
            imageVersion="v.1.2.4"
            extraTags="+3 more"
            timestamp="11 minutes ago"
            date="Today"
            isDragStart={isMsgDragStart}
          />
          <MessageItem
            commitHash="1045a77"
            imageVersion="v.1.2.5"
            extraTags="+3 more"
            timestamp="14 minutes ago"
            approved={true}
            date="Today"
            isDragStart={isMsgDragStart}
            onRemove={() => {
              const newItems = selectedStage.data.queueItems.filter(item => item.commitHash !== "1045a77");
              selectedStage.data.queueItems = newItems;
            }}
          />
        </div>
      );
      
      case 'history':
      return (
        <div className='pv3 ph2'>
          <RunItem
            status="Passed"
            commitTitle="Run #4"
            commitHash="1045a77"
            imageVersion="v.1.2.1"
            extraTags="+1 more"
            timestamp="8 minutes ago"
            date="Today"
          />
          <RunItem
            status="Passed"
            commitTitle="Run #5"
            commitHash="1045a77"
            imageVersion="v.1.2.1"
            extraTags="+2 more"
            timestamp="8 minutes ago"
            date="Today"
          />
          <RunItem
            status="Failed"
            commitTitle="Run #6"
            commitHash="1045a77"
            imageVersion="v.1.2.1"
            extraTags="+2 more"
            timestamp="8 minutes ago"
            date="Today"
          />
          <RunItem
            status="Passed"
            commitTitle="Run #7"
            commitHash="1045a77"
            imageVersion="v.1.2.1"
            extraTags="+2 more"
            timestamp="8 minutes ago"
            date="Today"
          />
          <RunItem
            status="Passed"
            commitTitle="Run #8"
            commitHash="1045a77"
            imageVersion="v.1.2.1"
            extraTags="+2 more"
            timestamp="8 minutes ago"
            date="Today"
          />
          <RunItem
            status= "Failed"
            commitTitle="Run #9"
            commitHash="1045a77"
            imageVersion="v.1.2.1"
            extraTags="+2 more"
            timestamp="8 minutes ago"
            date="Today"
          />
        </div>
      );
      
      case 'history':
      return (
        <div className="pv3 ph4">
        
        <h2 className="f4 mb0">Run History</h2>
        <p className="mb3">A record of recent executions for this stage.</p>
        
        {/* Randomized history runs for visual variety */}
        <div className="bg-white shadow-1 mv3 ph3 pv2 br3">
        <div className="flex pv1">
        <div className="w-60 mb2 mb1">
        <div className="flex">
        <div className="flex-auto">
        <div className="flex">
        <img src={icnCommit} className="mt1 mr2" />
        <a href="workflow.html" className="measure truncate">FEAT-202: Add logging to API</a>
        </div>
        <div className="f5 overflow-auto nowrap mt1">
        <div className="flex items-center">
        <img src={faviconPinned} alt="Favicon" className="h1 w1 mr2" />
        <a href="workflow.html" className="link db flex-shrink-0 f6 w3 tc white mr2 ba br2 bg-green">Passed</a>
        <a href="workflow.html" className="link dark-gray underline-hover">Stage Deployment ⋮ <code className="f5 gray">09:12</code></a>
        </div>
        </div>
        </div>
        </div>
        </div>
        <div className="w-40">
        <div className="flex flex-row-reverse items-center">
        <img src={require("./images/profile-3.jpg")} width="32" height="32" className="db br-100 ba b--black-50" />
        <div className="f5 gray ml2 ml3-m ml0 mr3 tr">Today, 09:12<br /> by Alex Green</div>
        </div>
        </div>
        </div>
        <div className="flex">
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge"> code: 2d4e6f8</span>
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge">image: v.4.1.4</span>
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge ba b--black-50 bw1">terraform: v.2.3.0</span>
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge">type: enterprise</span>
        </div>
        </div>
        <div className="bg-white shadow-1 mv3 ph3 pv2 br3">
        <div className="flex pv1">
        <div className="w-60 mb2 mb1">
        <div className="flex">
        <div className="flex-auto">
        <div className="flex">
        <img src={icnCommit} className="mt1 mr2" />
        <a href="workflow.html" className="measure truncate">FIX-555: Resolve memory leak</a>
        </div>
        <div className="f5 overflow-auto nowrap mt1">
        <div className="flex items-center">
        <img src={faviconPinned} alt="Favicon" className="h1 w1 mr2" />
        <a href="workflow.html" className="link db flex-shrink-0 f6 w3 tc white mr2 ba br2 bg-red">Failed</a>
        <a href="workflow.html" className="link dark-gray underline-hover">Stage Deployment ⋮ <code className="f5 gray">08:45</code></a>
        </div>
        </div>
        </div>
        </div>
        </div>
        <div className="w-40">
        <div className="flex flex-row-reverse items-center">
        <img src={require("./images/profile-2.jpg")} width="32" height="32" className="db br-100 ba b--black-50" />
        <div className="f5 gray ml2 ml3-m ml0 mr3 tr">Today, 08:45<br /> by Nina Petrova</div>
        </div>
        </div>
        </div>
        <div className="flex">
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge"> code: 3e5f7h9</span>
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge">image: v.4.1.5</span>
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge ba b--black-50 bw1">terraform: v.2.2.1</span>
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge">type: community</span>
        </div>
        </div>
        <div className="bg-white shadow-1 mv3 ph3 pv2 br3">
        <div className="flex pv1">
        <div className="w-60 mb2 mb1">
        <div className="flex">
        <div className="flex-auto">
        <div className="flex">
        <img src={icnCommit} className="mt1 mr2" />
        <a href="workflow.html" className="measure truncate">DOCS-777: Update README.md</a>
        </div>
        <div className="f5 overflow-auto nowrap mt1">
        <div className="flex items-center">
        <img src={faviconPinned} alt="Favicon" className="h1 w1 mr2" />
        <a href="workflow.html" className="link db flex-shrink-0 f6 w3 tc white mr2 ba br2 bg-green">Passed</a>
        <a href="workflow.html" className="link dark-gray underline-hover">Stage Deployment ⋮ <code className="f5 gray">07:30</code></a>
        </div>
        </div>
        </div>
        </div>
        </div>
        <div className="w-40">
        <div className="flex flex-row-reverse items-center">
        <img src={require("./images/profile-3.jpg")} width="32" height="32" className="db br-100 ba b--black-50" />
        <div className="f5 gray ml2 ml3-m ml0 mr3 tr">Yesterday, 17:20<br /> by Marko Jovanovic</div>
        </div>
        </div>
        </div>
        <div className="flex">
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge"> code: 4f6g8h0</span>
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge">image: v.4.1.1</span>
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge ba b--black-50 bw1">terraform: v.2.1.0</span>
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge">type: docs</span>
        </div>
        </div>
        <div className="bg-white shadow-1 mv3 ph3 pv2 br3">
        <div className="flex pv1">
        <div className="w-60 mb2 mb1">
        <div className="flex">
        <div className="flex-auto">
        <div className="flex">
        <img src={icnCommit} className="mt1 mr2" />
        <a href="workflow.html" className="measure truncate">PERF-23: Load test improvements</a>
        </div>
        <div className="f5 overflow-auto nowrap mt1">
        <div className="flex items-center">
        <img src={faviconPinned} alt="Favicon" className="h1 w1 mr2" />
        <a href="workflow.html" className="link db flex-shrink-0 f6 w3 tc white mr2 ba br2 bg-green">Passed</a>
        <a href="workflow.html" className="link dark-gray underline-hover">Stage Deployment ⋮ <code className="f5 gray">06:55</code></a>
        </div>
        </div>
        </div>
        </div>
        </div>
        <div className="w-40">
        <div className="flex flex-row-reverse items-center">
        <img src={require("./images/profile-3.jpg")} width="32" height="32" className="db br-100 ba b--black-50" />
        <div className="f5 gray ml2 ml3-m ml0 mr3 tr">Yesterday, 12:10<br /> by Ana Milic</div>
        </div>
        </div>
        </div>
        <div className="flex">
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge"> code: 5j7k9l2</span>
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge">image: v.4.0.9</span>
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge ba b--black-50 bw1">terraform: v.2.0.9</span>
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge">type: perf</span>
        </div>
        </div>
        <div className="bg-white shadow-1 mv3 ph3 pv2 br3">
        <div className="flex pv1">
        <div className="w-60 mb2 mb1">
        <div className="flex">
        <div className="flex-auto">
        <div className="flex">
        <img src={icnCommit} className="mt1 mr2" />
        <a href="workflow.html" className="measure truncate">QA-72: Regression test suite</a>
        </div>
        <div className="f5 overflow-auto nowrap mt1">
        <div className="flex items-center">
        <img src={faviconPinned} alt="Favicon" className="h1 w1 mr2" />
        <a href="workflow.html" className="link db flex-shrink-0 f6 w3 tc white mr2 ba br2 bg-red">Failed</a>
        <a href="workflow.html" className="link dark-gray underline-hover">Stage Deployment ⋮ <code className="f5 gray">06:02</code></a>
        </div>
        </div>
        </div>
        </div>
        </div>
        <div className="w-40">
        <div className="flex flex-row-reverse items-center">
        <img src={require("./images/profile-4.jpg")} width="32" height="32" className="db br-100 ba b--black-50" />
        <div className="f5 gray ml2 ml3-m ml0 mr3 tr">Yesterday, 08:54<br /> by Jovana Simic</div>
        </div>
        </div>
        </div>
        <div className="flex">
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge"> code: 6m8n0p3</span>
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge">image: v.4.0.8</span>
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge ba b--black-50 bw1">terraform: v.2.0.8</span>
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge">type: qa</span>
        </div>
        </div>
        <div className="bg-white shadow-1 mv3 ph3 pv2 br3">
        <div className="flex pv1">
        <div className="w-60 mb2 mb1">
        <div className="flex">
        <div className="flex-auto">
        <div className="flex">
        <img src={icnCommit} className="mt1 mr2" />
        <a href="workflow.html" className="measure truncate">DB-44: Migrate to Postgres 15</a>
        </div>
        <div className="f5 overflow-auto nowrap mt1">
        <div className="flex items-center">
        <img src={faviconPinned} alt="Favicon" className="h1 w1 mr2" />
        <a href="workflow.html" className="link db flex-shrink-0 f6 w3 tc white mr2 ba br2 bg-green">Passed</a>
        <a href="workflow.html" className="link dark-gray underline-hover">Stage Deployment ⋮ <code className="f5 gray">05:43</code></a>
        </div>
        </div>
        </div>
        </div>
        </div>
        <div className="w-40">
        <div className="flex flex-row-reverse items-center">
        <img src={require("./images/profile-3.jpg")} width="32" height="32" className="db br-100 ba b--black-50" />
        <div className="f5 gray ml2 ml3-m ml0 mr3 tr">Yesterday, 07:30<br /> by Luka Stojanovic</div>
        </div>
        </div>
        </div>
        <div className="flex">
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge"> code: 7q9r1s4</span>
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge">image: v.4.0.7</span>
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge ba b--black-50 bw1">terraform: v.2.0.7</span>
        <span className="bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2 pipeline-badge">type: db</span>
        </div>
        </div>
        
        </div>
      );
      
      case 'queue':
      return (
        <div className="pv3 ph4">
        <h3 className="textg font-semibold mb-3">Queue</h3>
        <div className="deployment-queue">
        <div className="queue-item">
        <span className="queue-icon">⏳</span>
        <span>Feature: Add user authentication</span>
        </div>
        <div className="queue-item">
        <span className="queue-icon">⏳</span>
        <span>Bugfix: Fix login redirect</span>
        </div>
        <div className="queue-more">
        <a href="#">View all queue</a>
        </div>
        </div>
        </div>
      );
      
      case 'settings':
        return (
          <div className="pv2 ph3 h-full">
            {viewModeToggle}
            <div className="bg-white h-full">
              {viewMode === 'form' ? (
                <div className='text-sm pa2'>
                  <div className='pt2 mb2 ttu flex items-center b'><i className="material-symbols-outlined mid-gray mr1 f4 hidden">notes</i>Stage details</div>
                  <div className="space-y-2">
                    <div className="flex items-start w-full">
                      <div className='gray w-1/4'>Name</div>
                      <div className="block w-full">
                        {selectedStage?.data?.label || ''}
                      </div>
                    </div>
                    <div className="flex items-start w-full">
                      <div className='gray w-1/4'>ID</div>
                      <div className="block w-full">
                        {selectedStage?.id || ''}
                      </div>
                    </div>
                   
                  </div>
                  <div className='pt2 mb2 ttu flex items-center b'><i className="material-symbols-outlined mid-gray mr1 f4 fill hidden">play_arrow</i>Executor</div>
                  <div className="space-y-2">
                    <div className="flex items-start w-full">
                      <div className='gray w-1/4'>Type</div>
                      <div className="block w-full">
                        Semaphore
                      </div>
                    </div>
                    <div className="flex items-start w-full">
                      <div className='gray w-1/4'>Project ID</div>
                      <div className="block w-full">
                        1234567890
                      </div>
                    </div>
                    <div className="flex items-start w-full">
                      <div className='gray w-1/4'>Branch</div>
                      <div className="block w-full">
                        main
                      </div>
                    </div>
                    <div className="flex items-start w-full">
                      <div className='gray w-1/4'>Pipeline file</div>
                      <div className="block w-full">
                       .semaphore/pipeline_3.yml
                      </div>
                    </div>
                   
                  </div>
                  <div className='mt3 mb2 ttu flex items-center b'><i className="material-symbols-outlined mid-gray mr1 f4 hidden">local_police</i>Gates</div>
                  <div className="space-y-2">
                    <div className="flex items-start w-full">
                      <div className='gray w-1/4'>Manual approval</div>
                      <div className="block w-full">
                        True
                      </div>
                    </div>

                   
                  </div>
                  
                  <div className='mt3 mb2 ttu flex items-center b'><i className="material-symbols-outlined mid-gray mr1 f4 hidden">link</i>Connections</div>
                  <div className="space-y-2">
                    <div className="flex items-start w-full">
                      <div className='gray w-1/4'>#1</div>
                      <div className="flex items-center justify-between w-full">
                        <span className="bg-black-05 h-[26px] text-gray-600 text-xs px-1 py-1 br2 leading-none flex items-center ba b--black-05 code">
                          <i className="material-symbols-outlined mr1 text-xs">rocket_launch</i>
                          Deploy to US East
                        </span>
                        <span className='text-xs dark-green '>Stage</span>
                      </div>
                     
                    </div>
                    <div className="flex items-start w-full">
                      <div className='gray w-1/4'>#2</div>
                      <div className="flex items-center justify-between w-full">
                        <span className="bg-black-05 h-[26px] text-gray-600 text-xs px-1 py-1 br2 leading-none flex items-center ba b--black-05 code">
                          <i className="material-symbols-outlined mr1 text-xs">bolt</i>
                          Terraform
                        </span>
                        <span className='text-xs gray'>Event source</span>
                      </div>
                     
                    </div>
                   
                  </div>
                  <div className='mt3 mb2 ttu flex items-center b'><i className="material-symbols-outlined mid-gray mr1 f4 hidden">input</i>Inputs</div>
                  <div className="space-y-2">
                    <div className="flex items-start w-full">
                      <div className='gray w-1/4'>IMAGE</div>
                      <div className='w-full'>
                        <div className="flex items-center w-full">
                          <span className="bg-black-05 h-[26px] text-gray-600 text-xs px-1 py-1 br2 leading-none flex items-center ba b--black-05 code">
                            <i className="material-symbols-outlined mr1 text-xs">rocket_launch</i>
                            Deploy to US East
                          </span>.
                          <span className="bg-washed-purple h-[26px] text-gray-600 text-xs px-1 py-1 br2 leading-none flex items-center ba b--black-05 code">
                            outputs
                          </span>.
                          <span className="bg-washed-yellow h-[26px] text-gray-600 text-xs px-1 py-1 br2 leading-none flex items-center ba b--black-05 code">
                            IMAGE
                          </span>
                        </div>
                        <div className="flex items-center w-full mt1">
                          <span className="bg-black-05 h-[26px] text-gray-600 text-xs px-1 py-1 br2 leading-none flex items-center ba b--black-05 code">
                            <i className="material-symbols-outlined mr1 text-xs">rocket_launch</i>
                            Terraform
                          </span>.
                          <span className="bg-washed-purple h-[26px] text-gray-600 text-xs px-1 py-1 br2 leading-none flex items-center ba b--black-05 code">
                            lastExecution
                          </span>.
                          <span className="bg-washed-yellow h-[26px] text-gray-600 text-xs px-1 py-1 br2 leading-none flex items-center ba b--black-05 code">
                          [RESULT_PASSED]
                          </span>
                        </div>
                      </div>
                     
                    </div>
                    <div className="flex items-start w-full">
                      <div className='gray w-1/4'>TERRAFORM</div>
                      <div className='w-full'>
                        <div className="flex items-center w-full">
                          <span className="bg-black-05 h-[26px] text-gray-600 text-xs px-1 py-1 br2 leading-none flex items-center ba b--black-05 code">
                            <i className="material-symbols-outlined mr1 text-xs">rocket_launch</i>
                            Deploy to US East
                          </span>.
                          <span className="bg-washed-purple h-[26px] text-gray-600 text-xs px-1 py-1 br2 leading-none flex items-center ba b--black-05 code">
                            lastExecution
                          </span>.
                          <span className="bg-washed-yellow h-[26px] text-gray-600 text-xs px-1 py-1 br2 leading-none flex items-center ba b--black-05 code">
                          [RESULT_PASSED]
                          </span>
                        </div>
                        <div className="flex items-center w-full mt1">
                          <span className="bg-black-05 h-[26px] text-gray-600 text-xs px-1 py-1 br2 leading-none flex items-center ba b--black-05 code">
                            <i className="material-symbols-outlined mr1 text-xs">rocket_launch</i>
                            Terraform
                          </span>.
                          <span className="bg-washed-purple h-[26px] text-gray-600 text-xs px-1 py-1 br2 leading-none flex items-center ba b--black-05 code">
                          terraform
                          </span>.
                          <span className="bg-washed-yellow h-[26px] text-gray-600 text-xs px-1 py-1 br2 leading-none flex items-center ba b--black-05 code">
                          ref
                          </span>
                        </div>
                      </div>
                     
                    </div>
                   
                  </div>
                  <div className='mt3 mb2 ttu flex items-center b'><i className="material-symbols-outlined mid-gray mr1 f4 hidden">output</i>Outputs</div>
                  <div className="space-y-2">
                    <div className="flex items-start w-full">
                      <div className='gray w-1/4'>Image</div>
                      <div className="flex items-center justify-between w-full">
                        <div className='flex items-center'>
                          <span className="h-[26px] text-gray-600 text-xs px-1 py-1 br2 leading-none flex items-center ba b--black-05 code">this</span>.
                          <span className="bg-washed-purple h-[26px] text-gray-600 text-xs px-1 py-1 br2 leading-none flex items-center ba b--black-05 code">
                            outputs
                          </span>.
                          <span className="bg-washed-yellow h-[26px] text-gray-600 text-xs px-1 py-1 br2 leading-none flex items-center ba b--black-05 code">
                            IMAGE
                          </span>
                        </div>
                        <span className='text-xs dark-red'>required</span>
                      </div>
                    </div>
                    
                   
                  </div>
                </div>
              ) : (
                <div className="">
                  <pre className="bg-white p-4 code text-xs">
                    {JSON.stringify(selectedStage, null, 2)}
                  </pre>
                </div>
              )}
            </div>
          </div>
        );
      
      default:
      return null;
    }
  };
  
  return (
    <aside
    ref={sidebarRef}
    className="sidebar"
    style={{
      width: width,
      minWidth: 300,
      maxWidth: 800,
      position: 'absolute',
      height: 'auto',
      top: 48,
      right: 0,
      bottom: 0,
      zIndex: 10,
      boxShadow: 'rgba(0,0,0,0.07) -2px 0 12px',
      background: '#fff',
      transition: isDragging.current ? 'none' : 'width 0.2s',
      display: 'flex',
      flexDirection: 'column',
    }}
    >
    {/* Sidebar Header with Stage Name */}
    <div className="sidebar-header ">
    <div className="sidebar-header-title flex items-center">
    {selectedStage.type === 'deploymentCard' ? (
        <span class="material-symbols-outlined mr1">rocket_launch</span>
    
    ) : (
      <svg viewBox="0 0 16 16" width="20" height="20" fill="currentColor">
      <path fillRule="evenodd" d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.01.08-2.11 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.11.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.19 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/>
      </svg>
    )}
    <span className="f4 b">{selectedStage.data.label}</span>
    </div>
    <button className="pa0 bg-transparent" onClick={onClose} title="Close sidebar"><i className="material-symbols-outlined">close</i></button>
    </div>

   
    <div className="flex items-start px-4 hidden">
      <i className="material-symbols-outlined mr1 text-sm">play_circle</i>
      <div className="text-sm">
        <div className="mb1 ttu">Executor</div>
        <div className="flex items-center code text-xs">
          <div className="gray"><div><img src={semaphore} width={20} className="mx-1"/><span className='b'>Semaphore project/Pipeline name</span></div>
            <div className="b bg-black-05">Image</div>
            <div>Terraform</div><div>Something</div>
          </div>
          <div className="">
            <div className="pl2">1045a77</div>
            <div className="b bg-black-05 pl2">v.1.2.1</div>
            <div className="pl2">32.32</div>
            <div className="pl2">adsfasdf</div>
          </div>
        </div>
      </div>
    </div>
   
    {/* Sidebar Tabs */}
    <div className="sidebar-tabs ph2">
    {tabs.map(tab => (
      <button
      key={tab.key}
      className={`tab-button${activeTab === tab.key ? ' active' : ''}`}
      onClick={() => setActiveTab(tab.key)}
      >
      {tab.label}
      </button>
    ))}
    </div>
    <div className="sidebar-content bg-near-white min-h-0 relative overflow-auto">
    {renderTabContent()}
    </div>
    
    
    {/* Resize Handle */}
    <div
    className="resize-handle"
    style={{
      width: 8,
      cursor: 'ew-resize',
      background: isDragging.current ? '#e0e0e0' : '#f0f0f0',
      position: 'absolute',
      left: 0,
      top: 0,
      bottom: 0,
      zIndex: 100,
      borderRadius: '4px',
      transition: 'background 0.2s',
    }}
    onMouseDown={handleMouseDown}
    onMouseEnter={() => { if (!isDragging.current) sidebarRef.current.style.cursor = 'ew-resize'; }}
    onMouseLeave={() => { if (!isDragging.current) sidebarRef.current.style.cursor = 'default'; }}
    />
    </aside>
  );
});

// Initial stages configuration
const chainLength = 5;
// Allow x/y for each stage in the new chain
const newChainStagePositions = [
  { x: -400, y: -730 }, // prod-cluster (Kubernetes Integration)
  { x: 100, y: -800 },  // Stage 1 (straight right)
  { x: 600, y: -1000 },  // Stage 2 (up and right, parallel)
  { x: 600, y: -600 },  // Stage 3 (down and right, parallel)
  { x: 1150, y: -800 }, // Stage 4 (centered between 2 and 3, further right)
];
const newChainStages = [
  {
    id: String(1000),
    type: 'githubIntegration',
    data: {
      repoName: 'prod-cluster',
      repoUrl: 'europe-west3-a/prod-cluster',
      lastEvent: {
        type: 'zebra',
        release: 'Updated, Endpoints Changed',
        timestamp: '2025-04-09 09:30 AM',
      },
      status: 'Idle',
      timestamp: 'Never run',
      labels: ['new123', 'v.0.1.0', 'integration'],
      queue: [],
      integrationType: 'kubernetes',
    },
    position: newChainStagePositions[0],
    style: { width: 320 },
  },
  {
    id: String(1001),
    type: 'deploymentCard',
    data: {
      icon: 'storage',
      label: 'Sync Cluster',
      status: 'Running',
      timestamp: 'Built 15 min ago',
      labels: ['docker', 'build', 'v.1.0.0'],
      queue: ['Build: Dockerfile', 'Push: Container Registry'],
      queueIcon: 'pending',
      queueIconClass: 'indigo',
    },
    position: newChainStagePositions[1],
    style: { width: 320 },
  },
  {
    id: String(1002),
    type: 'deploymentCard',
    data: {
      icon: 'cloud',
      label: 'Deploy to US cluster',
      status: 'Running',
      timestamp: 'Deploying now',
      labels: ['staging', 'v.1.0.0'],
      queue: ['Deploy: Helm Chart', 'Scale: Increase replicas'],
      queueIcon: 'pending',
      queueIconClass: 'indigo',
    },
    position: newChainStagePositions[2],
    style: { width: 320 },
  },
  {
    id: String(1003),
    type: 'deploymentCard',
    data: {
      icon: 'cloud_done',
      label: 'Deploy to Asia cluster',
      status: 'Passed',
      timestamp: 'Completed 10 min ago',
      labels: ['tests', 'integration', 'v.1.0.0'],
      queue: [],
      queueIcon: 'pending',
      queueIconClass: 'indigo',
    },
    position: newChainStagePositions[3],
    style: { width: 320 },
  },
  {
    id: String(1004),
    type: 'deploymentCard',
    data: {
      icon: 'cloud_done',
      label: 'Health Check & Cleanup',
      status: 'Failed',
      timestamp: 'Ready for deployment',
      labels: ['production', 'v.1.0.0'],
      queue: ['Deploy: Helm Chart'],
      queueIcon: 'flaky',
      queueIconClass: 'purple',
    },
    position: newChainStagePositions[4],
    style: { width: 320 },
  },
];
const newChainListeners = [];
// prod-cluster → Sync Cluster (1000 → 1001)
newChainListeners.push({
  id: 'e1000-1001',
  source: '1000',
  target: '1001',
  type: 'bezier',
  animated: true,
  style: { stroke: '#888', strokeDasharray: '6 4', strokeWidth: 2 },
  label: 'Promote to Sync Cluster',
  labelStyle: { fill: '#000', fontWeight: 500 },
  labelBgStyle: { fill: 'rgba(255, 255, 255, 0.9)', fillOpacity: 0.9 },
  markerEnd: { type: MarkerType.ArrowClosed },
});
// Sync Cluster → Deploy to US cluster (1001 → 1002)
newChainListeners.push({
  id: 'e1001-1002',
  source: '1001',
  target: '1002',
  type: 'bezier',
  animated: true,
  style: { stroke: '#888', strokeDasharray: '6 4', strokeWidth: 2 },
  label: 'Sync → US Cluster',
  labelStyle: { fill: '#000', fontWeight: 500 },
  labelBgStyle: { fill: 'rgba(255, 255, 255, 0.9)', fillOpacity: 0.9 },
  markerEnd: { type: MarkerType.ArrowClosed },
});
// Sync Cluster → Deploy to Asia cluster (1001 → 1003)
newChainListeners.push({
  id: 'e1001-1003',
  source: '1001',
  target: '1003',
  type: 'bezier',
  animated: false,
  style: { stroke: '#888', strokeWidth: 2 },
  label: 'Sync → Asia Cluster',
  labelStyle: { fill: '#000', fontWeight: 500 },
  labelBgStyle: { fill: 'rgba(255, 255, 255, 0.9)', fillOpacity: 0.9 },
  markerEnd: { type: MarkerType.ArrowClosed },
});
// US cluster → Health Check & Cleanup (1002 → 1004)
newChainListeners.push({
  id: 'e1002-1004',
  source: '1002',
  target: '1004',
  type: 'bezier',
  animated: false,
  style: { stroke: '#888', strokeWidth: 2 },
  label: 'US → Cleanup',
  labelStyle: { fill: '#000', fontWeight: 500 },
  labelBgStyle: { fill: 'rgba(255, 255, 255, 0.9)', fillOpacity: 0.9 },
  markerEnd: { type: MarkerType.ArrowClosed },
});
// Asia cluster → Health Check & Cleanup (1003 → 1004)
newChainListeners.push({
  id: 'e1003-1004',
  source: '1003',
  target: '1004',
  type: 'bezier',
  animated: false,
  style: { stroke: '#888', strokeWidth: 2 },
  label: 'Asia → Cleanup',
  labelStyle: { fill: '#000', fontWeight: 500 },
  labelBgStyle: { fill: 'rgba(255, 255, 255, 0.9)', fillOpacity: 0.9 },
  markerEnd: { type: MarkerType.ArrowClosed },
});

const initialStages = [
  // First workflow - Semaphore
  {
    id: '0',
    type: 'githubIntegration',
    data: { 
      repoName: 'semaphoreio/semaphore',
      repoUrl: 'https://github.com/semaphoreio/semaphore',
      lastEvent: {
        type: 'push',
        release: 'main',
        timestamp: '2025-04-09 09:30 AM'
      },
      status: 'Passed',
      timestamp: 'Deployed 2 hours ago',
      labels: ['1045a77', 'v.4.1.3', 'v.2.3.1', 'community'],
      queue: ['Feature: Add user authentication', 'Bugfix: Fix login redirect', 'Feature: Add dark mode'],
      queueIcon: 'flaky', // default icon
      queueIconClass: 'purple', // default color class
    positionAbsolute: { left: Position.Left, right: Position.Right },

    },
    position: { x: -400, y: 159 },
    style: {
      width: 320,
    },
  },
  {
    id: '1',
    type: 'deploymentCard',
    data: { 
      icon: 'storage',
      label: 'Development Environment',
      status: 'Passed',
      timestamp: 'Deployed 2 hours ago',
      labels: ['1045a77', 'v.4.1.3', 'v.2.3.1', 'community'],
      queue: ['Feature: Add user authentication', 'Bugfix: Fix layout on mobile', 'Feature: Add dark mode'],
      queueIcon: 'flaky', // default icon
      queueIconClass: 'purple', // default color class
    positionAbsolute: { left: Position.Left, right: Position.Right },

    },
    position: { x: 100, y: 77 },
    style: {
      width: 320,
    },
  },
  {
    id: '2',
    type: 'deploymentCard',
    data: { 
      icon: 'storage',
      label: 'Staging Environment',
      status: 'Passed',
      timestamp: 'Deployed just now',
      labels: ['7a9b23c', 'v.4.1.3', 'v.2.3.1', 'community'],
      queue: ['FEAT-312: Investigate flaky test'],
      queueIcon: 'flaky', // default icon
      queueIconClass: 'purple', // default color class
    positionAbsolute: { left: Position.Left, right: Position.Right },

    },
    position: { x: 600, y: 122 },
    style: {
      width: 320,
    },
  },
  {
    id: '3',
    type: 'deploymentCard',
    data: { 
      icon: 'cloud',
      label: 'Production - US',
      status: 'Failed',
      timestamp: 'Failed just now',
      labels: ['5e3d12b', 'v.4.1.3', 'v.2.3.1', 'community'],
      queue: [
        'FEAT-400: Flaky test detected',
        'BUG-512: Flaky network error'
      ],
      queueIcon: 'flaky', // default icon
      queueIconClass: 'purple', // default color class
    positionAbsolute: { left: Position.Left, right: Position.Right },

    },
    position: { x: 1150, y: -150 },
    style: {
      width: 320,
    },
  },
  {
    id: '5',
    type: 'deploymentCard',
    data: { 
      icon: 'cloud',
      label: 'Production - JP',
      status: 'Passed',
      timestamp: 'Deployed just now',
      labels: ['5e3d12b', 'v.4.1.3', 'v.2.3.1', 'community'],
      queue: ['FEAT-211: Partially rebuild pipeline'],
      queueIcon: 'timer', // orange timer icon
      queueIconClass: 'orange', // orange color class
    positionAbsolute: { left: Position.Left, right: Position.Right },

    },
    position: { x: 1750, y: -128 },
    style: {
      width: 320,
    },
  },
  {
    id: '4',
    type: 'deploymentCard',
    data: { 
      icon: 'cloud',
      label: 'Production - EU',
      status: 'Running',
      timestamp: 'Deploying now',
      labels: ['5e3d12b', 'v.4.1.3', 'v.2.3.1', 'community'],
      queue: [],
      queueIcon: 'flaky', // default icon
      queueIconClass: 'purple', // default color class
    positionAbsolute: { left: Position.Left, right: Position.Right },

    },
    position: { x: 1150, y: 450 },
    style: {
      width: 320,
    },
  },
  
  // Second workflow - Toolbox
  {
    id: '6',
    type: 'githubIntegration',
    data: { 
      repoName: 'buckets/my-app-data',
      repoUrl: 'https://s3.console.aws.amazon.com/s3/buckets/my-app-data',
      lastEvent: {
        type: 'push',
        release: 'main',
        timestamp: '2025-04-09 09:30 AM'
      },
      status: 'Passed',
      timestamp: 'Deployed 2 hours ago',
      labels: ['3e7a91d', 'v.4.1.3', 'v.2.3.1', 'community'],
      queue: ['Test: Integration tests', 'Test: Performance benchmarks'],
      queueIcon: 'flaky', // default icon
      queueIconClass: 'purple', // default color class
    positionAbsolute: { left: Position.Left, right: Position.Right },

    },
    position: { x: -400, y: 888 },
    style: {
      width: 320,
    },
  },
  {
    id: '7',
    type: 'deploymentCard',
    data: { 
      icon: 'storage',
      label: 'Platform Test',
      status: 'Passed',
      timestamp: 'Completed 1 hour ago',
      labels: ['3e7a91d', 'v.4.1.3', 'v.2.3.1', 'community'],
      queue: ['Test: Integration tests', 'Test: Performance benchmarks'],
      queueIcon: 'flaky', // default icon
      queueIconClass: 'purple', // default color class
    positionAbsolute: { left: Position.Left, right: Position.Right },

    },
    position: { x: 100, y: 827 },
    style: {
      width: 320,
    },
  },
  {
    id: '8',
    type: 'deploymentCard',
    data: { 
      icon: 'lan',
      label: 'Infra - Publish',
      status: 'Running',
      timestamp: 'Deploying now',
      labels: ['3e7a91d', 'v.4.1.3', 'v.2.3.1', 'community'],
      queue: [],
      queueIcon: 'flaky', // default icon
      queueIconClass: 'purple', // default color class,
    positionAbsolute: { left: Position.Left, right: Position.Right },

    },
    position: { x: 600, y: 860 },
    style: {
      width: 320,
    },
  },
  // New chain stages
  ...newChainStages,
];

// Initial listeners configuration - connecting the stages
const initialListeners = [
  // First workflow connections
  {
    id: 'e0-1',
    source: '0',
    target: '1',
    type: 'bezier',
    animated: false,
    label: 'Trigger Build',
    style: { stroke: '#888', strokeWidth: 2 },
    labelStyle: { fill: '#000', fontWeight: 500 },
    labelBgStyle: { fill: 'rgba(255, 255, 255, 0.9)', fillOpacity: 0.9 },
  },
  {
    id: 'e1-2',
    source: '1',
    target: '2',
    type: 'bezier',
    animated: false,
    label: 'Promote to Staging',
    style: { stroke: '#888', strokeWidth: 2 },
    labelStyle: { fill: '#000', fontWeight: 500 },
    labelBgStyle: { fill: 'rgba(255, 255, 255, 0.9)', fillOpacity: 0.9 },
  },
  {
    id: 'e2-3',
    source: '2',
    target: '3',
    type: 'bezier',
    animated: false,
    label: 'Promote to US',
    style: { stroke: '#888', strokeWidth: 2 },
    labelStyle: { fill: '#000', fontWeight: 500 },
    labelBgStyle: { fill: 'rgba(255, 255, 255, 0.9)', fillOpacity: 0.9 },
  },
  {
    id: 'e2-4',
    source: '2',
    target: '4',
    type: 'bezier',
    animated: true,
    label: 'Promote to EU',
    style: { stroke: '#888', strokeDasharray: '6 4', strokeWidth: 2 },
    labelStyle: { fill: '#000', fontWeight: 500 },
    labelBgStyle: { fill: 'rgba(255, 255, 255, 0.9)', fillOpacity: 0.9 },
  },
  {
    id: 'e3-5',
    source: '3',
    target: '5',
    type: 'bezier',
    animated: false,
    label: 'Promote to JP',
    style: { stroke: '#888', strokeWidth: 2 },
    labelStyle: { fill: '#000', fontWeight: 500 },
    labelBgStyle: { fill: 'rgba(255, 255, 255, 0.9)', fillOpacity: 0.9 },
  },
  
  // Second workflow connections
  {
    id: 'e6-7',
    source: '6',
    target: '7',
    type: 'bezier',
    animated: false,
    label: 'Run Tests',
    style: { stroke: '#888', strokeWidth: 2 },
    labelStyle: { fill: '#000', fontWeight: 500 },
    labelBgStyle: { fill: 'rgba(255, 255, 255, 0.9)', fillOpacity: 0.9 },
  },
  {
    id: 'e7-8',
    source: '7',
    target: '8',
    type: 'bezier',
    animated: true,
    label: 'Deploy to Production',
    style: { stroke: '#888', strokeDasharray: '6 4', strokeWidth: 2 },
    labelStyle: { fill: '#000', fontWeight: 500 },
    labelBgStyle: { fill: 'rgba(255, 255, 255, 0.9)', fillOpacity: 0.9 },
  },
  // New chain listeners
  ...newChainListeners,
];

function WorkflowEditor() {
  const [stages, setStages, onStagesChange] = useNodesState(initialStages);
  const [listeners, setListeners, onListenersChange] = useEdgesState(initialListeners);
  const [selectedStage, setSelectedStage] = useState(null);
  const [selectedEdge, setSelectedEdge] = useState(null);
  const [iconAction, setIconAction] = useState(null); 
  const [reactFlowInstance, setReactFlowInstance] = useState(null);
  
  const SIDEBAR_WIDTH = 400;
  
  // Handle stage deletion
  const handleDeleteStage = (stageId) => {
    // Remove the stage
    setStages((currentStages) => currentStages.filter(stage => stage.id !== stageId));
    
    // Remove any connections to/from this stage
    setListeners((currentListeners) => 
      currentListeners.filter(listener => 
        listener.source !== stageId && listener.target !== stageId
      )
    );
    
    // Close sidebar if the deleted stage was selected
    if (selectedStage && selectedStage.id === stageId) {
      setSelectedStage(null);
    }
  };
  
  // Handle edge deletion
  const handleDeleteEdge = (edgeId) => {
    // Remove the edge
    setListeners((currentListeners) => 
      currentListeners.filter(listener => listener.id !== edgeId)
    );
    
    // Clear selected edge
    setSelectedEdge(null);
  };
  
  // Define stage types using memoization to prevent unnecessary re-renders
  const stageTypes = React.useMemo(() => ({
    deploymentCard: (props) => <DeploymentCardStage {...props} onDelete={handleDeleteStage} id={props.id}/>,
    githubIntegration: GitHubIntegration,
  }), []); 
  
  // Helper to generate a unique Stage ID
  const generateStageId = (existingStages) => {
    let maxId = 0;
    existingStages.forEach(s => {
      const idNum = parseInt(s.id, 10);
      if (!isNaN(idNum) && idNum >= maxId) maxId = idNum + 1;
    });
    return String(maxId);
  };
  
  // Handle new connections between stages
  const onConnect = useCallback(
    (params) => setListeners((eds) => {
      // Animate/dash if connecting from staging (2 or 7) to production (4, 5, or 8) but NOT 3
      const stagingIds = ['2', '7'];
      const dashedProductionIds = ['4', '5', '8'];
      if ((stagingIds.includes(params.source) && dashedProductionIds.includes(params.target)) ||
      (dashedProductionIds.includes(params.source) && stagingIds.includes(params.target))) {
        return addEdge({ ...params, type: ConnectionLineType.Bezier, animated: true, style: { stroke: '#888', strokeDasharray: '6 4', strokeWidth: 2 } }, eds);
      }
      return addEdge({ ...params, type: ConnectionLineType.Bezier, animated: false, style: { stroke: '#888', strokeWidth: 2 } }, eds);
    }),
    [setListeners]
  );
  
  // Handle stage click to show sidebar and zoom
  const onStageClick = useCallback((event, stage) => {
    setSelectedStage(stage);
    // Zoom into the selected stage if instance is available
    if (reactFlowInstance && stage && stage.position) {
      reactFlowInstance.setCenter(
        stage.position.x + (stage.style?.width || 320) / 2 + SIDEBAR_WIDTH / 2,
        stage.position.y + 80, 
        { zoom: 1.2, duration: 800 }
      );
    }
    setIconAction(null); 
  }, [reactFlowInstance]);
  
  // Close sidebar
  const closeSidebar = () => {
    setSelectedStage(null);
  };
  
  // Handle pane click to close sidebar when clicking on empty canvas
  const onPaneClick = useCallback(() => {
    setSelectedStage(null);
    setSelectedEdge(null);
  }, []);
  
  // Handle edge click to select/deselect edge
  const onEdgeClick = useCallback((event, edge) => {
    event.stopPropagation(); // Prevent triggering pane click
    setSelectedEdge(prev => prev?.id === edge.id ? null : edge);
    setSelectedStage(null); // Deselect any selected stage
  }, []);

  // Handle node addition from sidebar
  const handleAddNode = (type, position) => {
    console.log(position);
    const newId = generateStageId(stages);
    let data = {};
    if(type === 'deploymentCard') {
      data = {  
        icon: 'cloud_done',
        label: 'Deploy to Asia cluster',
        status: 'Passed',
        timestamp: 'Completed 10 min ago',
        labels: ['tests', 'integration', 'v.1.0.0'],
        lastEvent: {
          type: 'push',
          release: 'main',
          timestamp: '2025-04-09 09:30 AM'
        },
        queue: [],
        queueIcon: 'pending',
        queueIconClass: 'indigo',
      }
    }else{
        data = {
          repoName: 'semaphoreio/semaphore',
          repoUrl: 'https://github.com/semaphoreio/semaphore',
          lastEvent: {
            type: 'push',
            release: 'main',
            timestamp: '2025-04-09 09:30 AM'
          },
          status: 'Passed',
          timestamp: 'Deployed 2 hours ago',
          labels: ['1045a77', 'v.4.1.3', 'v.2.3.1', 'community'],
          queue: ['Feature: Add user authentication', 'Bugfix: Fix login redirect', 'Feature: Add dark mode'],
          queueIcon: 'flaky',
          queueIconClass: 'purple',
          style: { width: 320 }
        }
    }
    const newNode = {
      id: newId,
      type: type,
      position: position,
        data: data,
      style: {
        width: 320,
      }

    };
    setStages((currentStages) => [...currentStages, newNode]);
  };

  // Handle drag start from sidebar
  const [draggingType, setDraggingType] = useState(null);
  const [draggingPosition, setDraggingPosition] = useState(null);

  const handleDragStart = (event, type) => {
    event.dataTransfer.setData('text/plain', type);
    event.dataTransfer.effectAllowed = 'copy';
    event.target.style.cursor = 'grabbing';
    setDraggingType(type);
  };

  const handleDragOver = useCallback((event) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'copy';
    const rect = reactFlowWrapper.current.getBoundingClientRect();
    const position = {
      x: event.clientX - rect.left,
      y: event.clientY - rect.top
    };
    setDraggingPosition(position);
  }, []);

  const handleDrop = useCallback((event) => {
    event.preventDefault();
    const type = event.dataTransfer.getData('text/plain');
    const rect = reactFlowWrapper.current.getBoundingClientRect();
    const position = {
      x: event.clientX - rect.left,
      y: event.clientY - rect.top
    };
    handleAddNode(type, position);
    setDraggingType(null);
    setDraggingPosition(null);
  }, [handleAddNode]);

  // Add drag event listeners to the ReactFlow wrapper
  const handleDragLeave = () => {
    setDraggingType(null);
    setDraggingPosition(null);
  };

  useEffect(() => {
    if (reactFlowWrapper.current) {
      reactFlowWrapper.current.addEventListener('dragover', handleDragOver);
      reactFlowWrapper.current.addEventListener('drop', handleDrop);
      reactFlowWrapper.current.addEventListener('dragleave', handleDragLeave);
    }
    return () => {
      if (reactFlowWrapper.current) {
        reactFlowWrapper.current.removeEventListener('dragover', handleDragOver);
        reactFlowWrapper.current.removeEventListener('drop', handleDrop);
        reactFlowWrapper.current.removeEventListener('dragleave', handleDragLeave);
      }
    };
  }, [handleDragOver, handleDrop, handleDragLeave]);

  // Render a preview node while dragging
  const renderDragPreview = () => {
    if (!draggingType || !draggingPosition) return null;
    const previewNode = {
      id: 'preview',
      type: draggingType === 'semaphore' ? 'githubIntegration' : draggingType,
      position: draggingPosition,
      data: {
        title: draggingType.charAt(0).toUpperCase() + draggingType.slice(1),
        description: 'Short description of the workflow recipe goes in here.',
        style: { width: 320 }
      },
      style: {
        opacity: 0.7,
        backgroundColor: '#f0f0f0',
        border: '2px dashed #ccc'
      }
    };
    return (
      <div
        style={{
          position: 'absolute',
          left: draggingPosition.x,
          top: draggingPosition.y,
          pointerEvents: 'none'
        }}
      >
        <div className="node-preview">
          <div className="node-content">
            <h3>{previewNode.data.title}</h3>
            <p>{previewNode.data.description}</p>
          </div>
        </div>
      </div>
    );
  };


  
  // Handle icon block actions
  const handleIconAction = (action) => {
    setIconAction(action);
    // You can perform additional logic here, e.g., open modals, show info, etc.
  };
  

  
  // Ref for the ReactFlow wrapper div
  const reactFlowWrapper = useRef(null);

  // Export handler
  const handleExport = () => {
    if (!reactFlowWrapper.current) return;
    htmlToImage.toPng(reactFlowWrapper.current.querySelector('.react-flow'))
      .then((dataUrl) => {
        const link = document.createElement('a');
        link.download = 'workflow-chain.png';
        link.href = dataUrl;
        link.click();
      })
      .catch((err) => {
        alert('Failed to export image: ' + err);
      });
  };

  // Create edge styles with selection highlight and hide labels
  const edgesWithStyles = React.useMemo(() => {
    return listeners.map(edge => {
      // Create a new edge object without the label property
      const { label, ...edgeWithoutLabel } = edge;
      
      return {
        ...edgeWithoutLabel,
        style: {
          ...edge.style,
          stroke: selectedEdge?.id === edge.id ? '#3b82f6' : '#888888', // Gray connectors, blue when selected
          strokeWidth: selectedEdge?.id === edge.id ? 3 : edge.style?.strokeWidth || 2,
        },
        // Keep other properties but remove visible label
        labelStyle: {
          ...edge.labelStyle,
          fill: 'transparent', // Make text transparent (invisible)
        },
        labelBgStyle: {
          ...edge.labelBgStyle,
          fill: 'transparent', // Make background transparent
          fillOpacity: 0,
        },
      };
    });
  }, [listeners, selectedEdge]);
  
  // Use memoization to prevent unnecessary re-renders of ReactFlow
  const reactFlowElement = React.useMemo(() => (
    <ReactFlow
      nodes={stages}
      edges={edgesWithStyles}
      onNodesChange={onStagesChange}
      onEdgesChange={onListenersChange}
      onConnect={onConnect}
      onNodeClick={onStageClick}
      onEdgeClick={onEdgeClick}
      onPaneClick={onPaneClick}
      nodeTypes={stageTypes}
      connectionLineType={ConnectionLineType.Bezier}
      fitView
      fitViewOptions={{ padding: 0.3 }}
      minZoom={0.4}
      maxZoom={1.5}
      onInit={setReactFlowInstance}
      style={{ width: '100%', height: '100%' }} // Fixed dimensions to prevent layout shifts
    >
      <Controls style={{ position: 'absolute', right: 8, left: 'auto', bottom: 8 }} />
      <Background variant={BackgroundVariant.Dots} gap={24} size={2} color="#96A0A6" className="bg-gray-100"/>
    </ReactFlow>
  ), [stages, listeners, onStagesChange, onListenersChange, onConnect, onStageClick, onPaneClick, stageTypes, edgesWithStyles, onEdgeClick]);
  
  return (
    <div className='flex h-full w-full page-wrapper'>
      <Navigation className="flex"/>
    
      
      
    <div className="flex w-full h-full" ref={reactFlowWrapper}>
      <div className='group peer relative block h-full bg-transparent'>
      <ComponentSidebar onAddNode={handleAddNode} onDragStart={handleDragStart} />
      </div>
      {renderDragPreview()}
     
      <button
        onClick={handleExport}
        style={{ position: 'absolute', top: 4, right: 8, zIndex: 1000, background: '#222', color: 'white', padding: '8px 16px', borderRadius: 4, border: 'none', fontWeight: 600, cursor: 'pointer', boxShadow: '0 2px 8px rgba(128,128,128,0.20)' }}
      >
        Export as Image
      </button>
      <div className="flex-grow h-full" style={{ position: 'relative', zIndex: 1 }}>
        {reactFlowElement}
      </div>
      
      {/* Edge Delete UI */}
      {renderDragPreview()}
      {selectedEdge && (
        <div 
          className="absolute flex gap-2 bg-white shadow-gray-lg px-3 py-2 border z-10 rounded-lg"
          style={{ 
            top: '50%', 
            left: '50%', 
            transform: 'translate(-50%, -50%)',
            zIndex: 1000,
          }}
        >
          <div className="flex flex-col items-center">
            <div className="mb-2 font-medium">Selected Connection: {selectedEdge.id}</div>
            <Tippy content="Delete this connection" placement="top">
              <button 
                className="hover:bg-red-100 text-red-600 p-2 rounded-md flex items-center" 
                title="Delete Connection"
                onClick={() => handleDeleteEdge(selectedEdge.id)}
              >
                <span className="material-icons" style={{fontSize:20}}>delete</span>
                <span className="ml-2">Delete Connection</span>
              </button>
            </Tippy>
          </div>
        </div>
      )}
      
      {selectedStage && (
        <Sidebar 
          selectedStage={selectedStage} 
          onClose={closeSidebar} 
        />
      )}
    </div>
    </div>
  );
}

export default WorkflowEditor;

<style jsx>{`
  .pipeline-badge {
    display: inline-flex;
    align-items: center;
    height: 1.8em;
  }
  .status-badge {
    display: inline-flex;
    align-items: center;
    height: 1.8em;
    padding: 0.2em 0.5em;
    border-radius: 0.2em;
    font-size: 0.8em;
    font-weight: 500;
  }
  .status-badge.passed {
    background-color: #c6efce;
    color: #2e865f;
  }
  .status-badge.running {
    background-color: #f7d2c4;
    color: #7a2518;
  }
  .status-badge.failed {
    background-color: #f2c6c6;
    color: #7a2518;
  }
`}</style>
  
  function OverlayModal({ open, onClose, children }) {
    if (!open) return null;
    return (
      <div className="modal is-open" aria-hidden={!open} style={{position:'fixed',top:0,left:0,right:0,bottom:0,zIndex:999999}}>
      <div className="modal-overlay" style={{position:'fixed',top:0,left:0,right:0,bottom:0,background:'rgba(40,50,50,0.6)',zIndex:999999}} onClick={onClose} />
      <div className="modal-content" style={{position:'fixed',top:'50%',left:'50%',transform:'translate(-50%, -50%)',zIndex:1000000,background:'#fff',borderRadius:8,boxShadow:'0 6px 40px rgba(128,128,128,0.20)',maxWidth:600,width:'90vw',padding:32}}>
      <button onClick={onClose} style={{position:'absolute',top:8,right:12,background:'none',border:'none',fontSize:26,color:'#888',cursor:'pointer'}} aria-label="Close">×</button>
      {children}
      </div>
      </div>
    );
  }
  