import { createContext } from 'react';

export interface GlobalLoadingOptions {
  title?: string;
  description?: string;
}

export type SubmissionLoadingAction = 'create' | 'save' | 'update';

export interface LoadingEntry {
  id: symbol;
  options: GlobalLoadingOptions;
}

export interface GlobalLoadingContextValue {
  hide: (id: symbol) => void;
  show: (id: symbol, options: GlobalLoadingOptions) => void;
}

export const GlobalLoadingContext = createContext<GlobalLoadingContextValue | null>(null);
