import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import {
  ErrorBoundary,
  LoadingSpinner,
  LoadingState,
  EmptyState,
  Alert,
  InlineError,
  Retry,
  Progress,
  SkeletonCard,
  SkeletonTable
} from '../ErrorHandling';

describe('ErrorHandling Components', () => {
  describe('ErrorBoundary', () => {
    it('renders children when there is no error', () => {
      render(
        <ErrorBoundary>
          <div>Test Content</div>
        </ErrorBoundary>
      );
      expect(screen.getByText('Test Content')).toBeInTheDocument();
    });

    it('renders error fallback when child throws error', () => {
      const ThrowError = () => {
        throw new Error('Test error');
      };

      // Suppress console.error for this test
      const spy = vi.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <ErrorBoundary>
          <ThrowError />
        </ErrorBoundary>
      );

      expect(screen.getByText(/Something went wrong/i)).toBeInTheDocument();
      expect(screen.getByText(/Test error/i)).toBeInTheDocument();
      spy.mockRestore();
    });

    it('calls onReset when reset button is clicked', () => {
      const onReset = vi.fn();
      const ThrowError = () => {
        throw new Error('Test error');
      };

      const spy = vi.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <ErrorBoundary onReset={onReset}>
          <ThrowError />
        </ErrorBoundary>
      );

      const resetButton = screen.getByText(/Try again/i);
      fireEvent.click(resetButton);
      expect(onReset).toHaveBeenCalled();
      spy.mockRestore();
    });
  });

  describe('LoadingSpinner', () => {
    it('renders with default size', () => {
      const { container } = render(<LoadingSpinner />);
      expect(container.querySelector('.h-8.w-8')).toBeInTheDocument();
    });

    it('renders with custom size', () => {
      const { container } = render(<LoadingSpinner size="sm" />);
      expect(container.querySelector('.h-4.w-4')).toBeInTheDocument();
    });

    it('has accessibility attributes', () => {
      render(<LoadingSpinner />);
      const spinner = screen.getByRole('status');
      expect(spinner).toHaveAttribute('aria-label', 'Loading');
    });
  });

  describe('LoadingState', () => {
    it('renders loading message', () => {
      render(<LoadingState message="Loading data..." />);
      expect(screen.getByText('Loading data...')).toBeInTheDocument();
    });

    it('renders default message when not provided', () => {
      render(<LoadingState />);
      expect(screen.getByText('Loading...')).toBeInTheDocument();
    });

    it('includes loading spinner', () => {
      render(<LoadingState />);
      expect(screen.getByRole('status')).toBeInTheDocument();
    });
  });

  describe('EmptyState', () => {
    it('renders title and description', () => {
      render(
        <EmptyState
          title="No Data"
          description="There is nothing to display"
        />
      );
      expect(screen.getByText('No Data')).toBeInTheDocument();
      expect(screen.getByText('There is nothing to display')).toBeInTheDocument();
    });

    it('renders icon when provided', () => {
      const Icon = () => <svg data-testid="test-icon" />;
      render(<EmptyState title="Empty" icon={Icon} />);
      expect(screen.getByTestId('test-icon')).toBeInTheDocument();
    });

    it('renders action button when provided', () => {
      const handleAction = vi.fn();
      render(
        <EmptyState
          title="Empty"
          action={{ label: 'Create New', onClick: handleAction }}
        />
      );
      const button = screen.getByText('Create New');
      fireEvent.click(button);
      expect(handleAction).toHaveBeenCalled();
    });
  });

  describe('Alert', () => {
    it('renders info variant by default', () => {
      render(<Alert>Information</Alert>);
      expect(screen.getByRole('alert')).toHaveClass('bg-blue-50');
    });

    it('renders different variants', () => {
      const { rerender } = render(<Alert variant="success">Success</Alert>);
      expect(screen.getByRole('alert')).toHaveClass('bg-green-50');

      rerender(<Alert variant="warning">Warning</Alert>);
      expect(screen.getByRole('alert')).toHaveClass('bg-yellow-50');

      rerender(<Alert variant="error">Error</Alert>);
      expect(screen.getByRole('alert')).toHaveClass('bg-red-50');
    });

    it('renders title when provided', () => {
      render(<Alert title="Alert Title">Message</Alert>);
      expect(screen.getByText('Alert Title')).toBeInTheDocument();
    });

    it('can be dismissed', () => {
      const onDismiss = vi.fn();
      render(<Alert onDismiss={onDismiss}>Dismissible</Alert>);
      
      const closeButton = screen.getByRole('button');
      fireEvent.click(closeButton);
      expect(onDismiss).toHaveBeenCalled();
    });
  });

  describe('InlineError', () => {
    it('renders error message', () => {
      render(<InlineError message="Invalid input" />);
      expect(screen.getByText('Invalid input')).toBeInTheDocument();
    });

    it('has error icon', () => {
      const { container } = render(<InlineError message="Error" />);
      expect(container.querySelector('svg')).toBeInTheDocument();
    });
  });

  describe('Retry', () => {
    it('renders error and retry button', () => {
      const onRetry = vi.fn();
      render(
        <Retry
          error={new Error('Failed to load')}
          onRetry={onRetry}
        />
      );
      
      expect(screen.getByText(/Something went wrong/i)).toBeInTheDocument();
      expect(screen.getByText(/Failed to load/i)).toBeInTheDocument();
      
      const retryButton = screen.getByText('Try Again');
      fireEvent.click(retryButton);
      expect(onRetry).toHaveBeenCalled();
    });

    it('shows custom message when provided', () => {
      render(
        <Retry
          error={new Error('Error')}
          onRetry={() => {}}
          message="Custom error message"
        />
      );
      expect(screen.getByText('Custom error message')).toBeInTheDocument();
    });
  });

  describe('Progress', () => {
    it('renders with percentage', () => {
      render(<Progress value={50} max={100} />);
      const progress = screen.getByRole('progressbar');
      expect(progress).toHaveAttribute('aria-valuenow', '50');
      expect(progress).toHaveAttribute('aria-valuemax', '100');
    });

    it('shows label when provided', () => {
      render(<Progress value={75} max={100} label="Upload progress" />);
      expect(screen.getByText('Upload progress')).toBeInTheDocument();
    });

    it('shows percentage text when showValue is true', () => {
      render(<Progress value={33} max={100} showValue />);
      expect(screen.getByText('33%')).toBeInTheDocument();
    });

    it('renders indeterminate state', () => {
      const { container } = render(<Progress value={0} max={100} />);
      const bar = container.querySelector('[style*="width"]');
      expect(bar).toHaveStyle({ width: '0%' });
    });
  });

  describe('SkeletonCard', () => {
    it('renders skeleton structure', () => {
      const { container } = render(<SkeletonCard />);
      expect(container.querySelector('.animate-pulse')).toBeInTheDocument();
    });

    it('hides header when showHeader is false', () => {
      const { container } = render(<SkeletonCard showHeader={false} />);
      const skeletons = container.querySelectorAll('.h-4');
      // Should have fewer skeleton lines without header
      expect(skeletons.length).toBeLessThan(5);
    });
  });

  describe('SkeletonTable', () => {
    it('renders specified number of rows', () => {
      const { container } = render(<SkeletonTable rows={3} />);
      const rows = container.querySelectorAll('tbody tr');
      expect(rows).toHaveLength(3);
    });

    it('renders specified number of columns', () => {
      const { container } = render(<SkeletonTable rows={1} columns={4} />);
      const cells = container.querySelectorAll('tbody td');
      expect(cells).toHaveLength(4);
    });

    it('renders table header', () => {
      const { container } = render(<SkeletonTable />);
      expect(container.querySelector('thead')).toBeInTheDocument();
    });
  });
});
