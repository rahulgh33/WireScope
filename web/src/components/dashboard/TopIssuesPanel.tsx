import { PieChart, Pie, Cell, ResponsiveContainer, Legend, Tooltip } from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Skeleton } from '@/components/ui/Skeleton';
import { useDashboardOverview } from '@/hooks/useDashboard';

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
  const { isLoading, error } = useDashboardOverview();

  // Mock data for now - in real app, this would come from diagnostics API
  const mockIssues: IssueData[] = [
    { name: 'DNS-bound', value: 45, color: COLORS[0] },
    { name: 'Server-bound', value: 30, color: COLORS[1] },
    { name: 'Throughput', value: 15, color: COLORS[2] },
    { name: 'Handshake', value: 10, color: COLORS[3] },
  ];

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
          <Skeleton className="h-64 w-full" />
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
        <ResponsiveContainer width="100%" height={250}>
          <PieChart>
            <Pie
              data={mockIssues}
              cx="50%"
              cy="50%"
              labelLine={false}
              label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
              outerRadius={80}
              fill="#8884d8"
              dataKey="value"
            >
              {mockIssues.map((entry, index) => (
                <Cell key={`cell-${index}`} fill={entry.color} />
              ))}
            </Pie>
            <Tooltip />
            <Legend />
          </PieChart>
        </ResponsiveContainer>
        <div className="mt-4 space-y-2">
          {mockIssues.map((issue) => (
            <div key={issue.name} className="flex items-center justify-between text-sm">
              <div className="flex items-center gap-2">
                <div
                  className="w-3 h-3 rounded-full"
                  style={{ backgroundColor: issue.color }}
                />
                <span>{issue.name}</span>
              </div>
              <span className="font-medium">{issue.value}%</span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
