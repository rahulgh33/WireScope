import { PieChart, Pie, Cell, ResponsiveContainer, Legend, Tooltip } from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Skeleton } from '@/components/ui/Skeleton';
import { useDiagnostics } from '@/hooks/useDiagnostics';

const COLORS = [
  '#ef4444', // red
  '#f97316', // orange
  '#eab308', // yellow
  '#22c55e', // green
  '#3b82f6', // blue
  '#8b5cf6', // purple
];

interface IssueData {
  name: string;
  value: number;
  color: string;
}

export function TopIssuesPanel() {
  const { data: diagnosticsData, isLoading, error } = useDiagnostics();

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Top Issues</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center h-64 text-muted-foreground">
            Error loading issues
          </div>
        </CardContent>
      </Card>
    );
  }

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Top Issues</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <Skeleton className="h-64 w-full" />
          </div>
        </CardContent>
      </Card>
    );
  }

  // Calculate issue distribution from real diagnostics data
  const diagnostics = diagnosticsData?.diagnostics || [];
  const issueCounts: Record<string, number> = {};
  
  diagnostics.forEach((diag: any) => {
    const label = diag.label || 'Unknown';
    issueCounts[label] = (issueCounts[label] || 0) + 1;
  });

  const issues: IssueData[] = Object.entries(issueCounts)
    .map(([name, count], index) => ({
      name,
      value: count,
      color: COLORS[index % COLORS.length],
    }))
    .sort((a, b) => b.value - a.value)
    .slice(0, 6); // Top 6 issues

  if (issues.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Top Issues</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center h-64 text-muted-foreground">
            No issues detected
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Top Issues</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="h-64">
          <ResponsiveContainer width="100%" height="100%">
            <PieChart>
              <Pie
                data={issues}
                cx="50%"
                cy="50%"
                innerRadius={60}
                outerRadius={100}
                paddingAngle={5}
                dataKey="value"
              >
                {issues.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={entry.color} />
                ))}
              </Pie>
              <Tooltip 
                formatter={(value, name) => [`${value}`, name]}
                contentStyle={{
                  backgroundColor: 'hsl(var(--background))',
                  border: '1px solid hsl(var(--border))',
                  borderRadius: '6px',
                }}
              />
              <Legend 
                verticalAlign="bottom" 
                height={36}
                wrapperStyle={{
                  paddingTop: '20px',
                }}
                formatter={(value) => `${value}`}
              />
            </PieChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  );
}
