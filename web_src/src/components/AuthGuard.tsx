import React, { useEffect, useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { User } from '../stores/userStore';

interface AuthGuardProps {
  children: React.ReactNode;
}

const AuthGuard: React.FC<AuthGuardProps> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    console.log('[AuthGuard] useEffect triggered, current path:', location.pathname);
    
    const fetchUser = async () => {
      console.log('[AuthGuard] Starting fetchUser');
      
      try {
        const response = await fetch('/api/v1/user/profile', {
          method: 'GET',
          credentials: 'include',
          headers: {
            'Content-Type': 'application/json',
          },
        });
        
        console.log('[AuthGuard] Profile response status:', response.status);
        
        if (!response.ok) {
          throw new Error(`Failed to fetch user: ${response.status}`);
        }
        
        const userData = await response.json();
        console.log('[AuthGuard] User data received:', userData);
        setUser(userData);
      } catch (error) {
        console.log('[AuthGuard] Error fetching user:', error);
        // User is not authenticated, redirect to login
        const currentPath = location.pathname + location.search;
        console.log('[AuthGuard] Redirecting to login from:', currentPath);
        navigate(`/login?redirect=${encodeURIComponent(currentPath)}`, { replace: true });
      } finally {
        console.log('[AuthGuard] Setting loading to false');
        setLoading(false);
      }
    };

    fetchUser();
  }, []); // Only run once on mount

  // Show loading spinner while fetching user
  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-zinc-900">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          <p className="text-sm text-gray-600 dark:text-gray-400">Loading...</p>
        </div>
      </div>
    );
  }

  // User is authenticated, render the protected content
  return <>{children}</>;
};

export default AuthGuard;