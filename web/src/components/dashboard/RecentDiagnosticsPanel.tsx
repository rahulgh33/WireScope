import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Skeleton } from '@/components/ui/Skeleton';
import { formatRelativeTime } from '@/lib/utils';
import { useDiagnostics } from '@/hooks/useDiagnostics';

export function RecentDiagnosticsPanel({ loading }: { loading?: boolean }) {
  const { data: diagnosticsData, isLoading: diagnosticsLoading, error } = useDiagnostics();

  if (loading || diagnosticsLoading) {
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

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Recent Diagnostics</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center h-32 text-muted-foreground">
            Error loading diagnostics
          </div>
        </CardContent>
      </Card>
    );
  }

  const diagnostics = diagnosticsData?.diagnostics?.slice(0, 5) || [];

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'error': return 'text-red-600';
      case 'warning': return 'text-yellow-600';
      default: return 'text-blue-600';
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Recent Diagnostics</CardTitle>
      </CardHeader>
      <CardContent>
        {diagnostics.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            No recent diagnostics
          </div>
        ) : (
          <div className="space-y-4">
            {diagnostics.map((diagnostic: any) => (
              <div
                key={diagnostic.id}
                className="flex flex-col space-y-2 p-4 rounded-lg border border-border"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-2">
                    <span className="font-medium">{diagnostic.client_id}</span>
                    <span className="text-muted-foreground">@</span>
                    <span className="font-mono text-sm">{diagnostic.target}</span>
                  </div>
                  <span className="text-sm text-muted-foreground">
                    {formatRelativeTime(diagnostic.timestamp)}
                  </span>
                </div>
                <div className="flex items-center space-x-2">
                  <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                    diagnostic.severity === 'error' ? 'bg-red-100 text-red-800' :
                    diagnostic.severity === 'warning' ? 'bg-yellow-100 text-yellow-800' :
                    'bg-blue-100 text-blue-800'
                  }`}>
                    {diagnostic.label}
                  </span>
                </div>
                <p className="text-sm text-muted-foreground">
                  {diagnostic.description}
                </p>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
