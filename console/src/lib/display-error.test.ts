import { ApiClientError } from '@/api/client';
import { getVisibleErrorMessage } from '@/lib/display-error';

describe('getVisibleErrorMessage', () => {
  it('returns the backend message for ApiClientError', () => {
    const error = new ApiClientError(400, {
      code: 'INVALID_REQUEST',
      message: 'backend technical message',
      messageKey: 'error.invalidRequest',
      requestId: 'req-123',
    });

    expect(getVisibleErrorMessage(error, 'fallback')).toBe('backend technical message');
  });

  it('returns the localized fallback for generic errors', () => {
    expect(getVisibleErrorMessage(new Error('boom'), 'Algo salió mal')).toBe('Algo salió mal');
    expect(getVisibleErrorMessage('boom', 'Unknown error')).toBe('Unknown error');
  });
});
