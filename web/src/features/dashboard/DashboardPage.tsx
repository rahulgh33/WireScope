import { MetricCards } from '@/components/dashboard/MetricCards';
import { LatencyTrendsChart } from '@/components/dashboard/LatencyTrendsChart';
import { TopIssuesPanel } from '@/components/dashboard/TopIssuesPanel';
import { RecentDiagnosticsPanel } from '@/components/dashboard/RecentDiagnosticsPanel';

export function DashboardPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Dashboard</h1>
        <p className="text-muted-foreground mt-1">
          Overview of your network telemetry and performance metrics
        </p>
      </div>

      {/* Metric Cards */}
      <MetricCards />

      {/* Main Chart */}
      <LatencyTrendsChart />

      {/* Bottom Row */}
      <div className="grid gap-4 md:grid-cols-2">
        <TopIssuesPanel />
        <RecentDiagnosticsPanel />
      </div>
    </div>
  );
}
