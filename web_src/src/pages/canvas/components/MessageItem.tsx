import React, { useRef, useState } from 'react';
import { SuperplaneStageEvent, SuperplaneInputValue } from '@/api-client';
import { formatRelativeTime } from '../utils/stageEventUtils';

interface MessageItemProps {
  event: SuperplaneStageEvent;
  onApprove?: (eventId: string) => void;
  onRemove?: (eventId: string) => void;
  approved?: boolean;
  isDragStart?: boolean;
  executionRunning?: boolean;
}

const MessageItem = React.memo(({ 
  event, 
  onApprove, 
  onRemove, 
  approved = false, 
  isDragStart = false,
  executionRunning = false 
}: MessageItemProps) => {
  const [isExpanded, setIsExpanded] = React.useState(false);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const toggleExpand = () => {
    setIsExpanded(!isExpanded);
  };

  const handleDropdownClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    setIsDropdownOpen(!isDropdownOpen);
  };

  const handleRemove = () => {
    if (onRemove && event.id) {
      onRemove(event.id);
    }
    setIsDropdownOpen(false);
  };

  const handleApprove = () => {
    if (onApprove && event.id && !executionRunning) {
      onApprove(event.id);
    }
  };

  React.useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, []);

  // Get commit hash and image version from inputs if available
  const getInputValue = (name: string): string => {
    const input = event.inputs?.find(input => input.name?.toLowerCase().includes(name.toLowerCase()));
    return input?.value || '—';
  };

  const commitHash = getInputValue('code') || getInputValue('commit') || getInputValue('hash');
  const imageVersion = getInputValue('image') || getInputValue('version');

  return (
    <div className="run-item flex items-start mv1 pa2 ba br2 bg-white w-full" style={{
      '--mv1': '0.5rem 0',
      '--pa2': '0.5rem',
      '--ba': '1px solid',
      '--br2': '0.25rem',
      '--mr1': '0.25rem',
      '--mr2': '0.5rem',
      '--pt1': '0.25rem 0 0 0',
      '--pt2': '0.5rem 0 0 0',
      '--mt1': '0.25rem 0 0 0',
      '--pl2': '0 0 0 0.5rem',
      '--mb1': '0 0 0.25rem 0',
      '--f1': '3rem',
      '--f6': '0.875rem'
    } as React.CSSProperties}>
      <button 
        className="btn btn-outline btn-small"
        onClick={toggleExpand}
        title={isExpanded ? "Hide details" : "Show details"}
        style={{
          padding: '0',
          marginRight: 'var(--mr1)',
          lineHeight: 'none',
          background: 'transparent',
          border: '1px solid var(--lighter-gray)',
          borderRadius: '0.125rem'
        }}
      >
        <span className="material-symbols-outlined">{isExpanded ? 'arrow_drop_down' : 'arrow_right'}</span>
      </button>
      
      <div className='w-full'>
        <div className="flex justify-between">
          <div className="flex items-center">
            <span 
              className="material-symbols-outlined mr1"
              style={{ 
                color: 'var(--orange)', 
                fontSize: 'var(--f1)',
                marginRight: 'var(--mr1)'
              }}
            >
              input
            </span>
            <a href="#" className="truncate b flex" style={{ fontWeight: 'bold' }}>
              Event #{event.id?.substring(0, 8) || 'Unknown'}
            </a>
          </div>
          <div className="flex items-center">
            <div 
              className="text-xs tr inline-block"
              style={{ 
                color: 'var(--gray)',
                textAlign: 'right',
                marginLeft: '1rem',
                marginRight: 'var(--mr2)'
              }}
            >
              {event.createdAt ? formatRelativeTime(event.createdAt) : 'Unknown time'}
            </div>
          </div>
        </div>
        
        <div className="w-full">
          {/* Collapsed view - show key inputs */}
          {!isExpanded && event.inputs && event.inputs.length > 0 && (
            <div className="flex items-center" style={{ paddingTop: 'var(--pt1)' }}>
              <span 
                className="text-xs px-1 py-1 mr2 leading-none ba code"
                style={{
                  backgroundColor: 'var(--black-05)',
                  color: 'var(--gray)',
                  borderColor: 'var(--black-05)',
                  borderRadius: '0.25rem',
                  marginRight: 'var(--mr2)',
                  fontFamily: 'monospace'
                }}
              >
                code: {commitHash}
              </span>
              <span 
                className="text-xs px-1 py-1 mr2 leading-none ba code"
                style={{
                  backgroundColor: 'var(--black-10)',
                  color: 'var(--black)',
                  borderColor: 'var(--black-10)',
                  borderRadius: '0.25rem',
                  marginRight: 'var(--mr2)',
                  fontFamily: 'monospace'
                }}
              >
                image: {imageVersion}
              </span>
              {event.inputs.length > 2 && (
                <span className="text-xs px-2 py-1" style={{ marginRight: 'var(--mr2)' }}>
                  +{event.inputs.length - 2} more
                </span>
              )}
            </div>
          )}
          
          {/* Expanded view */}
          {isExpanded && event.inputs && event.inputs.length > 0 && (
            <div style={{ paddingTop: 'var(--pt2)' }}>
              <div className="flex items-center" style={{ marginBottom: 'var(--mb1)' }}>
                <span 
                  className="material-symbols-outlined text-sm mr1"
                  style={{ 
                    color: 'var(--gray)',
                    marginRight: 'var(--mr1)'
                  }}
                >
                  nest_clock_farsight_analog
                </span>
                {event.createdAt ? new Date(event.createdAt).toLocaleString() : 'Unknown date'}
              </div>
              
              <div className="flex justify-between">
                <div className='w-1/2'>
                  <div className="flex items-start"> 
                    <span 
                      className="material-symbols-outlined text-sm"
                      style={{ marginRight: 'var(--mr1)' }}
                    >
                      input
                    </span>
                    <div className="text-sm">
                      <div className='ttu' style={{ marginBottom: 'var(--mb1)', textTransform: 'uppercase' }}>
                        Inputs
                      </div>
                      <div className="flex items-center code text-xs" style={{ fontFamily: 'monospace' }}>
                        <div style={{ color: 'var(--gray)' }}>
                          {event.inputs.map((input, index) => (
                            <div 
                              key={index} 
                              style={{ 
                                backgroundColor: index % 2 === 1 ? 'var(--black-05)' : 'transparent' 
                              }}
                            >
                              {input.name}
                            </div>
                          ))}
                        </div>
                        <div style={{ paddingLeft: 'var(--pl2)' }}>
                          {event.inputs.map((input, index) => (
                            <div 
                              key={index} 
                              style={{ 
                                backgroundColor: index % 2 === 1 ? 'var(--black-05)' : 'transparent',
                                paddingLeft: 'var(--pl2)'
                              }}
                            >
                              {input.value}
                            </div>
                          ))}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
        
        {/* Bottom section with approval */}
        <div 
          className="flex items-center justify-between bt"
          style={{
            marginTop: 'var(--mt1)',
            borderTop: '1px solid var(--black-075)',
            paddingTop: 'var(--pt2)'
          }}
        >
          <div className="flex items-center text-xs">
            <span 
              className="material-symbols-outlined mr1"
              style={{ 
                color: 'var(--gray)',
                fontSize: 'var(--f6)',
                marginRight: 'var(--mr1)'
              }}
            >
              schedule
            </span>
            {event.state === 'STATE_WAITING' ? 'Waiting for approval' : 'Ready to run'}
          </div>
          
          {event.state === 'STATE_WAITING' && event.approvals && event.approvals.length > 0 && (
            <div className="flex items-center text-xs">
              <span 
                className="material-symbols-outlined"
                style={{ 
                  color: 'var(--gray)',
                  fontSize: 'var(--f6)'
                }}
              >
                check_circle
              </span>
              <div style={{ marginLeft: 'var(--mr1)' }}>
                approved by <a href="#" style={{ color: 'var(--black)', textDecoration: 'underline' }}>
                  {event.approvals.length} person{event.approvals.length !== 1 ? 's' : ''}
                </a>
                {event.approvals.length < 3 && ', waiting for more'}
              </div>
            </div>
          )}
          
          <div className="flex items-center">
            {event.state === 'STATE_WAITING' && onApprove && (
              <button 
                onClick={handleApprove}
                disabled={executionRunning || approved}
                className="btn btn-secondary btn-small"
                style={{
                  backgroundColor: approved ? 'var(--lightest-green)' : '',
                  borderColor: approved ? 'var(--washed-green)' : '',
                  color: approved ? 'var(--dark-green)' : '',
                  pointerEvents: approved ? 'none' : 'auto'
                }}
              >
                <span className="material-symbols-outlined text-sm">check</span>
              </button>
            )}
            
            <div className="relative" ref={dropdownRef}>
              <button 
                className="more-options btn btn-link btn-small"
                onClick={handleDropdownClick}
                style={{ padding: '0' }}
              >
                <span 
                  className="material-symbols-outlined"
                  style={{ fontSize: '1.125rem' }}
                >
                  more_vert
                </span>
              </button>
              {isDropdownOpen && (
                <div 
                  className="absolute right-0 bg-white shadow-lg rounded-lg w-32 z-10"
                  style={{ marginTop: 'var(--mt1)' }}
                >
                  <div className="py-1">
                    <button 
                      onClick={handleRemove}
                      className="block w-full text-left px-4 py-2 hover:bg-gray-100"
                    >
                      Remove
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
});

export default MessageItem;