import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/Skeleton';
import { useDiagnostics } from '@/hooks/useDiagnostics';
import { AlertCircle, CheckCircle, XCircle, Clock } from 'lucide-react';

export function DiagnosticsPage() {
  const { data, isLoading, error } = useDiagnostics();

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold">Diagnostics</h1>
          <p className="text-muted-foreground mt-1">
            Network performance diagnostics and analysis
          </p>
        </div>
        <div className="space-y-4">
          {[...Array(5)].map((_, i) => (
            <Skeleton key={i} className="h-32 w-full" />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold">Diagnostics</h1>
          <p className="text-muted-foreground mt-1">
            Network performance diagnostics and analysis
          </p>
        </div>
        <div className="text-center py-12 text-muted-foreground">
          Error loading diagnostics. Please try again.
        </div>
      </div>
    );
  }

  const getSeverityIcon = (severity: string) => {
    switch (severity) {
      case 'error':
        return <XCircle className="h-5 w-5 text-red-500" />;
      case 'warning':
        return <AlertCircle className="h-5 w-5 text-yellow-500" />;
      default:
        return <CheckCircle className="h-5 w-5 text-green-500" />;
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'error':
        return 'destructive';
      case 'warning':
        return 'warning';
      default:
        return 'default';
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Diagnostics</h1>
        <p className="text-muted-foreground mt-1">
          Network performance diagnostics and analysis
        </p>
      </div>

      {/* Diagnostics List */}
      <div className="space-y-4">
        {data?.diagnostics.map((diagnostic: any) => (
          <Card key={diagnostic.id}>
            <CardHeader>
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  {getSeverityIcon(diagnostic.severity)}
                  <div>
                    <CardTitle className="text-lg">
                      {diagnostic.client_id} @ {diagnostic.target}
                    </CardTitle>
                    <CardDescription className="flex items-center gap-2 mt-1">
                      <Clock className="h-4 w-4" />
                      {new Date(diagnostic.timestamp).toLocaleString()}
                    </CardDescription>
                  </div>
                </div>
                <Badge variant={getSeverityColor(diagnostic.severity) as any}>
                  {diagnostic.label}
                </Badge>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground mb-3">{diagnostic.description}</p>
              {diagnostic.metrics && (
                <div className="flex gap-4 text-sm">
                  {Object.entries(diagnostic.metrics).map(([key, value]) => (
                    <div key={key}>
                      <span className="text-muted-foreground">{key}: </span>
                      <span className="font-medium">{String(value)}</span>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      {data?.diagnostics.length === 0 && (
        <div className="text-center py-12 text-muted-foreground">
          No diagnostics available
        </div>
      )}
    </div>
  );
}
