import { useState } from 'react';
import { useTargets } from '@/hooks/useTargets';
import { TargetCard } from '@/components/targets/TargetCard';
import { Skeleton } from '@/components/ui/Skeleton';

type HealthFilter = 'all' | 'healthy' | 'warning' | 'critical';
type SortOption = 'clients_desc' | 'latency_desc' | 'latency_asc' | 'name_asc';

export function TargetsPage() {
  const [health, setHealth] = useState<HealthFilter>('all');
  const [sort, setSort] = useState<SortOption>('clients_desc');

  const { data, isLoading, error } = useTargets({ health, sort });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Targets</h1>
        <p className="text-muted-foreground mt-1">
          Monitor target endpoints and their health status
        </p>
      </div>

      {/* Filters */}
      <div className="flex flex-col md:flex-row gap-4 items-start md:items-center justify-between">
        {/* Health Filters */}
        <div className="flex gap-2 flex-wrap">
          {(['all', 'healthy', 'warning', 'critical'] as const).map((filterOption) => (
            <button
              key={filterOption}
              onClick={() => setHealth(filterOption)}
              className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                health === filterOption
                  ? 'bg-primary text-white'
                  : 'bg-secondary text-foreground hover:bg-secondary/80'
              }`}
            >
              {filterOption.charAt(0).toUpperCase() + filterOption.slice(1)}
            </button>
          ))}
        </div>

        {/* Sort Dropdown */}
        <select
          value={sort}
          onChange={(e) => setSort(e.target.value as SortOption)}
          className="px-3 py-2 border border-border rounded-md bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary"
        >
          <option value="clients_desc">Most Clients</option>
          <option value="latency_desc">Highest Latency</option>
          <option value="latency_asc">Lowest Latency</option>
          <option value="name_asc">Name (A-Z)</option>
        </select>
      </div>

      {/* Target List */}
      {error && (
        <div className="text-center py-12 text-muted-foreground">
          Error loading targets. Please try again.
        </div>
      )}

      {isLoading && (
        <div className="space-y-4">
          {[...Array(5)].map((_, i) => (
            <Skeleton key={i} className="h-36 w-full" />
          ))}
        </div>
      )}

      {!isLoading && !error && data?.targets && (
        <>
          {data.targets.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              No targets found. Try adjusting your filters.
            </div>
          ) : (
            <div className="space-y-4">
              {data.targets.map((target) => (
                <TargetCard key={target.target} target={target} />
              ))}
            </div>
          )}
        </>
      )}

      {!isLoading && data?.targets && data.targets.length > 0 && (
        <div className="text-center text-sm text-muted-foreground">
          Showing {data.targets.length} target{data.targets.length !== 1 ? 's' : ''}
        </div>
      )}
    </div>
  );
}
