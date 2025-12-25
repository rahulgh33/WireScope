import React, { useState } from 'react';
import { HelpCircle } from 'lucide-react';

// Tooltip Component
interface TooltipProps {
  content: string | React.ReactNode;
  children: React.ReactNode;
  position?: 'top' | 'bottom' | 'left' | 'right';
  delay?: number;
}

export function Tooltip({ content, children, position = 'top', delay = 200 }: TooltipProps) {
  const [isVisible, setIsVisible] = useState(false);
  const [timeoutId, setTimeoutId] = useState<number | null>(null);

  const showTooltip = () => {
    const id = window.setTimeout(() => setIsVisible(true), delay);
    setTimeoutId(id);
  };

  const hideTooltip = () => {
    if (timeoutId) {
      window.clearTimeout(timeoutId);
    }
    setIsVisible(false);
  };

  const positionClasses = {
    top: 'bottom-full left-1/2 -translate-x-1/2 mb-2',
    bottom: 'top-full left-1/2 -translate-x-1/2 mt-2',
    left: 'right-full top-1/2 -translate-y-1/2 mr-2',
    right: 'left-full top-1/2 -translate-y-1/2 ml-2'
  };

  return (
    <div
      className="relative inline-block"
      onMouseEnter={showTooltip}
      onMouseLeave={hideTooltip}
      onFocus={showTooltip}
      onBlur={hideTooltip}
    >
      {children}
      {isVisible && (
        <div
          className={`absolute z-50 px-3 py-2 text-sm text-white bg-gray-900 rounded-lg shadow-lg whitespace-nowrap pointer-events-none ${positionClasses[position]}`}
          role="tooltip"
        >
          {content}
          <div className="absolute w-2 h-2 bg-gray-900 transform rotate-45" style={{
            [position === 'top' ? 'bottom' : position === 'bottom' ? 'top' : position === 'left' ? 'right' : 'left']: '-4px',
            [position === 'top' || position === 'bottom' ? 'left' : 'top']: '50%',
            transform: 'translateX(-50%) rotate(45deg)'
          }} />
        </div>
      )}
    </div>
  );
}

// Help Icon with Tooltip
export function HelpTooltip({ content }: { content: string }) {
  return (
    <Tooltip content={content}>
      <button
        type="button"
        className="inline-flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
        aria-label="Help"
      >
        <HelpCircle className="h-4 w-4" />
      </button>
    </Tooltip>
  );
}

// Screen Reader Only Text
export function ScreenReaderOnly({ children }: { children: React.ReactNode }) {
  return (
    <span className="sr-only">
      {children}
    </span>
  );
}

// Visually Hidden (for accessibility)
export function VisuallyHidden({ children }: { children: React.ReactNode }) {
  return (
    <span className="absolute w-px h-px p-0 -m-px overflow-hidden whitespace-nowrap border-0">
      {children}
    </span>
  );
}

// Focus Trap (for modals/dialogs)
export function useFocusTrap(ref: React.RefObject<HTMLElement>) {
  React.useEffect(() => {
    if (!ref.current) return;

    const element = ref.current;
    const focusableElements = element.querySelectorAll(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );
    const firstElement = focusableElements[0] as HTMLElement;
    const lastElement = focusableElements[focusableElements.length - 1] as HTMLElement;

    const handleTab = (e: KeyboardEvent) => {
      if (e.key !== 'Tab') return;

      if (e.shiftKey) {
        if (document.activeElement === firstElement) {
          lastElement?.focus();
          e.preventDefault();
        }
      } else {
        if (document.activeElement === lastElement) {
          firstElement?.focus();
          e.preventDefault();
        }
      }
    };

    element.addEventListener('keydown', handleTab);
    firstElement?.focus();

    return () => {
      element.removeEventListener('keydown', handleTab);
    };
  }, [ref]);
}

// Keyboard Navigation Helper
export function useKeyboardNavigation(
  items: any[],
  onSelect: (item: any) => void,
  options: { loop?: boolean; vertical?: boolean } = {}
) {
  const [selectedIndex, setSelectedIndex] = useState(0);
  const { loop = true, vertical = true } = options;

  const handleKeyDown = (e: React.KeyboardEvent) => {
    const nextKey = vertical ? 'ArrowDown' : 'ArrowRight';
    const prevKey = vertical ? 'ArrowUp' : 'ArrowLeft';

    if (e.key === nextKey) {
      e.preventDefault();
      setSelectedIndex((prev) => {
        const next = prev + 1;
        return loop ? next % items.length : Math.min(next, items.length - 1);
      });
    } else if (e.key === prevKey) {
      e.preventDefault();
      setSelectedIndex((prev) => {
        const next = prev - 1;
        return loop ? (next + items.length) % items.length : Math.max(next, 0);
      });
    } else if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      onSelect(items[selectedIndex]);
    } else if (e.key === 'Home') {
      e.preventDefault();
      setSelectedIndex(0);
    } else if (e.key === 'End') {
      e.preventDefault();
      setSelectedIndex(items.length - 1);
    }
  };

  return { selectedIndex, handleKeyDown, setSelectedIndex };
}

// Skip Link for Accessibility
export function SkipLink({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <a
      href={href}
      className="sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4 focus:z-50 focus:px-4 focus:py-2 focus:bg-primary focus:text-white focus:rounded-md focus:shadow-lg"
    >
      {children}
    </a>
  );
}

// Accessible Button with Loading State
interface AccessibleButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  loading?: boolean;
  loadingText?: string;
  children: React.ReactNode;
}

export function AccessibleButton({
  loading,
  loadingText = 'Loading...',
  children,
  disabled,
  ...props
}: AccessibleButtonProps) {
  return (
    <button
      {...props}
      disabled={loading || disabled}
      aria-busy={loading}
      aria-disabled={loading || disabled}
    >
      {loading ? (
        <>
          <span className="inline-block animate-spin mr-2">⏳</span>
          <ScreenReaderOnly>{loadingText}</ScreenReaderOnly>
          <span aria-hidden="true">{loadingText}</span>
        </>
      ) : (
        children
      )}
    </button>
  );
}

// Live Region for Dynamic Updates (for screen readers)
interface LiveRegionProps {
  children: React.ReactNode;
  polite?: boolean;
}

export function LiveRegion({ children, polite = true }: LiveRegionProps) {
  return (
    <div
      role="status"
      aria-live={polite ? 'polite' : 'assertive'}
      aria-atomic="true"
      className="sr-only"
    >
      {children}
    </div>
  );
}

// Accessible Form Field
interface FormFieldProps {
  label: string;
  id: string;
  error?: string;
  hint?: string;
  required?: boolean;
  children: React.ReactNode;
}

export function FormField({ label, id, error, hint, required, children }: FormFieldProps) {
  const hintId = `${id}-hint`;
  const errorId = `${id}-error`;

  return (
    <div className="space-y-2">
      <label htmlFor={id} className="block text-sm font-medium">
        {label}
        {required && <span className="text-destructive ml-1" aria-label="required">*</span>}
      </label>
      {hint && (
        <p id={hintId} className="text-sm text-muted-foreground">
          {hint}
        </p>
      )}
      <div>
        {React.cloneElement(children as React.ReactElement<any>, {
          id,
          'aria-describedby': [hint && hintId, error && errorId].filter(Boolean).join(' ') || undefined,
          'aria-invalid': error ? 'true' : undefined,
          'aria-required': required
        })}
      </div>
      {error && (
        <p id={errorId} className="text-sm text-destructive" role="alert">
          {error}
        </p>
      )}
    </div>
  );
}

// Announce to Screen Reader
export function announce(message: string, polite: boolean = true) {
  const liveRegion = document.createElement('div');
  liveRegion.setAttribute('role', 'status');
  liveRegion.setAttribute('aria-live', polite ? 'polite' : 'assertive');
  liveRegion.setAttribute('aria-atomic', 'true');
  liveRegion.className = 'sr-only';
  liveRegion.textContent = message;

  document.body.appendChild(liveRegion);

  setTimeout(() => {
    document.body.removeChild(liveRegion);
  }, 1000);
}
