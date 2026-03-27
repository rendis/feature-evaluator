import userEvent from '@testing-library/user-event';

import { SegmentForm } from './segment-form';

import { render, screen } from '@/test/test-utils';

const mutationState = vi.hoisted(() => ({
  create: vi.fn(),
  update: vi.fn(),
}));

vi.mock('@/mutations/segment-mutations', () => ({
  useCreateSegment: () => ({ mutate: mutationState.create, isPending: false }),
  useUpdateSegment: () => ({ mutate: mutationState.update, isPending: false }),
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

  it('sends the cache configuration in the create payload', async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();

    render(<SegmentForm open={true} onOpenChange={onOpenChange} />);

    await user.type(screen.getByLabelText('fields.name'), 'Segment with cache');
    await user.type(screen.getByLabelText('fields.description'), 'Description');

    await user.click(screen.getByRole('switch', { name: 'cache.membership' }));
    await user.click(screen.getByRole('switch', { name: 'cache.record' }));

    await user.clear(screen.getByLabelText('TTL', { selector: '#segment-membership-cache-ttl' }));
    await user.type(screen.getByLabelText('TTL', { selector: '#segment-membership-cache-ttl' }), '120');
    await user.clear(screen.getByLabelText('TTL', { selector: '#segment-record-cache-ttl' }));
    await user.type(screen.getByLabelText('TTL', { selector: '#segment-record-cache-ttl' }), '240');

    await user.click(screen.getByRole('button', { name: 'actions.save' }));

    expect(mutationState.create).toHaveBeenCalledWith(
      expect.objectContaining({
        key: 'segment_with_cache',
        membershipCacheEnabled: true,
        membershipCacheTTLSeconds: 120,
        recordCacheEnabled: true,
        recordCacheTTLSeconds: 240,
      }),
      expect.any(Object),
    );
  });
});
