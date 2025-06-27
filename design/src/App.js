import React from 'react';
import './index.css';
import './App.css';
import WorkflowEditor from './WorkflowEditor';

function App() {
  return (
    <div className="flex flex-col h-screen font-sans">
      <main className="flex-grow flex flex-col">
        <WorkflowEditor />
      </main>
    </div>
  );
}

export default App;
