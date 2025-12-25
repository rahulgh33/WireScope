import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';

interface CommonIssuesPanelProps {
  issues: string[];
}

export function CommonIssuesPanel({ issues }: CommonIssuesPanelProps) {
  if (!issues || issues.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Common Issues</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-green-600 font-medium">
            No issues detected - Target is healthy
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Common Issues</CardTitle>
        <p className="text-sm text-muted-foreground mt-1">
          Frequently occurring problems for this target
        </p>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {issues.map((issue, index) => (
            <div
              key={index}
              className="flex items-start gap-3 p-3 rounded-lg border border-red-200 bg-red-50 dark:border-red-900 dark:bg-red-900/10"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="2"
                className="h-5 w-5 text-red-600 flex-shrink-0 mt-0.5"
              >
                <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
                <line x1="12" y1="9" x2="12" y2="13" />
                <line x1="12" y1="17" x2="12.01" y2="17" />
              </svg>
              <div className="flex-1">
                <p className="text-sm font-medium text-red-900 dark:text-red-100">{issue}</p>
              </div>
            </div>
          ))}
        </div>

        <div className="mt-4 p-3 rounded-lg bg-blue-50 dark:bg-blue-900/10 border border-blue-200 dark:border-blue-900">
          <div className="flex items-start gap-2">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth="2"
              className="h-5 w-5 text-blue-600 flex-shrink-0 mt-0.5"
            >
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="16" x2="12" y2="12" />
              <line x1="12" y1="8" x2="12.01" y2="8" />
            </svg>
            <div className="text-sm text-blue-900 dark:text-blue-100">
              <p className="font-medium">Recommendation</p>
              <p className="mt-1">
                Review the per-client performance data above to identify which clients are
                experiencing these issues. Consider investigating network conditions or server
                configuration.
              </p>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
