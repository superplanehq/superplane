import { create } from 'zustand';

export interface User {
  id: string;
  email: string;
  name: string;
  avatar_url: string;
  created_at: string;
  organization_id?: string;
  account_providers?: Array<{
    id: string;
    user_id: string;
    provider: string;
    provider_id: string;
    username: string;
    email: string;
    name: string;
    avatar_url: string;
    created_at: string;
  }>;
}

interface UserState {
  user: User | null;
  loading: boolean;
  error: string | null;
  
  // Actions
  fetchUser: () => Promise<void>;
  setUser: (user: User) => void;
  clearUser: () => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

export const useUserStore = create<UserState>((set, get) => ({
  user: null,
  loading: false,
  error: null,
  
  fetchUser: async () => {
    const { loading } = get();
    
    // Avoid multiple concurrent requests
    if (loading) return;
    
    set({ loading: true, error: null });
    
    try {
      const response = await fetch('/api/v1/user/profile', {
        method: 'GET',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error(`Failed to fetch user: ${response.status}`);
      }
      
      const userData = await response.json();
      set({ user: userData, loading: false });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to fetch user';
      set({ error: errorMessage, loading: false });
      throw error; // Re-throw so AuthGuard can catch it and redirect
    }
  },
  
  setUser: (user: User) => {
    set({ user, error: null });
  },
  
  clearUser: () => {
    set({ user: null, error: null });
  },
  
  setLoading: (loading: boolean) => {
    set({ loading });
  },
  
  setError: (error: string | null) => {
    set({ error });
  },
}));