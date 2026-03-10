import userEvent from '@testing-library/user-event';

import { SegmentForm } from './segment-form';

import { render, screen } from '@/test/test-utils';

vi.mock('@/mutations/segment-mutations', () => ({
  useCreateSegment: () => ({ mutate: vi.fn(), isPending: false }),
  useUpdateSegment: () => ({ mutate: vi.fn(), isPending: false }),
}));

describe('SegmentForm', () => {
  it('applies the segment key normalization on blur', async () => {
    const user = userEvent.setup();

    render(<SegmentForm open={true} onOpenChange={vi.fn()} />);

    const nameInput = screen.getByLabelText('fields.name') as HTMLInputElement;
    await user.type(nameInput, 'Mi Segment.Á_Test@2026---');
    await user.tab();

    expect(screen.getByText('mi_segment_a_test_2026')).toBeInTheDocument();
  });
});
