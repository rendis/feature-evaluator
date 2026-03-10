import { ExperimentStatusBadge } from './experiment-status-badge';

import type { ExperimentStatus } from '@/api/types';

import { render, screen } from '@/test/test-utils';


describe('ExperimentStatusBadge', () => {
  const cases: { status: ExperimentStatus; variant: string }[] = [
    { status: 'draft', variant: 'secondary' },
    { status: 'running', variant: 'success' },
    { status: 'paused', variant: 'warning' },
    { status: 'completed', variant: 'default' },
  ];

  it.each(cases)('renders badge for status "$status"', ({ status }) => {
    render(<ExperimentStatusBadge status={status} />);
    // i18n returns the key as-is since no translations are loaded
    expect(screen.getByText(`status.${status}`)).toBeInTheDocument();
  });

  it('uses secondary variant for draft', () => {
    const { container } = render(<ExperimentStatusBadge status="draft" />);
    const badge = container.firstElementChild;
    expect(badge?.className).toContain('bg-secondary');
  });

  it('uses success variant for running', () => {
    const { container } = render(<ExperimentStatusBadge status="running" />);
    const badge = container.firstElementChild;
    expect(badge?.className).toContain('bg-success');
  });

  it('uses warning variant for paused', () => {
    const { container } = render(<ExperimentStatusBadge status="paused" />);
    const badge = container.firstElementChild;
    expect(badge?.className).toContain('bg-warning');
  });

  it('uses default (primary) variant for completed', () => {
    const { container } = render(<ExperimentStatusBadge status="completed" />);
    const badge = container.firstElementChild;
    expect(badge?.className).toContain('bg-primary');
  });
});
