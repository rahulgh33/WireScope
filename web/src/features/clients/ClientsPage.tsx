import { useState } from 'react';
import { useClients } from '@/hooks/useClients';
import { ClientCard } from '@/components/clients/ClientCard';
import { Input } from '@/components/ui/Input';
import { Button } from '@/components/ui/Button';
import { Skeleton } from '@/components/ui/Skeleton';

type SortOption = 'latency_desc' | 'latency_asc' | 'last_seen_desc' | 'name_asc';

export function ClientsPage() {
  const [search, setSearch] = useState('');
  const [sort, setSort] = useState<SortOption>('last_seen_desc');
  const [statusFilter, setStatusFilter] = useState<string[]>(['active']); // Default to only 'active'

  const { data, isLoading, error } = useClients({
    search: search || undefined,
    sort,
    limit: 50,
  });

  const filteredClients = data?.clients.filter((client) => {
    if (statusFilter.length === 0) return true;
    return statusFilter.includes(client.status);
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Clients</h1>
        <p className="text-muted-foreground mt-1">
          Monitor and manage connected clients and their performance
        </p>
      </div>

      {/* Filters and Search */}
      <div className="flex flex-col md:flex-row gap-4">
        <div className="flex-1">
          <Input
            type="text"
            placeholder="Search clients..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full"
          />
        </div>

        <div className="flex gap-2">
          <select
            value={sort}
            onChange={(e) => setSort(e.target.value as SortOption)}
            className="px-3 py-2 border border-border rounded-md bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary"
          >
            <option value="last_seen_desc">Last Seen (Newest)</option>
            <option value="latency_desc">Latency (Highest)</option>
            <option value="latency_asc">Latency (Lowest)</option>
            <option value="name_asc">Name (A-Z)</option>
          </select>
        </div>
      </div>

      {/* Status Filters - Radio button behavior */}
      <div className="flex gap-2 flex-wrap">
        <button
          onClick={() => setStatusFilter([])}
          className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
            statusFilter.length === 0
              ? 'bg-primary text-white'
              : 'bg-secondary text-foreground hover:bg-secondary/80'
          }`}
        >
          All Status
        </button>
        {(['active', 'inactive', 'warning'] as const).map((status) => (
          <button
            key={status}
            onClick={() => {
              // Toggle behavior: if already selected, deselect (show all)
              // Otherwise, select only this status
              setStatusFilter((prev) =>
                prev.includes(status) && prev.length === 1
                  ? []
                  : [status]
              );
            }}
            className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
              statusFilter.includes(status)
                ? 'bg-primary text-white'
                : 'bg-secondary text-foreground hover:bg-secondary/80'
            }`}
          >
            {status.charAt(0).toUpperCase() + status.slice(1)}
          </button>
        ))}
      </div>

      {/* Client List */}
      {error && (
        <div className="text-center py-12 text-muted-foreground">
          Error loading clients. Please try again.
        </div>
      )}

      {isLoading && (
        <div className="space-y-4">
          {[...Array(5)].map((_, i) => (
            <Skeleton key={i} className="h-32 w-full" />
          ))}
        </div>
      )}

      {!isLoading && !error && filteredClients && (
        <>
          {filteredClients.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              No clients found. Try adjusting your filters.
            </div>
          ) : (
            <div className="space-y-4">
              {filteredClients.map((client) => (
                <ClientCard key={client.client_id} client={client} />
              ))}
            </div>
          )}

          {data?.has_more && (
            <div className="flex justify-center pt-4">
              <Button variant="outline">Load More</Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
