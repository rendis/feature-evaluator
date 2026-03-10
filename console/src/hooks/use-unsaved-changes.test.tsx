import userEvent from '@testing-library/user-event';

import { useUnsavedChanges } from './use-unsaved-changes';

import { render, screen } from '@/test/test-utils';

const navigateMock = vi.hoisted(() => vi.fn());
const useBlockerMock = vi.hoisted(() => vi.fn());

vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => navigateMock,
  useBlocker: (options: unknown) => useBlockerMock(options),
}));

function TestComponent({
  blockNavigation = false,
  isDirty = true,
}: {
  blockNavigation?: boolean;
  isDirty?: boolean;
}) {
  const { handleBack, UnsavedDialog } = useUnsavedChanges({
    isDirty,
    backTo: '/settings/external-apis',
    blockNavigation,
  });

  return (
    <>
      <button type="button" onClick={handleBack}>
        back
      </button>
      <UnsavedDialog />
    </>
  );
}

describe('useUnsavedChanges', () => {
  afterEach(() => {
    navigateMock.mockReset();
    useBlockerMock.mockReset();
  });

  it('uses the blocker resolver dialog for blocked in-app navigation', async () => {
    const user = userEvent.setup();
    const proceed = vi.fn();
    const reset = vi.fn();

    useBlockerMock.mockReturnValue({
      status: 'blocked',
      current: undefined,
      next: undefined,
      action: 'PUSH',
      proceed,
      reset,
    });

    render(<TestComponent blockNavigation />);

    expect(useBlockerMock).toHaveBeenCalledWith(
      expect.objectContaining({
        enableBeforeUnload: true,
        disabled: false,
        withResolver: true,
      }),
    );
    expect(screen.getByText('unsavedChanges.title')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'unsavedChanges.stay' }));
    expect(reset).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole('button', { name: 'unsavedChanges.leave' }));
    expect(proceed).toHaveBeenCalledTimes(1);
  });

  it('keeps the legacy confirmation flow when route blocking is disabled', async () => {
    const user = userEvent.setup();

    useBlockerMock.mockReturnValue({
      status: 'idle',
      current: undefined,
      next: undefined,
      action: undefined,
      proceed: undefined,
      reset: undefined,
    });

    render(<TestComponent />);

    await user.click(screen.getByRole('button', { name: 'back' }));

    expect(screen.getByText('unsavedChanges.title')).toBeInTheDocument();
    expect(navigateMock).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: 'unsavedChanges.leave' }));

    expect(navigateMock).toHaveBeenCalledWith({
      to: '/settings/external-apis',
      params: undefined,
    });
  });
});
