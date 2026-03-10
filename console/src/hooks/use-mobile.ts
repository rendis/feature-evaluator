import { useMediaQuery } from './use-media-query';

/** Returns true when viewport is below the desktop breakpoint (1024px). */
export function useMobile(): boolean {
  return useMediaQuery('(max-width: 1023px)');
}
