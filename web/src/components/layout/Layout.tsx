// Main Layout Component

import { Header } from './Header';
import { Sidebar } from './Sidebar';
import { useUIStore } from '@/stores/uiStore';
import { cn } from '@/lib/utils';

interface LayoutProps {
  children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  const { sidebarOpen } = useUIStore();

  return (
    <div className="min-h-screen bg-background">
      <Header />
      <div className="flex">
        <Sidebar />
        <main
          className={cn(
            'flex-1 transition-all duration-300 pt-16',
            sidebarOpen ? 'md:ml-64' : 'md:ml-0'
          )}
        >
          <div className="container mx-auto p-6">
            {children}
          </div>
        </main>
      </div>
    </div>
  );
}
