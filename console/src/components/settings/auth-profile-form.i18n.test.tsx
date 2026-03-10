import userEvent from '@testing-library/user-event';

import { AuthProfileBuilder } from '../auth-profiles/auth-profile-builder';

import { render, screen } from '@/test/test-utils';

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

const mutationState = vi.hoisted(() => ({
  createMutate: vi.fn(),
  updateMutate: vi.fn(),
  deleteMutate: vi.fn(),
  testMutate: vi.fn(),
}));

vi.mock('@/mutations/auth-profile-mutations', () => ({
  useCreateAuthProfile: () => ({
    mutate: mutationState.createMutate,
    isPending: false,
  }),
  useUpdateAuthProfile: () => ({
    mutate: mutationState.updateMutate,
    isPending: false,
  }),
  useDeleteAuthProfile: () => ({
    mutate: mutationState.deleteMutate,
    isPending: false,
  }),
  useTestAuthProfile: () => ({
    mutate: mutationState.testMutate,
    isPending: false,
  }),
}));

describe('AuthProfileBuilder i18n', () => {
  beforeEach(() => {
    mutationState.createMutate.mockReset();
    mutationState.updateMutate.mockReset();
    mutationState.deleteMutate.mockReset();
    mutationState.testMutate.mockReset();
  });

  it('renders translated auth profile copy for the create flow and custom builder hotspot', async () => {
    const user = userEvent.setup();

    render(<AuthProfileBuilder />, {
      providerProps: {
        locale: 'es',
        namespaces: ['common', 'settings'],
        useRealTranslations: true,
      },
    });

    // Step 1: Profile — title, type cards
    expect(screen.getByText('Crear auth profile')).toBeInTheDocument();
    expect(screen.getByText('Qué necesitas hacer')).toBeInTheDocument();
    expect(screen.getAllByText('API key o secreto fijo').length).toBeGreaterThan(0);

    // Step 2: Config — navigate to config step, verify api_key editor
    await user.click(screen.getByRole('button', { name: /Configuración/ }));
    expect(screen.getByText('Dónde se envía')).toBeInTheDocument();

    // Switch to Custom type: go back to profile step, select custom
    await user.click(screen.getByRole('button', { name: /Perfil/ }));
    await user.click(screen.getByRole('button', { name: /^Custom / }));

    // Step 2: Config — verify custom editor
    await user.click(screen.getByRole('button', { name: /Configuración/ }));
    expect(screen.getByText('Mapeo del request')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Agregar mapeo' })).toBeInTheDocument();

    // Step 3: Validation — verify success rule
    await user.click(screen.getByRole('button', { name: /Validación/ }));
    expect(screen.getByText('Cómo sabremos que la respuesta es válida')).toBeInTheDocument();
  });
});
