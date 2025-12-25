import { useState, useEffect, useRef } from 'react';
import { useDynamicChannels } from '@/hooks/useLiveUpdates';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Badge } from '@/components/ui/badge';
import { Activity, AlertTriangle, CheckCircle, Clock, Filter, Pause, Play } from 'lucide-react';
import { cn } from '@/lib/utils';

interface Event {
  id: string;
  timestamp: Date;
  type: 'aggregate' | 'diagnosis' | 'probe_status' | 'alert';
  channel: string;
  data: any;
}

interface LiveEventStreamProps {
  token: string;
  maxEvents?: number;
  channels?: string[];
}

export function LiveEventStream({ token, maxEvents = 100, channels = ['dashboard'] }: LiveEventStreamProps) {
  const [events, setEvents] = useState<Event[]>([]);
  const [isPaused, setIsPaused] = useState(false);
  const [filter, setFilter] = useState<string | null>(null);
  const eventBufferRef = useRef<Event[]>([]);
  const scrollRef = useRef<HTMLDivElement>(null);

  const { connectionStatus, isConnected, addChannel, removeChannel } = useDynamicChannels(token);

  // Add custom event listener for WebSocket messages
  useEffect(() => {
    if (!isConnected || isPaused) return;

    const handleWSMessage = (event: CustomEvent) => {
      const message = event.detail;
      
      // Create event from WebSocket message
      const newEvent: Event = {
        id: `${Date.now()}-${Math.random()}`,
        timestamp: new Date(),
        type: detectEventType(message),
        channel: message.channel || 'unknown',
        data: message.data,
      };

      if (!isPaused) {
        setEvents((prev) => {
          const updated = [newEvent, ...prev];
          return updated.slice(0, maxEvents);
        });
      } else {
        eventBufferRef.current.push(newEvent);
      }
    };

    // Note: This would need to be implemented in the WebSocket client
    // window.addEventListener('ws:message', handleWSMessage as EventListener);

    return () => {
      // window.removeEventListener('ws:message', handleWSMessage as EventListener);
    };
  }, [isConnected, isPaused, maxEvents]);

  const togglePause = () => {
    if (isPaused) {
      // Flush buffered events
      setEvents((prev) => {
        const updated = [...eventBufferRef.current, ...prev];
        eventBufferRef.current = [];
        return updated.slice(0, maxEvents);
      });
    }
    setIsPaused(!isPaused);
  };

  const clearEvents = () => {
    setEvents([]);
    eventBufferRef.current = [];
  };

  const filteredEvents = filter
    ? events.filter((event) => event.type === filter)
    : events;

  return (
    <Card className="h-[600px] flex flex-col">
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <Activity className="h-5 w-5" />
            Live Event Stream
            <Badge variant={isConnected ? 'default' : 'secondary'}>
              {isConnected ? 'Connected' : 'Disconnected'}
            </Badge>
          </CardTitle>
          <div className="flex items-center gap-2">
            <button
              onClick={togglePause}
              className="inline-flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-md bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700"
            >
              {isPaused ? (
                <>
                  <Play className="h-4 w-4" />
                  Resume
                </>
              ) : (
                <>
                  <Pause className="h-4 w-4" />
                  Pause
                </>
              )}
            </button>
            <button
              onClick={clearEvents}
              className="px-3 py-2 text-sm font-medium rounded-md bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700"
            >
              Clear
            </button>
          </div>
        </div>
        
        {/* Filters */}
        <div className="flex items-center gap-2 mt-2">
          <Filter className="h-4 w-4 text-gray-500" />
          <button
            onClick={() => setFilter(null)}
            className={cn(
              'px-2 py-1 text-xs rounded-md',
              filter === null
                ? 'bg-blue-500 text-white'
                : 'bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700'
            )}
          >
            All
          </button>
          <button
            onClick={() => setFilter('aggregate')}
            className={cn(
              'px-2 py-1 text-xs rounded-md',
              filter === 'aggregate'
                ? 'bg-blue-500 text-white'
                : 'bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700'
            )}
          >
            Aggregates
          </button>
          <button
            onClick={() => setFilter('diagnosis')}
            className={cn(
              'px-2 py-1 text-xs rounded-md',
              filter === 'diagnosis'
                ? 'bg-blue-500 text-white'
                : 'bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700'
            )}
          >
            Diagnostics
          </button>
          <button
            onClick={() => setFilter('alert')}
            className={cn(
              'px-2 py-1 text-xs rounded-md',
              filter === 'alert'
                ? 'bg-blue-500 text-white'
                : 'bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700'
            )}
          >
            Alerts
          </button>
        </div>
      </CardHeader>
      
      <CardContent className="flex-1 overflow-hidden">
        <ScrollArea className="h-full" ref={scrollRef}>
          <div className="space-y-2">
            {filteredEvents.length === 0 ? (
              <div className="flex items-center justify-center h-32 text-gray-500">
                <div className="text-center">
                  <Clock className="h-8 w-8 mx-auto mb-2 opacity-50" />
                  <p>Waiting for events...</p>
                </div>
              </div>
            ) : (
              filteredEvents.map((event) => (
                <EventCard key={event.id} event={event} />
              ))
            )}
          </div>
        </ScrollArea>
      </CardContent>
    </Card>
  );
}

function EventCard({ event }: { event: Event }) {
  const getEventIcon = () => {
    switch (event.type) {
      case 'alert':
        return <AlertTriangle className="h-4 w-4 text-red-500" />;
      case 'diagnosis':
        return <Activity className="h-4 w-4 text-yellow-500" />;
      case 'aggregate':
        return <CheckCircle className="h-4 w-4 text-blue-500" />;
      default:
        return <Activity className="h-4 w-4 text-gray-500" />;
    }
  };

  const getEventBadgeColor = () => {
    switch (event.type) {
      case 'alert':
        return 'destructive';
      case 'diagnosis':
        return 'warning';
      case 'aggregate':
        return 'default';
      default:
        return 'secondary';
    }
  };

  return (
    <div className="flex items-start gap-3 p-3 rounded-lg border bg-card hover:bg-accent/50 transition-colors">
      <div className="mt-0.5">{getEventIcon()}</div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <Badge variant={getEventBadgeColor() as any} className="text-xs">
            {event.type}
          </Badge>
          <span className="text-xs text-gray-500">
            {event.timestamp.toLocaleTimeString()}
          </span>
        </div>
        <pre className="text-xs text-gray-600 dark:text-gray-400 overflow-x-auto">
          {JSON.stringify(event.data, null, 2)}
        </pre>
      </div>
    </div>
  );
}

function detectEventType(message: any): Event['type'] {
  if (message.data?.type === 'alert') return 'alert';
  if (message.data?.type === 'diagnosis') return 'diagnosis';
  if (message.data?.type === 'aggregate') return 'aggregate';
  if (message.data?.type === 'probe_status') return 'probe_status';
  return 'aggregate';
}
