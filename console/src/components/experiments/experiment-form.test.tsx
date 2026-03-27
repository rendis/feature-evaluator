import userEvent from '@testing-library/user-event';

import { ExperimentForm } from './experiment-form';

import { render, screen } from '@/test/test-utils';

const mutationState = vi.hoisted(() => ({
  create: vi.fn(),
  update: vi.fn(),
}));

vi.mock('@/mutations/experiment-mutations', () => ({
  useCreateExperiment: () => ({ mutate: mutationState.create, isPending: false }),
  useUpdateExperiment: () => ({ mutate: mutationState.update, isPending: false }),
}));

describe('ExperimentForm cache controls', () => {
  it('sends lookup cache settings with the create payload', async () => {
    const user = userEvent.setup();

    render(<ExperimentForm featureKeys={['feature-a']} />, {
      providerProps: { namespaces: ['experiments', 'common'] },
    });

    await user.type(screen.getByLabelText('form.name'), 'Experiment cache');
    await user.type(screen.getByLabelText('form.description'), 'Desc');
    await user.selectOptions(screen.getByLabelText('form.featureKey'), 'feature-a');

    const cacheSwitch = screen.getByRole('switch', { name: 'cache.lookup.enabled' });
    await user.click(cacheSwitch);
    const ttlInput = screen.getByLabelText('cache.lookup.ttl');
    await user.clear(ttlInput);
    await user.type(ttlInput, '180');

    await user.click(screen.getByRole('button', { name: 'form.create' }));

    expect(mutationState.create).toHaveBeenCalledWith(
      expect.objectContaining({
        featureKey: 'feature-a',
        lookupCacheEnabled: true,
        lookupCacheTTLSeconds: 180,
      }),
      expect.any(Object),
    );
  });
});
