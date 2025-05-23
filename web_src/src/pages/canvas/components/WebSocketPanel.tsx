import React, { useState } from 'react';

interface WebSocketPanelProps {
  status: 'connecting' | 'connected' | 'disconnected';
  messages: string[];
  onSendMessage: (message: string) => void;
  onClose: () => void;
}

export const WebSocketPanel: React.FC<WebSocketPanelProps> = ({ 
  status, 
  messages, 
  onSendMessage, 
  onClose 
}) => {
  const [newMessage, setNewMessage] = useState('');

  // Handle form submission for sending WebSocket messages
  const handleSendMessage = (e: React.FormEvent) => {
    e.preventDefault();
    if (newMessage.trim()) {
      onSendMessage(newMessage);
      setNewMessage('');
    }
  };

  // Get status indicator color
  const getStatusColor = () => {
    switch(status) {
      case 'connected': return 'bg-green-500';
      case 'connecting': return 'bg-yellow-500';
      case 'disconnected': return 'bg-red-500';
      default: return 'bg-gray-500';
    }
  };

  return (
    <div className="fixed bottom-16 right-4 w-96 bg-white rounded-lg shadow-xl border overflow-hidden">
      <div className="p-3 bg-gray-100 border-b flex justify-between items-center">
        <div className="flex items-center gap-2">
          <span className={`w-3 h-3 rounded-full ${getStatusColor()}`}></span>
          <h3 className="font-medium">WebSocket {status}</h3>
        </div>
        <button onClick={onClose} className="text-gray-500 hover:text-gray-700">
          <span className="material-symbols-outlined">close</span>
        </button>
      </div>
      
      {/* Messages container */}
      <div className="p-3 max-h-60 overflow-y-auto bg-gray-50" style={{ minHeight: '200px' }}>
        {messages.length === 0 ? (
          <p className="text-gray-500 text-center p-4">No messages yet</p>
        ) : (
          <ul className="space-y-2">
            {messages.map((msg, index) => (
              <li key={index} className="p-2 bg-white rounded border">
                {msg}
              </li>
            ))}
          </ul>
        )}
      </div>
      
      {/* Message input form */}
      <form onSubmit={handleSendMessage} className="p-3 border-t flex">
        <input
          type="text"
          value={newMessage}
          onChange={(e) => setNewMessage(e.target.value)}
          placeholder="Type a message to send..."
          className="flex-1 border rounded-l px-3 py-2"
        />
        <button 
          type="submit" 
          disabled={status !== 'connected'}
          className={`px-4 py-2 rounded-r ${status === 'connected' ? 'bg-blue-500 hover:bg-blue-600 text-white' : 'bg-gray-300 text-gray-500 cursor-not-allowed'}`}
        >
          Send
        </button>
      </form>
    </div>
  );
};

// This button can be used to toggle showing the WebSocket panel
export const WebSocketToggleButton: React.FC<{
  status: 'connecting' | 'connected' | 'disconnected';
  onClick: () => void;
}> = ({ status, onClick }) => {
  const getStatusColor = () => {
    switch(status) {
      case 'connected': return 'bg-green-500';
      case 'connecting': return 'bg-yellow-500';
      case 'disconnected': return 'bg-red-500';
      default: return 'bg-gray-500';
    }
  };

  return (
    <button 
      onClick={onClick}
      className="fixed bottom-4 right-4 flex items-center space-x-2 bg-white px-3 py-2 rounded-full shadow-lg border hover:bg-gray-50"
    >
      <span className={`w-3 h-3 rounded-full ${getStatusColor()}`}></span>
      <span>WebSocket {status}</span>
    </button>
  );
};
