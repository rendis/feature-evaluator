import { formatExternalApiPreview } from './external-api-test-utils';

describe('external api test utils', () => {
  it('pretty prints json strings for traffic previews', () => {
    expect(formatExternalApiPreview('{"approved":true,"user":{"id":"123"}}')).toBe(
      JSON.stringify(
        {
          approved: true,
          user: { id: '123' },
        },
        null,
        2,
      ),
    );
  });

  it('keeps plain text responses untouched', () => {
    expect(formatExternalApiPreview('Forbidden')).toBe('Forbidden');
  });
});
