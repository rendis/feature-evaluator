import userEvent from '@testing-library/user-event';

import { FeatureBuilder } from './feature-builder';

import { render, screen } from '@/test/test-utils';

const featureMutationState = vi.hoisted(() => ({
  createPending: false,
  updatePending: false,
  deletePending: false,
}));

vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => vi.fn(),
}));

vi.mock('@/hooks/use-unsaved-changes', () => ({
  useUnsavedChanges: () => ({
    handleBack: vi.fn(),
    UnsavedDialog: () => null,
    markClean: vi.fn(),
  }),
}));

vi.mock('@/components/shared/page-header', () => ({
  PageHeader: ({ title, description }: { title: string; description?: string }) => (
    <div>
      <h1>{title}</h1>
      {description ? <p>{description}</p> : null}
    </div>
  ),
}));

vi.mock('@/components/shared/tag-combobox', () => ({
  TagCombobox: () => null,
}));

vi.mock('@/mutations/feature-mutations', () => ({
  useCreateFeature: () => ({ mutate: vi.fn(), isPending: featureMutationState.createPending }),
  useUpdateFeature: () => ({ mutate: vi.fn(), isPending: featureMutationState.updatePending }),
  useDeleteFeature: () => ({ mutate: vi.fn(), isPending: featureMutationState.deletePending }),
}));

describe('FeatureBuilder', () => {
  afterEach(() => {
    featureMutationState.createPending = false;
    featureMutationState.updatePending = false;
    featureMutationState.deletePending = false;
  });

  it('auto-generates key from name on create', async () => {
    const user = userEvent.setup();

    render(<FeatureBuilder />);

    const nameInput = screen.getByLabelText('fields.name');
    await user.type(nameInput, 'My Cool Feature');

    const keyInput = screen.getByLabelText('fields.key') as HTMLInputElement;
    expect(keyInput.value).toBe('my_cool_feature');
  });

  it('normalizes key on manual edit and blur', async () => {
    const user = userEvent.setup();

    render(<FeatureBuilder />);

    const keyInput = screen.getByLabelText('fields.key') as HTMLInputElement;
    await user.type(keyInput, 'Mi Flag.Á_Test@2026---');
    await user.tab();

    expect(keyInput.value).toBe('mi_flag_a_test_2026');
  });

  it('shows the global loading modal while creating a feature', () => {
    featureMutationState.createPending = true;

    render(<FeatureBuilder />);

    expect(screen.getByRole('heading', { name: 'Creando registro' })).toBeInTheDocument();
    expect(
      screen.getByText('Espera un momento mientras completamos la operacion.'),
    ).toBeInTheDocument();
  });
});
