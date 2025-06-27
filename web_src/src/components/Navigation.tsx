import React from 'react';
import semaphoreLogo from '../assets/semaphore-logo-sign-black.svg';

const Navigation: React.FC = () => {
  return (
    <div className="fixed top-0 left-0 right-0 z-50 bg-white shadow-sm">
      <div className="flex items-center justify-between px-4 md:px-6 py-2">
        <a href="#" className="flex items-center flex-shrink-0 text-decoration-none">
          <img src={semaphoreLogo} alt="Semaphore Logo" className="h-6" width={26} /> 
          <strong className="ml-2 text-lg text-gray-900">SuperPlane</strong>
        </a>
        <div className="flex items-center flex-shrink-0">
          <div className="flex-shrink-0 p-1 m-1 cursor-pointer bg-gray-100 hover:bg-gray-200 rounded-full">
            <div className="w-6 h-6 rounded-full border border-gray-400 bg-gray-200"></div>    
          </div>
        </div>
      </div>
    </div>
  );
};

export default Navigation;