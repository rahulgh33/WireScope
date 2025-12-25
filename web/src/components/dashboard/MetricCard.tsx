import React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Skeleton } from '@/components/ui/Skeleton';
import { cn } from '@/lib/utils';

interface MetricCardProps {
  title: string;
  value: string | number;
  unit?: string;
  change?: {
    value: number;
    period: string;
  };
  icon?: React.ReactNode;
  loading?: boolean;
  status?: 'default' | 'success' | 'warning' | 'danger';
}

export function MetricCard({
  title,
  value,
  unit,
  change,
  icon,
  loading,
  status = 'default',
}: MetricCardProps) {
  if (loading) {
    return (
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">
            <Skeleton className="h-4 w-24" />
          </CardTitle>
          <Skeleton className="h-4 w-4 rounded" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-8 w-32 mb-2" />
          <Skeleton className="h-3 w-20" />
        </CardContent>
      </Card>
    );
  }

  const statusColors = {
    default: '',
    success: 'text-green-600 dark:text-green-400',
    warning: 'text-yellow-600 dark:text-yellow-400',
    danger: 'text-red-600 dark:text-red-400',
  };

  const changeColor = change && change.value > 0 ? 'text-green-600' : 'text-red-600';

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        {icon && <div className="text-muted-foreground">{icon}</div>}
      </CardHeader>
      <CardContent>
        <div className={cn('text-2xl font-bold', statusColors[status])}>
          {value}
          {unit && <span className="text-base font-normal ml-1">{unit}</span>}
        </div>
        {change && (
          <p className={cn('text-xs mt-1', changeColor)}>
            {change.value > 0 ? '+' : ''}
            {change.value}% from {change.period}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
