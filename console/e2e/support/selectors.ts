import type { Locator, Page } from '@playwright/test';

export function switchByLabel(page: Page, label: string): Locator {
  return page.getByRole('switch', { name: label });
}

export function checkboxByLabel(page: Page, label: string): Locator {
  return page.getByRole('checkbox', { name: label });
}

export function numberInputByLabel(page: Page, label: string): Locator {
  return page.getByLabel(label);
}

export function tabByName(page: Page, name: string): Locator {
  return page.getByRole('tab', { name });
}

export function buttonByName(page: Page, name: string): Locator {
  return page.getByRole('button', { name });
}
