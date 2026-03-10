import userEvent from '@testing-library/user-event';

import { WorkspaceFormDialog } from './workspace-form-dialog';

import type { ComponentProps } from 'react';

import { render, screen } from '@/test/test-utils';

// Mock the mutation hooks
vi.mock('@/mutations/workspace-mutations', () => ({
  useCreateWorkspace: () => ({ mutate: vi.fn(), isPending: false }),
  useUpdateWorkspace: () => ({ mutate: vi.fn(), isPending: false }),
}));

const testWorkspace = {
  id: '1',
  key: 'test-ws',
  name: 'Test Workspace',
  description: '',
  createdAt: '2025-01-01T00:00:00Z',
  updatedAt: '2025-01-01T00:00:00Z',
  createdBy: 'user1',
};

function renderDialog(props: Partial<ComponentProps<typeof WorkspaceFormDialog>> = {}) {
  return render(<WorkspaceFormDialog open={true} onOpenChange={vi.fn()} {...props} />);
}

describe('WorkspaceFormDialog', () => {
  it('renders form fields when open', () => {
    renderDialog();
    expect(screen.getByLabelText('fields.name')).toBeInTheDocument();
    expect(screen.getByLabelText('fields.key')).toBeInTheDocument();
    expect(screen.getByLabelText('fields.description')).toBeInTheDocument();
  });

  it('does not render form when closed', () => {
    render(<WorkspaceFormDialog open={false} onOpenChange={vi.fn()} />);
    expect(screen.queryByLabelText('fields.name')).not.toBeInTheDocument();
  });

  it('shows create title when no workspace provided', () => {
    renderDialog();
    expect(screen.getByText('form.createTitle')).toBeInTheDocument();
  });

  it('shows edit title when workspace provided', () => {
    renderDialog({ workspace: testWorkspace });
    expect(screen.getByText('form.editTitle')).toBeInTheDocument();
  });

  it('disables key field when editing', () => {
    renderDialog({ workspace: testWorkspace });
    expect(screen.getByLabelText('fields.key')).toBeDisabled();
  });

  it('key field is enabled when creating', () => {
    renderDialog();
    expect(screen.getByLabelText('fields.key')).not.toBeDisabled();
  });

  it('auto-slugifies the key field from name input', async () => {
    const user = userEvent.setup();
    renderDialog();
    const nameInput = screen.getByLabelText('fields.name');
    await user.type(nameInput, 'My New Workspace');
    const keyInput = screen.getByLabelText('fields.key') as HTMLInputElement;
    expect(keyInput.value).toBe('my_new_workspace');
  });

  it('applies the shared key normalization on manual input blur', async () => {
    const user = userEvent.setup();
    renderDialog();
    const keyInput = screen.getByLabelText('fields.key') as HTMLInputElement;
    await user.type(keyInput, 'Mi Workspace.Á_Test@2026---');
    await user.tab();
    expect(keyInput.value).toBe('mi_workspace_a_test_2026');
  });

  it('renders cancel and save buttons', () => {
    renderDialog();
    expect(screen.getByRole('button', { name: /actions\.cancel/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /actions\.save/i })).toBeInTheDocument();
  });
});
