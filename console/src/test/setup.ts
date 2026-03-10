import '@testing-library/jest-dom/vitest';

const storageBackingStore = new Map<string, string>();

const memoryStorage: Storage = {
  get length() {
    return storageBackingStore.size;
  },
  clear() {
    storageBackingStore.clear();
  },
  getItem(key: string) {
    return storageBackingStore.get(key) ?? null;
  },
  key(index: number) {
    return Array.from(storageBackingStore.keys())[index] ?? null;
  },
  removeItem(key: string) {
    storageBackingStore.delete(key);
  },
  setItem(key: string, value: string) {
    storageBackingStore.set(key, value);
  },
};

Object.defineProperty(window, 'localStorage', {
  configurable: true,
  value: memoryStorage,
});

// Polyfill ResizeObserver for jsdom (used by Radix UI components)
globalThis.ResizeObserver = class ResizeObserver {
  observe() {
    return undefined;
  }
  unobserve() {
    return undefined;
  }
  disconnect() {
    return undefined;
  }
};

Object.defineProperty(window, 'matchMedia', {
  configurable: true,
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
    addListener: () => undefined,
    removeListener: () => undefined,
    dispatchEvent: () => false,
  }),
});
