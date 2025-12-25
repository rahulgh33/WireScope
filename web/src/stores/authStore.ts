// Authentication Store (Zustand)

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User } from '@/types/api';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  csrfToken: string | null;
  setUser: (user: User | null) => void;
  setCSRFToken: (token: string | null) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      isAuthenticated: false,
      csrfToken: null,
      setUser: (user) => set({ user, isAuthenticated: !!user }),
      setCSRFToken: (csrfToken) => set({ csrfToken }),
      logout: () => set({ user: null, isAuthenticated: false, csrfToken: null }),
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
        // Don't persist CSRF token (should come from server)
      }),
    }
  )
);
