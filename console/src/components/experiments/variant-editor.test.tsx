import userEvent from '@testing-library/user-event';

import { VariantEditor } from './variant-editor';

import type { Variant } from '@/api/types';

import { render, screen } from '@/test/test-utils';

const twoVariants: Variant[] = [
  { key: 'control', value: 'false', weight: 50 },
  { key: 'treatment', value: 'true', weight: 50 },
];

const threeVariants: Variant[] = [
  ...twoVariants,
  { key: 'extra', value: 'maybe', weight: 0 },
];

describe('VariantEditor', () => {
  it('renders all variant rows', () => {
    const onChange = vi.fn();
    render(<VariantEditor variants={twoVariants} onChange={onChange} />);

    expect(screen.getByDisplayValue('control')).toBeInTheDocument();
    expect(screen.getByDisplayValue('treatment')).toBeInTheDocument();
  });

  it('calls onChange with a new variant when add button is clicked', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<VariantEditor variants={twoVariants} onChange={onChange} />);

    const addButton = screen.getByRole('button', { name: /form\.addVariant/i });
    await user.click(addButton);

    expect(onChange).toHaveBeenCalledWith([
      ...twoVariants,
      { key: '', value: '', weight: 50 },
    ]);
  });

  it('disables remove buttons when only 2 variants exist', () => {
    const onChange = vi.fn();
    render(<VariantEditor variants={twoVariants} onChange={onChange} />);

    const removeButtons = screen.getAllByRole('button').filter(
      (btn) => !btn.textContent?.includes('form.addVariant'),
    );
    // The trash buttons (all except the add button)
    const trashButtons = removeButtons.filter((btn) => btn.querySelector('svg'));
    for (const btn of trashButtons) {
      expect(btn).toBeDisabled();
    }
  });

  it('enables remove buttons when more than 2 variants exist', () => {
    const onChange = vi.fn();
    render(<VariantEditor variants={threeVariants} onChange={onChange} />);

    const trashButtons = screen.getAllByRole('button').filter(
      (btn) => btn.querySelector('svg') && !btn.textContent?.includes('form.addVariant'),
    );
    for (const btn of trashButtons) {
      expect(btn).not.toBeDisabled();
    }
  });

  it('calls onChange without the removed variant', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<VariantEditor variants={threeVariants} onChange={onChange} />);

    // Click the first enabled trash button
    const trashButtons = screen.getAllByRole('button').filter(
      (btn) => btn.querySelector('svg') && !btn.textContent?.includes('form.addVariant'),
    );
    await user.click(trashButtons[0]);

    expect(onChange).toHaveBeenCalledWith([twoVariants[1], threeVariants[2]]);
  });

  it('calls onChange when key input changes', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<VariantEditor variants={twoVariants} onChange={onChange} />);

    const keyInput = screen.getByDisplayValue('control');
    await user.clear(keyInput);
    // onChange is called per keystroke on clear
    expect(onChange).toHaveBeenCalled();
  });

  it('renders weight inputs with correct values', () => {
    const onChange = vi.fn();
    render(<VariantEditor variants={twoVariants} onChange={onChange} />);

    const weightInputs = screen.getAllByRole('spinbutton');
    expect(weightInputs).toHaveLength(2);
    expect(weightInputs[0]).toHaveValue(50);
    expect(weightInputs[1]).toHaveValue(50);
  });

  it('disables all inputs when disabled prop is true', () => {
    const onChange = vi.fn();
    render(<VariantEditor variants={twoVariants} onChange={onChange} disabled />);

    const inputs = screen.getAllByRole('textbox');
    for (const input of inputs) {
      expect(input).toBeDisabled();
    }

    const addButton = screen.getByRole('button', { name: /form\.addVariant/i });
    expect(addButton).toBeDisabled();
  });
});
