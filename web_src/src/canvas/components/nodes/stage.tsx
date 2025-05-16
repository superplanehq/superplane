import { useState, ReactNode, useMemo } from 'react';
import type { NodeProps } from '@xyflow/react';
import Tippy from '@tippyjs/react';
import 'tippy.js/dist/tippy.css'; 
import CustomBarHandle from './handle';
import { StageNodeType, QueueState } from '@/canvas/types/flow';
// import { QueueState } from '../../types/flow';

// Define the data type for the deployment card
// Using Record<string, unknown> to satisfy ReactFlow's Node constraint
export default function StageNode(props: NodeProps<StageNodeType>) {
  const [showOverlay, setShowOverlay] = useState(false);
  
  // Filter events by their state
  const pendingEvents = useMemo(() => 
    props.data.queues?.filter(event => event.state === QueueState.PENDING) || [], 
    [props.data.queues]
  );
  
  const waitingEvents = useMemo(() => 
    props.data.queues?.filter(event => event.state === QueueState.WAITING) || [], 
    [props.data.queues]
  );
  
  const processedEvents = useMemo(() => 
    props.data.queues?.filter(event => event.state === QueueState.PROCESSED) || [], 
    [props.data.queues]
  );
  
  const handleAction = (action: string) => {
    if (action === 'code') setShowOverlay(true);
  };
  return (
    <div className={`bg-white roundedg shadow-md border ${props.selected ? 'ring-2 ring-blue-500' : 'border-gray-200'} relative`}>
      {props.selected && (
        <div className="absolute -top-10 left-1/2 -translate-x-1/2 flex gap-2 bg-white shadow-lg br4 px-3 py-2 border z-10">
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
          <span className="material-symbols-outlined text-lg">{props.data.icon}</span>
        </span>
        <span className="font-bold text-gray-900 flex-1">{props.data.label}</span>
        {/* Example action button (menu) */}
        <button className="ml-2 p-1 rounded hover:bg-gray-200 transition" title="More actions">
          <span className="material-symbols-outlined text-gray-500">more_vert</span>
        </button>
      </div>
      <div className="p-4">
        <div className="flex justify-between items-center mb-3">
          <span className={`status-badge ${props.data.status ? props.data.status.toLowerCase() : ''}`}>{props.data.status}</span>
          <span className="text-xs text-gray-500">{props.data.timestamp}</span>
        </div>
        <div className="flex flex-wrap gap-1 mb-3">
          <span className="pipeline-badge bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2">
            code: {props.data.labels && props.data.labels[0] ? props.data.labels[0] : '—'}
          </span>
          <span className="pipeline-badge bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2">
            image: {props.data.labels && props.data.labels[1] ? props.data.labels[1] : '—'}
          </span>
          <span className="pipeline-badge bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2">
            terraform: {props.data.labels && props.data.labels[2] ? props.data.labels[2] : '—'}
          </span>
          <span className="pipeline-badge bg-gray-100 text-gray-700 text-xs px-2 py-1 rounded-full mr2">
            type: {props.data.labels && props.data.labels[3] ? props.data.labels[3] : '—'}
          </span>
        </div>
      </div>
      <div className="border-t border-gray-200 p-4">
        {/* PENDING Queue Section */}
        <h4 className="text-sm font-medium text-gray-700 mb-2">Pending Runs</h4>
        { pendingEvents.length > 0 ? (
          <>
            {/* Show the first pending item with details */}
            <div className="flex items-center p-2 bg-amber-50 rounded mb-1">
              <div className="material-symbols-outlined text-amber-600 mr-2">pending</div>
              <div className="flex-1">
                <div className="text-sm font-medium">{new Date(pendingEvents[0].created_at).toLocaleString()}</div>
                <div className="text-xs text-gray-600">ID: {pendingEvents[0].id.substring(0, 8)}...</div>
              </div>
            </div>
            {/* Show count of additional pending items */}
            {pendingEvents.length > 1 && (
              <div className="text-xs text-amber-600 hover:text-amber-800 mb-3">
                <a href="#" className="no-underline hover:underline">{pendingEvents.length - 1} more pending</a>
              </div>
            )}
          </>
        ) : (
          <div className="text-sm text-gray-500 italic mb-3">No pending items</div>
        )}
        
        {/* WAITING Queue Section */}
        {waitingEvents.length > 0 && (
          <>
            <h4 className="text-sm font-medium text-gray-700 mb-2 border-t pt-2">Waiting for Approval</h4>
            <div className="flex items-center p-2 bg-blue-50 rounded mb-1">
              <div className="material-symbols-outlined text-blue-600 mr-2">hourglass_empty</div>
              <div className="flex-1">
                <div className="text-sm font-medium">{new Date(waitingEvents[0].created_at).toLocaleString()}</div>
                <div className="text-xs text-gray-600">ID: {waitingEvents[0].id.substring(0, 8)}...</div>
              </div>
              <button 
                onClick={() => props.data.approve_stage_event(waitingEvents[0])}
                className="ml-2 inline-flex items-center px-2.5 py-1.5 border border-transparent text-xs font-medium rounded text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              >
                Approve
              </button>
            </div>
            {waitingEvents.length > 1 && (
              <div className="text-xs text-blue-600 hover:text-blue-800 mb-3">
                <a href="#" className="no-underline hover:underline">{waitingEvents.length - 1} more waiting</a>
              </div>
            )}
          </>
        )}
        
        {/* PROCESSED Queue Section - Only show the count */}
        {processedEvents.length > 0 && (
          <>
            <h4 className="text-sm font-medium text-gray-700 mb-2 border-t pt-2">Processed Recently</h4>
            <div className="flex items-center p-2 bg-green-50 rounded mb-1">
              <div className="material-symbols-outlined text-green-600 mr-2">check_circle</div>
              <div className="flex-1">
                <div className="text-sm">{processedEvents.length} processed</div>
                <div className="text-xs text-gray-600">Latest: {new Date(processedEvents[0].created_at).toLocaleString()}</div>
              </div>
            </div>
          </>
        )}
        
        {/* Show message when no queues exist */}
        {(!pendingEvents.length && !waitingEvents.length && !processedEvents.length) && (
          <div className="text-sm text-gray-500 italic">No queue activity</div>
        )}

      </div>
      <CustomBarHandle type="target" connections={props.data.connections} conditions={props.data.conditions}/>
      <CustomBarHandle type="source"/>
    </div>
  );
};

interface OverlayModalProps {
  open: boolean;
  onClose: () => void;
  children: ReactNode;
}

function OverlayModal({ open, onClose, children }: OverlayModalProps) {
  if (!open) return null;
  return (
    <div className="modal is-open" aria-hidden={!open} style={{position:'fixed',top:0,left:0,right:0,bottom:0,zIndex:999999}}>
      <div className="modal-overlay" style={{position:'fixed',top:0,left:0,right:0,bottom:0,background:'rgba(40,50,50,0.6)',zIndex:999999}} onClick={onClose} />
      <div className="modal-content" style={{position:'fixed',top:'50%',left:'50%',transform:'translate(-50%, -50%)',zIndex:1000000,background:'#fff',borderRadius:8,boxShadow:'0 6px 40px rgba(0,0,0,0.18)',maxWidth:600,width:'90vw',padding:32}}>
        <button onClick={onClose} style={{position:'absolute',top:8,right:12,background:'none',border:'none',fontSize:26,color:'#888',cursor:'pointer'}} aria-label="Close">×</button>
        {children}
      </div>
    </div>
  );
}
