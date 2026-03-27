import userEvent from '@testing-library/user-event';

import { LanguageToggle } from './language-toggle';

import { render, screen } from '@/test/test-utils';

const setLocale = vi.fn();

vi.mock('@/stores/locale-store', () => ({
  useLocaleStore: () => ({
    setLocale,
  }),
}));

describe('LanguageToggle', () => {
  beforeEach(() => {
    setLocale.mockReset();
  });

  it('marks the current language when the menu opens', async () => {
    const user = userEvent.setup();

    render(<LanguageToggle />, {
      providerProps: {
        locale: 'en',
      },
    });

    await user.click(screen.getByRole('button', { name: 'language.change' }));

    expect(screen.getByRole('menuitemradio', { name: 'language.en' })).toHaveAttribute(
      'data-state',
      'checked',
    );
    expect(screen.getByRole('menuitemradio', { name: 'language.es' })).toHaveAttribute(
      'data-state',
      'unchecked',
    );
  });
});
