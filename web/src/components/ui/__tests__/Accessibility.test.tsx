import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { renderHook, act } from '@testing-library/react';
import {
  Tooltip,
  HelpTooltip,
  ScreenReaderOnly,
  VisuallyHidden,
  useFocusTrap,
  useKeyboardNavigation,
  SkipLink,
  AccessibleButton,
  LiveRegion,
  FormField,
  announce
} from '../Accessibility';

describe('Accessibility Components', () => {
  describe('Tooltip', () => {
    it('shows tooltip on hover', async () => {
      render(
        <Tooltip content="Help text">
          <button>Hover me</button>
        </Tooltip>
      );

      const button = screen.getByText('Hover me');
      fireEvent.mouseEnter(button);

      await waitFor(() => {
        expect(screen.getByRole('tooltip')).toBeInTheDocument();
        expect(screen.getByText('Help text')).toBeInTheDocument();
      });
    });

    it('hides tooltip on mouse leave', async () => {
      render(
        <Tooltip content="Help text">
          <button>Hover me</button>
        </Tooltip>
      );

      const button = screen.getByText('Hover me');
      fireEvent.mouseEnter(button);
      await waitFor(() => expect(screen.getByRole('tooltip')).toBeInTheDocument());

      fireEvent.mouseLeave(button);
      await waitFor(() => {
        expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
      });
    });

    it('shows tooltip on focus', async () => {
      render(
        <Tooltip content="Help text">
          <button>Focus me</button>
        </Tooltip>
      );

      const button = screen.getByText('Focus me');
      fireEvent.focus(button);

      await waitFor(() => {
        expect(screen.getByRole('tooltip')).toBeInTheDocument();
      });
    });

    it('respects delay prop', async () => {
      render(
        <Tooltip content="Help text" delay={100}>
          <button>Hover me</button>
        </Tooltip>
      );

      const button = screen.getByText('Hover me');
      fireEvent.mouseEnter(button);

      // Should not appear immediately
      expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();

      // Should appear after delay
      await waitFor(() => {
        expect(screen.getByRole('tooltip')).toBeInTheDocument();
      }, { timeout: 200 });
    });
  });

  describe('HelpTooltip', () => {
    it('renders help icon with tooltip', async () => {
      render(<HelpTooltip content="Help information" />);
      
      const button = screen.getByLabelText('Help');
      expect(button).toBeInTheDocument();

      fireEvent.mouseEnter(button);
      await waitFor(() => {
        expect(screen.getByText('Help information')).toBeInTheDocument();
      });
    });
  });

  describe('ScreenReaderOnly', () => {
    it('renders content with sr-only class', () => {
      const { container } = render(
        <ScreenReaderOnly>Hidden from view</ScreenReaderOnly>
      );
      const element = container.querySelector('.sr-only');
      expect(element).toBeInTheDocument();
      expect(element).toHaveTextContent('Hidden from view');
    });
  });

  describe('VisuallyHidden', () => {
    it('renders content visually hidden', () => {
      const { container } = render(
        <VisuallyHidden>Accessible text</VisuallyHidden>
      );
      const element = container.querySelector('span');
      expect(element).toHaveClass('absolute', 'w-px', 'h-px');
    });
  });

  describe('useKeyboardNavigation', () => {
    it('navigates through items with arrow keys', () => {
      const items = ['Item 1', 'Item 2', 'Item 3'];
      const onSelect = vi.fn();

      const { result } = renderHook(() =>
        useKeyboardNavigation(items, onSelect)
      );

      expect(result.current.selectedIndex).toBe(0);

      // Simulate ArrowDown
      act(() => {
        result.current.handleKeyDown({
          key: 'ArrowDown',
          preventDefault: () => {}
        } as React.KeyboardEvent);
      });

      expect(result.current.selectedIndex).toBe(1);

      // Simulate ArrowUp
      act(() => {
        result.current.handleKeyDown({
          key: 'ArrowUp',
          preventDefault: () => {}
        } as React.KeyboardEvent);
      });

      expect(result.current.selectedIndex).toBe(0);
    });

    it('calls onSelect when Enter is pressed', () => {
      const items = ['Item 1', 'Item 2'];
      const onSelect = vi.fn();

      const { result } = renderHook(() =>
        useKeyboardNavigation(items, onSelect)
      );

      act(() => {
        result.current.handleKeyDown({
          key: 'Enter',
          preventDefault: () => {}
        } as React.KeyboardEvent);
      });

      expect(onSelect).toHaveBeenCalledWith('Item 1');
    });

    it('loops navigation when loop option is true', () => {
      const items = ['Item 1', 'Item 2', 'Item 3'];
      const { result } = renderHook(() =>
        useKeyboardNavigation(items, () => {}, { loop: true })
      );

      // Move to last item
      act(() => {
        result.current.setSelectedIndex(2);
      });

      // ArrowDown should loop to first
      act(() => {
        result.current.handleKeyDown({
          key: 'ArrowDown',
          preventDefault: () => {}
        } as React.KeyboardEvent);
      });

      expect(result.current.selectedIndex).toBe(0);
    });
  });

  describe('SkipLink', () => {
    it('renders skip link', () => {
      render(<SkipLink href="#main">Skip to main content</SkipLink>);
      const link = screen.getByText('Skip to main content');
      expect(link).toBeInTheDocument();
      expect(link).toHaveAttribute('href', '#main');
    });

    it('has sr-only class by default', () => {
      const { container } = render(
        <SkipLink href="#main">Skip</SkipLink>
      );
      expect(container.querySelector('.sr-only')).toBeInTheDocument();
    });
  });

  describe('AccessibleButton', () => {
    it('renders children when not loading', () => {
      render(<AccessibleButton>Click me</AccessibleButton>);
      expect(screen.getByText('Click me')).toBeInTheDocument();
    });

    it('shows loading state', () => {
      render(<AccessibleButton loading>Click me</AccessibleButton>);
      expect(screen.getByText('Loading...')).toBeInTheDocument();
    });

    it('disables button when loading', () => {
      render(<AccessibleButton loading>Click me</AccessibleButton>);
      const button = screen.getByRole('button');
      expect(button).toBeDisabled();
      expect(button).toHaveAttribute('aria-busy', 'true');
    });

    it('shows custom loading text', () => {
      render(
        <AccessibleButton loading loadingText="Saving...">
          Save
        </AccessibleButton>
      );
      expect(screen.getByText('Saving...')).toBeInTheDocument();
    });
  });

  describe('LiveRegion', () => {
    it('renders with polite aria-live', () => {
      const { container } = render(
        <LiveRegion>Status update</LiveRegion>
      );
      const region = container.querySelector('[role="status"]');
      expect(region).toHaveAttribute('aria-live', 'polite');
    });

    it('renders with assertive aria-live when not polite', () => {
      const { container } = render(
        <LiveRegion polite={false}>Alert message</LiveRegion>
      );
      const region = container.querySelector('[role="status"]');
      expect(region).toHaveAttribute('aria-live', 'assertive');
    });

    it('has sr-only class', () => {
      const { container } = render(
        <LiveRegion>Update</LiveRegion>
      );
      expect(container.querySelector('.sr-only')).toBeInTheDocument();
    });
  });

  describe('FormField', () => {
    it('renders label and input', () => {
      render(
        <FormField label="Username" id="username">
          <input type="text" />
        </FormField>
      );
      expect(screen.getByLabelText('Username')).toBeInTheDocument();
    });

    it('shows required indicator', () => {
      render(
        <FormField label="Email" id="email" required>
          <input type="email" />
        </FormField>
      );
      const requiredIndicator = screen.getByLabelText('required');
      expect(requiredIndicator).toBeInTheDocument();
    });

    it('displays hint text', () => {
      render(
        <FormField label="Password" id="password" hint="Must be at least 8 characters">
          <input type="password" />
        </FormField>
      );
      expect(screen.getByText('Must be at least 8 characters')).toBeInTheDocument();
    });

    it('displays error message', () => {
      render(
        <FormField label="Email" id="email" error="Invalid email address">
          <input type="email" />
        </FormField>
      );
      expect(screen.getByText('Invalid email address')).toBeInTheDocument();
      expect(screen.getByRole('alert')).toBeInTheDocument();
    });

    it('sets aria-invalid when error exists', () => {
      render(
        <FormField label="Email" id="email" error="Invalid">
          <input type="email" />
        </FormField>
      );
      const input = screen.getByLabelText('Email');
      expect(input).toHaveAttribute('aria-invalid', 'true');
    });

    it('sets aria-describedby for hint and error', () => {
      render(
        <FormField 
          label="Username" 
          id="username" 
          hint="Choose wisely"
          error="Already taken"
        >
          <input type="text" />
        </FormField>
      );
      const input = screen.getByLabelText('Username');
      const describedBy = input.getAttribute('aria-describedby');
      expect(describedBy).toContain('username-hint');
      expect(describedBy).toContain('username-error');
    });
  });

  describe('announce', () => {
    it('creates and removes live region', async () => {
      announce('Test announcement');
      
      await waitFor(() => {
        const liveRegion = document.querySelector('[role="status"]');
        expect(liveRegion).toBeInTheDocument();
        expect(liveRegion).toHaveTextContent('Test announcement');
      });

      // Wait for cleanup
      await waitFor(() => {
        const liveRegion = document.querySelector('[role="status"]');
        expect(liveRegion).not.toBeInTheDocument();
      }, { timeout: 1500 });
    });

    it('uses assertive when not polite', () => {
      announce('Urgent message', false);
      
      const liveRegion = document.querySelector('[aria-live="assertive"]');
      expect(liveRegion).toBeInTheDocument();
    });
  });
});
