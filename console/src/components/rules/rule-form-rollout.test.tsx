import { RolloutSection } from './rule-form-rollout';

import { fireEvent, render, screen } from '@/test/test-utils';

describe('RolloutSection', () => {
  it('keeps rollout controls visible while disabled', () => {
    const onChange = vi.fn();

    render(<RolloutSection enabled={false} percentage={50} onChange={onChange} />);

    expect(screen.getByRole('slider')).toBeInTheDocument();
    expect(screen.getByRole('spinbutton', { name: 'rollout.limitLabel' })).toBeDisabled();
    expect(screen.getByText('rollout.partial')).toBeInTheDocument();
  });

  it('enables rollout controls when the section is active', () => {
    const onChange = vi.fn();

    render(<RolloutSection enabled percentage={75} onChange={onChange} />);

    expect(screen.getByRole('spinbutton', { name: 'rollout.limitLabel' })).toBeEnabled();
  });

  it('calls onChange when the number input value changes', () => {
    const onChange = vi.fn();

    render(<RolloutSection enabled percentage={50} onChange={onChange} />);

    const numInput = screen.getByRole('spinbutton', { name: 'rollout.limitLabel' });
    fireEvent.change(numInput, { target: { value: '30' } });

    expect(onChange).toHaveBeenCalledWith(30);
  });

  it('shows all-users text when percentage is 100', () => {
    const onChange = vi.fn();

    render(<RolloutSection enabled percentage={100} onChange={onChange} />);

    expect(screen.getByText('rollout.allUsers')).toBeInTheDocument();
  });

  it('shows the percent symbol next to the number input', () => {
    const onChange = vi.fn();

    render(<RolloutSection enabled percentage={50} onChange={onChange} />);

    expect(screen.getByText('%')).toBeInTheDocument();
  });
});
