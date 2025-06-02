
import { useEffect, useState } from 'react';
import { Panel } from '@xyflow/react';
import { useWebSocketConnection } from '../hooks/useWebSocketEvent';

// Simple cn utility since we can't access @/lib/utils
type ClassValue = string | boolean | undefined | null;
const cn = (...classes: ClassValue[]) => {
  return classes.filter(Boolean).join(' ');
};

export const ConnectionStatus: React.FC = () => {
  const [isVisible, setIsVisible] = useState(false);
  const [autoHideTimer, setAutoHideTimer] = useState<NodeJS.Timeout | null>(null);
  const { connectionStatus } = useWebSocketConnection(''); // Pass the actual canvasId here

  // Handle auto-hide for connected state
  useEffect(() => {
    if (connectionStatus === 'connected') {
      setIsVisible(true);
      
      // Auto-hide after 3 seconds for connected state
      const timer = setTimeout(() => setIsVisible(false), 3000);
      setAutoHideTimer(timer);
      
      return () => {
        clearTimeout(timer);
      };
    } else if (connectionStatus === 'disconnected') {
      setIsVisible(true);
    }
  }, [connectionStatus]);
  
  // Clean up timer on unmount
  useEffect(() => {
    return () => {
      if (autoHideTimer) {
        clearTimeout(autoHideTimer);
      }
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const statusConfig = {
    connected: {
      text: 'Connected',
      icon: 'check_circle',
      className: 'text-green-500',
    },
    disconnected: {
      text: 'Disconnected',
      icon: 'error',
      className: 'text-red-500',
    },
    connecting: {
      text: 'Connecting...',
      icon: 'refresh',
      className: 'text-yellow-500 animate-spin',
    },
  }[connectionStatus === 'connected' ? 'connected' : 'disconnected'];

  if (!isVisible) return null;

  return (
    <Panel position="bottom-right" className="bg-white/80 backdrop-blur-sm p-2 rounded-md shadow-lg">
      <div className="flex items-center gap-2">
        <span 
          className={cn(
            'material-icons text-lg',
            statusConfig.className
          )}
          style={{ fontSize: '1.25rem' }}
        >
          {statusConfig.icon}
        </span>
        <span className="text-sm text-gray-700">
          {statusConfig.text}
        </span>
      </div>
    </Panel>
  );
};