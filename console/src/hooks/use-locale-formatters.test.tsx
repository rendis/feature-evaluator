import userEvent from '@testing-library/user-event';
import { useTranslation } from 'react-i18next';

import { useLocaleFormatters } from '@/hooks/use-locale-formatters';
import { render, screen, waitFor } from '@/test/test-utils';

function LocaleFormatterProbe() {
  const { i18n } = useTranslation('common');
  const { locale, formatDate, formatNumber, formatRelativeTime } = useLocaleFormatters();

  return (
    <div>
      <div data-testid="locale">{locale}</div>
      <div data-testid="date">
        {formatDate('2026-03-05T14:30:00Z', {
          timeZone: 'UTC',
          year: 'numeric',
          month: '2-digit',
          day: '2-digit',
        })}
      </div>
      <div data-testid="number">{formatNumber(12345.6)}</div>
      <div data-testid="relative">
        {formatRelativeTime('2026-03-06T12:00:00Z', new Date('2026-03-07T12:00:00Z'))}
      </div>
      <button type="button" onClick={() => void i18n.changeLanguage('en')}>
        switch
      </button>
    </div>
  );
}

describe('useLocaleFormatters', () => {
  it('rerenders copy and formatting when the app language changes', async () => {
    const user = userEvent.setup();

    render(<LocaleFormatterProbe />, {
      providerProps: {
        locale: 'es',
        namespaces: ['common'],
        useRealTranslations: true,
      },
    });

    expect(screen.getByTestId('locale')).toHaveTextContent('es');
    expect(screen.getByTestId('date')).toHaveTextContent('05/03/2026');
    expect(screen.getByTestId('number')).toHaveTextContent('12.345,6');
    expect(screen.getByTestId('relative')).toHaveTextContent('ayer');

    await user.click(screen.getByRole('button', { name: 'switch' }));

    await waitFor(() => {
      expect(screen.getByTestId('locale')).toHaveTextContent('en');
    });

    expect(screen.getByTestId('date')).toHaveTextContent('03/05/2026');
    expect(screen.getByTestId('number')).toHaveTextContent('12,345.6');
    expect(screen.getByTestId('relative')).toHaveTextContent('yesterday');
  });
});
