import {
  fireEvent,
  render as rtlRender,
  screen,
  waitFor,
  type RenderOptions,
} from '@testing-library/react';
import { type ReactElement, type ReactNode } from 'react';

import { TestProviders, type TestProviderProps } from '@/test/test-providers';

interface ExtendedRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  providerProps?: Omit<TestProviderProps, 'children'>;
}

function customRender(ui: ReactElement, options?: ExtendedRenderOptions) {
  const { providerProps, ...renderOptions } = options ?? {};

  function Wrapper({ children }: { children: ReactNode }) {
    return <TestProviders {...providerProps}>{children}</TestProviders>;
  }

  return rtlRender(ui, { wrapper: Wrapper, ...renderOptions });
}

export { fireEvent, screen, waitFor };
export { customRender as render };
