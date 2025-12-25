import React from 'react';
import { AlertCircle, AlertTriangle, Info, CheckCircle, X } from 'lucide-react';
import { Button } from '@/components/ui/Button';

// Error Boundary Component
interface ErrorBoundaryProps {
  children: React.ReactNode;
  fallback?: React.ReactNode;
  onReset?: () => void;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('Error caught by boundary:', error, errorInfo);
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null });
    this.props.onReset?.();
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <ErrorFallback
          error={this.state.error}
          onReset={this.handleReset}
        />
      );
    }

    return this.props.children;
  }
}

// Error Fallback UI
interface ErrorFallbackProps {
  error: Error | null;
  onReset?: () => void;
}

export function ErrorFallback({ error, onReset }: ErrorFallbackProps) {
  return (
    <div className="flex min-h-[400px] flex-col items-center justify-center p-8">
      <div className="text-center space-y-4 max-w-md">
        <AlertCircle className="h-16 w-16 text-destructive mx-auto" />
        <h2 className="text-2xl font-bold">Something went wrong</h2>
        <p className="text-muted-foreground">
          {error?.message || 'An unexpected error occurred'}
        </p>
        {onReset && (
          <Button onClick={onReset} className="mt-4">
            Try Again
          </Button>
        )}
      </div>
    </div>
  );
}

// Loading Spinner Component
export function LoadingSpinner({ size = 'md', className = '' }: { size?: 'sm' | 'md' | 'lg'; className?: string }) {
  const sizeClasses = {
    sm: 'h-4 w-4',
    md: 'h-8 w-8',
    lg: 'h-12 w-12'
  };

  return (
    <div className={`flex items-center justify-center ${className}`}>
      <div className={`${sizeClasses[size]} animate-spin rounded-full border-4 border-primary border-t-transparent`} />
    </div>
  );
}

// Loading State Component
export function LoadingState({ message = 'Loading...' }: { message?: string }) {
  return (
    <div className="flex min-h-[400px] flex-col items-center justify-center space-y-4">
      <LoadingSpinner size="lg" />
      <p className="text-muted-foreground">{message}</p>
    </div>
  );
}

// Empty State Component
interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  description?: string;
  action?: React.ReactNode;
}

export function EmptyState({ icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="flex min-h-[400px] flex-col items-center justify-center p-8">
      <div className="text-center space-y-4 max-w-md">
        {icon && <div className="flex justify-center text-muted-foreground">{icon}</div>}
        <h3 className="text-xl font-semibold">{title}</h3>
        {description && <p className="text-muted-foreground">{description}</p>}
        {action && <div className="mt-4">{action}</div>}
      </div>
    </div>
  );
}

// Alert Component
export type AlertVariant = 'info' | 'success' | 'warning' | 'error';

interface AlertProps {
  variant?: AlertVariant;
  title?: string;
  children: React.ReactNode;
  onDismiss?: () => void;
}

export function Alert({ variant = 'info', title, children, onDismiss }: AlertProps) {
  const variantConfig = {
    info: {
      icon: Info,
      className: 'bg-blue-50 border-blue-200 text-blue-900 dark:bg-blue-950 dark:border-blue-800 dark:text-blue-100'
    },
    success: {
      icon: CheckCircle,
      className: 'bg-green-50 border-green-200 text-green-900 dark:bg-green-950 dark:border-green-800 dark:text-green-100'
    },
    warning: {
      icon: AlertTriangle,
      className: 'bg-yellow-50 border-yellow-200 text-yellow-900 dark:bg-yellow-950 dark:border-yellow-800 dark:text-yellow-100'
    },
    error: {
      icon: AlertCircle,
      className: 'bg-red-50 border-red-200 text-red-900 dark:bg-red-950 dark:border-red-800 dark:text-red-100'
    }
  };

  const config = variantConfig[variant];
  const Icon = config.icon;

  return (
    <div className={`relative rounded-lg border p-4 ${config.className}`} role="alert">
      <div className="flex gap-3">
        <Icon className="h-5 w-5 flex-shrink-0 mt-0.5" />
        <div className="flex-1">
          {title && <h5 className="font-semibold mb-1">{title}</h5>}
          <div className="text-sm">{children}</div>
        </div>
        {onDismiss && (
          <button
            onClick={onDismiss}
            className="flex-shrink-0 p-1 hover:opacity-70 transition-opacity"
            aria-label="Dismiss"
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </div>
    </div>
  );
}

// Inline Error Message
export function InlineError({ message }: { message: string }) {
  return (
    <div className="flex items-center gap-2 text-sm text-destructive">
      <AlertCircle className="h-4 w-4" />
      <span>{message}</span>
    </div>
  );
}

// Retry Component
interface RetryProps {
  onRetry: () => void;
  error?: Error | null;
  message?: string;
}

export function Retry({ onRetry, error, message }: RetryProps) {
  return (
    <div className="flex flex-col items-center justify-center space-y-4 p-8">
      <AlertCircle className="h-12 w-12 text-muted-foreground" />
      <div className="text-center space-y-2">
        <h3 className="font-semibold">Failed to load data</h3>
        <p className="text-sm text-muted-foreground">
          {message || error?.message || 'An error occurred while loading'}
        </p>
      </div>
      <Button onClick={onRetry} variant="outline">
        Try Again
      </Button>
    </div>
  );
}

// Progress Indicator
interface ProgressProps {
  value: number;
  max?: number;
  label?: string;
  showValue?: boolean;
  className?: string;
}

export function Progress({ value, max = 100, label, showValue = false, className = '' }: ProgressProps) {
  const percentage = Math.min(Math.max((value / max) * 100, 0), 100);

  return (
    <div className={`w-full ${className}`} role="progressbar" aria-valuenow={value} aria-valuemax={max}>
      {(label || showValue) && (
        <div className="flex justify-between mb-1 text-sm text-muted-foreground">
          {label && <span>{label}</span>}
          {showValue && <span>{percentage.toFixed(0)}%</span>}
        </div>
      )}
      <div className="h-2 w-full bg-secondary rounded-full overflow-hidden">
        <div
          className="h-full bg-primary transition-all duration-300 ease-in-out"
          style={{ width: `${percentage}%` }}
          aria-hidden="true"
        />
      </div>
    </div>
  );
}

// Skeleton Loader
export function SkeletonCard({ showHeader = true }: { showHeader?: boolean }) {
  return (
    <div className="rounded-lg border border-border bg-card p-6 space-y-4 animate-pulse">
      {showHeader && <div className="h-4 bg-muted rounded w-1/4" />}
      <div className="h-8 bg-muted rounded w-1/2" />
      <div className="space-y-2">
        <div className="h-3 bg-muted rounded" />
        <div className="h-3 bg-muted rounded w-5/6" />
      </div>
    </div>
  );
}

export function SkeletonTable({ rows = 5, columns = 3 }: { rows?: number; columns?: number }) {
  return (
    <div className="w-full">
      <table className="w-full">
        <thead>
          <tr>
            {Array.from({ length: columns }).map((_, i) => (
              <th key={i} className="p-2">
                <div className="h-4 bg-muted rounded animate-pulse" />
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {Array.from({ length: rows }).map((_, i) => (
            <tr key={i}>
              {Array.from({ length: columns }).map((_, j) => (
                <td key={j} className="p-2">
                  <div className="h-4 bg-muted rounded animate-pulse" />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
