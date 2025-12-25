import { useState, useEffect, useCallback } from 'react';
import { useLiveUpdates } from '@/hooks/useLiveUpdates';
import { AlertCircle, CheckCircle, Info, X, Bell, BellOff } from 'lucide-react';
import { cn } from '@/lib/utils';

export interface Notification {
  id: string;
  type: 'success' | 'warning' | 'error' | 'info';
  title: string;
  message: string;
  timestamp: Date;
  action?: {
    label: string;
    onClick: () => void;
  };
  autoDismiss?: boolean;
  duration?: number; // milliseconds
}

interface NotificationSystemProps {
  token: string;
  position?: 'top-right' | 'top-left' | 'bottom-right' | 'bottom-left';
  maxNotifications?: number;
}

export function NotificationSystem({ 
  token, 
  position = 'top-right',
  maxNotifications = 5,
}: NotificationSystemProps) {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [isEnabled, setIsEnabled] = useState(true);
  const { isConnected } = useLiveUpdates({
    token,
    channels: ['dashboard', 'diagnostics'],
    enabled: isEnabled,
  });

  // Add notification
  const addNotification = useCallback((notification: Omit<Notification, 'id' | 'timestamp'>) => {
    if (!isEnabled) return;

    const newNotification: Notification = {
      ...notification,
      id: `notification-${Date.now()}-${Math.random()}`,
      timestamp: new Date(),
      autoDismiss: notification.autoDismiss ?? true,
      duration: notification.duration ?? 5000,
    };

    setNotifications((prev) => {
      const updated = [newNotification, ...prev];
      return updated.slice(0, maxNotifications);
    });

    // Auto-dismiss if enabled
    if (newNotification.autoDismiss) {
      setTimeout(() => {
        dismissNotification(newNotification.id);
      }, newNotification.duration);
    }

    // Request browser notification permission
    if ('Notification' in window && Notification.permission === 'granted') {
      new Notification(newNotification.title, {
        body: newNotification.message,
        icon: '/logo.png',
        badge: '/logo.png',
      });
    }
  }, [isEnabled, maxNotifications]);

  // Dismiss notification
  const dismissNotification = useCallback((id: string) => {
    setNotifications((prev) => prev.filter((n) => n.id !== id));
  }, []);

  // Clear all notifications
  const clearAll = () => {
    setNotifications([]);
  };

  // Toggle notifications on/off
  const toggleNotifications = () => {
    setIsEnabled(!isEnabled);
    if (isEnabled) {
      clearAll();
    }
  };

  // Request browser notification permission
  useEffect(() => {
    if ('Notification' in window && Notification.permission === 'default') {
      Notification.requestPermission();
    }
  }, []);

  // Listen for WebSocket events and create notifications
  // This would need to be integrated with the WebSocket message handler
  useEffect(() => {
    // Example: Listen for critical alerts
    const handleCriticalAlert = (event: CustomEvent) => {
      const data = event.detail;
      
      if (data.severity === 'critical' || data.severity === 'high') {
        addNotification({
          type: 'error',
          title: 'Critical Alert',
          message: data.message || 'A critical issue has been detected',
          autoDismiss: false,
        });
      }
    };

    // window.addEventListener('ws:alert', handleCriticalAlert as EventListener);

    return () => {
      // window.removeEventListener('ws:alert', handleCriticalAlert as EventListener);
    };
  }, [addNotification]);

  const positionClasses = {
    'top-right': 'top-4 right-4',
    'top-left': 'top-4 left-4',
    'bottom-right': 'bottom-4 right-4',
    'bottom-left': 'bottom-4 left-4',
  };

  return (
    <>
      {/* Notification toggle button */}
      <button
        onClick={toggleNotifications}
        className={cn(
          'fixed z-50 p-3 rounded-full shadow-lg transition-colors',
          position.includes('right') ? 'right-4' : 'left-4',
          position.includes('top') ? 'top-20' : 'bottom-20',
          isEnabled
            ? 'bg-blue-500 hover:bg-blue-600 text-white'
            : 'bg-gray-500 hover:bg-gray-600 text-white'
        )}
        title={isEnabled ? 'Disable notifications' : 'Enable notifications'}
      >
        {isEnabled ? <Bell className="h-5 w-5" /> : <BellOff className="h-5 w-5" />}
        {notifications.length > 0 && (
          <span className="absolute -top-1 -right-1 inline-flex h-5 w-5 items-center justify-center rounded-full bg-red-500 text-xs text-white">
            {notifications.length}
          </span>
        )}
      </button>

      {/* Notifications container */}
      <div className={cn('fixed z-50 flex flex-col gap-2 w-96 max-w-full', positionClasses[position])}>
        {notifications.map((notification) => (
          <NotificationCard
            key={notification.id}
            notification={notification}
            onDismiss={dismissNotification}
          />
        ))}
      </div>
    </>
  );
}

interface NotificationCardProps {
  notification: Notification;
  onDismiss: (id: string) => void;
}

function NotificationCard({ notification, onDismiss }: NotificationCardProps) {
  const getConfig = () => {
    switch (notification.type) {
      case 'success':
        return {
          icon: CheckCircle,
          bgColor: 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800',
          iconColor: 'text-green-500',
          titleColor: 'text-green-900 dark:text-green-100',
        };
      case 'warning':
        return {
          icon: AlertCircle,
          bgColor: 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800',
          iconColor: 'text-yellow-500',
          titleColor: 'text-yellow-900 dark:text-yellow-100',
        };
      case 'error':
        return {
          icon: AlertCircle,
          bgColor: 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800',
          iconColor: 'text-red-500',
          titleColor: 'text-red-900 dark:text-red-100',
        };
      case 'info':
      default:
        return {
          icon: Info,
          bgColor: 'bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800',
          iconColor: 'text-blue-500',
          titleColor: 'text-blue-900 dark:text-blue-100',
        };
    }
  };

  const config = getConfig();
  const Icon = config.icon;

  return (
    <div
      className={cn(
        'relative flex gap-3 p-4 rounded-lg border shadow-lg animate-in slide-in-from-right-4',
        config.bgColor
      )}
    >
      <Icon className={cn('h-5 w-5 flex-shrink-0 mt-0.5', config.iconColor)} />
      
      <div className="flex-1 min-w-0">
        <h4 className={cn('font-semibold text-sm', config.titleColor)}>
          {notification.title}
        </h4>
        <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
          {notification.message}
        </p>
        
        {notification.action && (
          <button
            onClick={() => {
              notification.action?.onClick();
              onDismiss(notification.id);
            }}
            className="mt-2 text-sm font-medium text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
          >
            {notification.action.label}
          </button>
        )}
        
        <p className="text-xs text-gray-500 mt-2">
          {notification.timestamp.toLocaleTimeString()}
        </p>
      </div>
      
      <button
        onClick={() => onDismiss(notification.id)}
        className="flex-shrink-0 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}

// Hook for programmatically adding notifications
export function useNotifications() {
  const addNotification = useCallback((notification: Omit<Notification, 'id' | 'timestamp'>) => {
    // Dispatch custom event that NotificationSystem listens to
    window.dispatchEvent(
      new CustomEvent('add-notification', { detail: notification })
    );
  }, []);

  const showSuccess = useCallback((title: string, message: string) => {
    addNotification({ type: 'success', title, message });
  }, [addNotification]);

  const showError = useCallback((title: string, message: string) => {
    addNotification({ type: 'error', title, message, autoDismiss: false });
  }, [addNotification]);

  const showWarning = useCallback((title: string, message: string) => {
    addNotification({ type: 'warning', title, message });
  }, [addNotification]);

  const showInfo = useCallback((title: string, message: string) => {
    addNotification({ type: 'info', title, message });
  }, [addNotification]);

  return {
    addNotification,
    showSuccess,
    showError,
    showWarning,
    showInfo,
  };
}
