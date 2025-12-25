import { useState } from 'react';
import { LiveDashboard } from '@/components/dashboard/LiveDashboard';
import { LiveEventStream } from '@/components/realtime/LiveEventStream';
import { LiveProbeStatus } from '@/components/realtime/LiveProbeStatus';
import { NotificationSystem } from '@/components/realtime/NotificationSystem';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Activity, Bell, Radio, TrendingUp } from 'lucide-react';

export function RealtimeDemoPage() {
  // In a real app, this would come from authentication
  const [token] = useState('demo-token');
  const [enableLiveUpdates, setEnableLiveUpdates] = useState(true);

  return (
    <div className="container mx-auto p-6 space-y-6">
      {/* Page Header */}
      <div className="space-y-2">
        <h1 className="text-3xl font-bold tracking-tight">Real-time Monitoring</h1>
        <p className="text-gray-600 dark:text-gray-400">
          Live telemetry data with WebSocket-powered updates
        </p>
      </div>

      {/* Settings Toggle */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="font-semibold">Live Updates</h3>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                Enable real-time data streaming via WebSocket
              </p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={enableLiveUpdates}
                onChange={(e) => setEnableLiveUpdates(e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-blue-600"></div>
            </label>
          </div>
        </CardContent>
      </Card>

      {/* Main Content Tabs */}
      <Tabs defaultValue="dashboard" className="space-y-6">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="dashboard" className="flex items-center gap-2">
            <TrendingUp className="h-4 w-4" />
            Dashboard
          </TabsTrigger>
          <TabsTrigger value="events" className="flex items-center gap-2">
            <Activity className="h-4 w-4" />
            Event Stream
          </TabsTrigger>
          <TabsTrigger value="probes" className="flex items-center gap-2">
            <Radio className="h-4 w-4" />
            Probes
          </TabsTrigger>
          <TabsTrigger value="alerts" className="flex items-center gap-2">
            <Bell className="h-4 w-4" />
            Alerts
          </TabsTrigger>
        </TabsList>

        <TabsContent value="dashboard" className="space-y-6">
          <LiveDashboard token={token} enableLiveUpdates={enableLiveUpdates} />
          
          <Card>
            <CardHeader>
              <CardTitle>About Live Dashboard</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
              <p>
                The Live Dashboard displays real-time metrics updated via WebSocket connections.
                Key features include:
              </p>
              <ul className="list-disc list-inside space-y-1 ml-2">
                <li>Real-time metric updates without manual refresh</li>
                <li>Connection status indicator with auto-reconnect</li>
                <li>Live trend indicators showing performance changes</li>
                <li>Efficient data streaming with minimal network overhead</li>
                <li>Automatic fallback to polling if WebSocket unavailable</li>
              </ul>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="events" className="space-y-6">
          <LiveEventStream token={token} maxEvents={50} channels={['dashboard', 'diagnostics']} />
          
          <Card>
            <CardHeader>
              <CardTitle>About Event Stream</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
              <p>
                The Event Stream provides a real-time view of all telemetry events flowing through
                the system. Features include:
              </p>
              <ul className="list-disc list-inside space-y-1 ml-2">
                <li>Live event updates as they occur</li>
                <li>Filtering by event type (aggregates, diagnostics, alerts)</li>
                <li>Pause/resume functionality to inspect events</li>
                <li>Event buffer to prevent data loss during pause</li>
                <li>JSON preview of event data</li>
              </ul>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="probes" className="space-y-6">
          <LiveProbeStatus token={token} maxProbes={30} />
          
          <Card>
            <CardHeader>
              <CardTitle>About Probe Status</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
              <p>
                The Probe Status view shows the real-time health and activity of all monitoring
                probes. Features include:
              </p>
              <ul className="list-disc list-inside space-y-1 ml-2">
                <li>Live status updates for each probe</li>
                <li>Activity indicators (active, inactive, error, unknown)</li>
                <li>Last seen timestamps with relative time display</li>
                <li>Performance metrics (latency, error rate)</li>
                <li>Summary counts by status type</li>
              </ul>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="alerts" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Alert Configuration</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="p-4 rounded-lg bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800">
                <div className="flex items-start gap-3">
                  <Bell className="h-5 w-5 text-blue-500 mt-0.5" />
                  <div>
                    <h4 className="font-semibold text-blue-900 dark:text-blue-100">
                      Notifications Active
                    </h4>
                    <p className="text-sm text-blue-700 dark:text-blue-300 mt-1">
                      The notification system is monitoring for critical alerts. You'll be notified
                      when important events occur, such as:
                    </p>
                    <ul className="list-disc list-inside text-sm text-blue-700 dark:text-blue-300 mt-2 ml-2">
                      <li>Critical performance degradation</li>
                      <li>Probe failures or disconnections</li>
                      <li>High error rates detected</li>
                      <li>Diagnosis of network issues</li>
                    </ul>
                  </div>
                </div>
              </div>

              <div className="space-y-2">
                <h4 className="font-semibold">Browser Notifications</h4>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  For the best experience, enable browser notifications to receive alerts even when
                  this tab is not active. Notifications are only shown for critical and high-severity
                  events.
                </p>
                <button
                  onClick={() => {
                    if ('Notification' in window) {
                      Notification.requestPermission();
                    }
                  }}
                  className="px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-md text-sm font-medium"
                >
                  Enable Browser Notifications
                </button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Notification System (always rendered) */}
      <NotificationSystem token={token} position="top-right" maxNotifications={5} />
    </div>
  );
}
