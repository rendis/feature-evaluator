import {
  formatDate,
  formatDateTime,
  formatNumber,
  formatRelativeTime,
  formatTime,
} from '@/lib/locale-format';

describe('locale format helpers', () => {
  const sampleDate = '2026-03-05T14:30:00Z';

  it('formats dates and times using the active locale', () => {
    expect(
      formatDate(sampleDate, 'es', {
        timeZone: 'UTC',
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
      }),
    ).toBe('05/03/2026');
    expect(
      formatDate(sampleDate, 'en', {
        timeZone: 'UTC',
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
      }),
    ).toBe('03/05/2026');

    expect(
      formatDateTime(sampleDate, 'es', {
        timeZone: 'UTC',
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        hourCycle: 'h23',
      }),
    ).toBe('05/03/2026, 14:30');
    expect(
      formatTime(sampleDate, 'en', {
        timeZone: 'UTC',
        hour: '2-digit',
        minute: '2-digit',
        hourCycle: 'h23',
      }),
    ).toBe('14:30');
  });

  it('formats numbers and percentages using the active locale', () => {
    expect(formatNumber(12345.6, 'es')).toBe('12.345,6');
    expect(formatNumber(12345.6, 'en')).toBe('12,345.6');
    expect(formatNumber(0.756, 'es', { style: 'percent', maximumFractionDigits: 1 })).toBe(
      '75,6\u00a0%',
    );
    expect(formatNumber(0.756, 'en', { style: 'percent', maximumFractionDigits: 1 })).toBe('75.6%');
  });

  it('formats relative time using the active locale', () => {
    const now = new Date('2026-03-07T12:00:00Z');

    expect(formatRelativeTime('2026-03-06T12:00:00Z', 'es', now)).toBe('ayer');
    expect(formatRelativeTime('2026-03-06T12:00:00Z', 'en', now)).toBe('yesterday');
  });

  it('returns an empty string for invalid dates', () => {
    expect(formatDate('invalid-date', 'es')).toBe('');
    expect(formatDateTime(undefined, 'en')).toBe('');
    expect(formatRelativeTime(null, 'es')).toBe('');
  });
});
