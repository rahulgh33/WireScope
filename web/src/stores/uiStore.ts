// UI State Store (Zustand)

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { ThemeMode, ConnectionStatus } from '@/types/models';

interface UIState {
  // Sidebar
  sidebarOpen: boolean;
  setSidebarOpen: (open: boolean) => void;
  toggleSidebar: () => void;

  // Theme
  theme: ThemeMode;
  setTheme: (theme: ThemeMode) => void;
  toggleTheme: () => void;

  // Time Range
  selectedTimeRange: string;
  setTimeRange: (range: string) => void;

  // Filters
  selectedClients: string[];
  setSelectedClients: (clients: string[]) => void;
  toggleClient: (clientId: string) => void;
  
  selectedTargets: string[];
  setSelectedTargets: (targets: string[]) => void;
  toggleTarget: (target: string) => void;

  // WebSocket Status
  wsStatus: ConnectionStatus;
  setWSStatus: (status: ConnectionStatus) => void;
  lastEventTimestamp: string | null;
  setLastEventTimestamp: (timestamp: string) => void;

  // Search
  searchQuery: string;
  setSearchQuery: (query: string) => void;

  // Reset
  resetFilters: () => void;
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      // Sidebar
      sidebarOpen: true,
      setSidebarOpen: (open) => set({ sidebarOpen: open }),
      toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),

      // Theme
      theme: 'light',
      setTheme: (theme) => {
        document.documentElement.setAttribute('data-theme', theme);
        set({ theme });
      },
      toggleTheme: () =>
        set((state) => {
          const newTheme = state.theme === 'light' ? 'dark' : 'light';
          document.documentElement.setAttribute('data-theme', newTheme);
          return { theme: newTheme };
        }),

      // Time Range
      selectedTimeRange: '24h',
      setTimeRange: (range) => set({ selectedTimeRange: range }),

      // Filters
      selectedClients: [],
      setSelectedClients: (clients) => set({ selectedClients: clients }),
      toggleClient: (clientId) =>
        set((state) => ({
          selectedClients: state.selectedClients.includes(clientId)
            ? state.selectedClients.filter((id) => id !== clientId)
            : [...state.selectedClients, clientId],
        })),

      selectedTargets: [],
      setSelectedTargets: (targets) => set({ selectedTargets: targets }),
      toggleTarget: (target) =>
        set((state) => ({
          selectedTargets: state.selectedTargets.includes(target)
            ? state.selectedTargets.filter((t) => t !== target)
            : [...state.selectedTargets, target],
        })),

      // WebSocket Status
      wsStatus: 'disconnected',
      setWSStatus: (status) => set({ wsStatus: status }),
      lastEventTimestamp: null,
      setLastEventTimestamp: (timestamp) => set({ lastEventTimestamp: timestamp }),

      // Search
      searchQuery: '',
      setSearchQuery: (query) => set({ searchQuery: query }),

      // Reset
      resetFilters: () =>
        set({
          selectedClients: [],
          selectedTargets: [],
          searchQuery: '',
        }),
    }),
    {
      name: 'ui-storage',
      partialize: (state) => ({
        sidebarOpen: state.sidebarOpen,
        theme: state.theme,
        selectedTimeRange: state.selectedTimeRange,
      }),
    }
  )
);

// Initialize theme on load
const initialTheme = useUIStore.getState().theme;
document.documentElement.setAttribute('data-theme', initialTheme);
