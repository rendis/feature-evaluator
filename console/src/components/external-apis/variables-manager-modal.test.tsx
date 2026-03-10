import userEvent from '@testing-library/user-event';

import { VariablesManagerModal } from './variables-manager-modal';

import { fireEvent, render, screen } from '@/test/test-utils';

describe('VariablesManagerModal', () => {
  it('renders manual variables without entering a render loop and allows renaming them', async () => {
    const user = userEvent.setup();
    const onRenameVariable = vi.fn();

    render(
      <VariablesManagerModal
        open
        onOpenChange={vi.fn()}
        variables={{
          campus_id: {
            origin: 'detected',
            type: 'string',
            required: true,
            locations: new Set(['url_path']),
          },
          manual_var_1: {
            origin: 'manual',
            type: 'any',
            required: false,
            locations: new Set(),
          },
        }}
        onAddVariable={vi.fn()}
        onRenameVariable={onRenameVariable}
        onRemoveVariable={vi.fn()}
        onTypeChange={vi.fn()}
        onRequiredChange={vi.fn()}
      />,
    );

    const manualInput = screen.getByDisplayValue('manual_var_1');
    expect(manualInput).toBeInTheDocument();
    expect(screen.getByText('variablesModal.badges.manual')).toBeInTheDocument();

    await user.clear(manualInput);
    await user.type(manualInput, 'campus_code');
    fireEvent.blur(manualInput);

    expect(onRenameVariable).toHaveBeenCalledWith('manual_var_1', 'campus_code');
    expect(screen.getByText('variablesModal.badges.detected')).toBeInTheDocument();
  });
});
