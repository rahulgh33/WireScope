import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Skeleton } from '@/components/ui/Skeleton';
import { formatRelativeTime } from '@/lib/utils';

interface Diagnostic {
  id: string;
  timestamp: string;
  client_id: string;
  target: string;
  diagnosis_label: string;
  severity: 'critical' | 'warning' | 'info';
  root_cause: string;
}

// Mock data for now - in real app, this would come from diagnostics API
const mockDiagnostics: Diagnostic[] = [
  {
    id: '1',
    timestamp: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
    client_id: 'client-001',
    target: 'api.example.com',
    diagnosis_label: 'DNS-bound',
    severity: 'critical',
    root_cause: 'High DNS latency (450ms)',
  },
  {
    id: '2',
    timestamp: new Date(Date.now() - 15 * 60 * 1000).toISOString(),
    client_id: 'client-002',
    target: 'cdn.example.com',
    diagnosis_label: 'Server-bound',
    severity: 'warning',
    root_cause: 'Slow TTFB (280ms)',
  },
  {
    id: '3',
    timestamp: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
    client_id: 'client-003',
    target: 'api.example.com',
    diagnosis_label: 'Throughput',
    severity: 'warning',
    root_cause: 'Low throughput detected',
  },
];

export function RecentDiagnosticsPanel({ loading }: { loading?: boolean }) {
  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Recent Diagnostics</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {[...Array(3)].map((_, i) => (
              <Skeleton key={i} className="h-20 w-full" />
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Recent Diagnostics</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {mockDiagnostics.map((diagnostic) => (
            <DiagnosticCard key={diagnostic.id} diagnostic={diagnostic} />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

function DiagnosticCard({ diagnostic }: { diagnostic: Diagnostic }) {
  const severityColors = {
    critical: 'bg-red-500',
    warning: 'bg-yellow-500',
    info: 'bg-blue-500',
  };

  const severityBgColors = {
    critical: 'bg-red-50 dark:bg-red-900/10 border-red-200 dark:border-red-900',
    warning: 'bg-yellow-50 dark:bg-yellow-900/10 border-yellow-200 dark:border-yellow-900',
    info: 'bg-blue-50 dark:bg-blue-900/10 border-blue-200 dark:border-blue-900',
  };

  return (
    <div
      className={`p-3 rounded-lg border ${severityBgColors[diagnostic.severity]} transition-colors hover:shadow-sm`}
    >
      <div className="flex items-start gap-3">
        <div className={`w-2 h-2 mt-1.5 rounded-full ${severityColors[diagnostic.severity]}`} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between gap-2">
            <span className="font-medium text-sm">
              {diagnostic.client_id} @ {diagnostic.target}
            </span>
            <span className="text-xs text-muted-foreground whitespace-nowrap">
              {formatRelativeTime(diagnostic.timestamp)}
            </span>
          </div>
          <div className="mt-1">
            <span className="text-xs font-medium text-muted-foreground">
              {diagnostic.diagnosis_label}
            </span>
            <p className="text-sm mt-0.5">{diagnostic.root_cause}</p>
          </div>
        </div>
      </div>
    </div>
  );
}
