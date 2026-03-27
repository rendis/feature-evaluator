import { Languages } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { normalizeAppLocale } from '@/lib/locale';
import { useLocaleStore } from '@/stores/locale-store';

export function LanguageToggle() {
  const { i18n, t } = useTranslation();
  const { setLocale } = useLocaleStore();
  const currentLanguage = normalizeAppLocale(i18n.resolvedLanguage ?? i18n.language);

  const changeLanguage = (lng: 'es' | 'en') => {
    setLocale(lng);
    void i18n.changeLanguage(lng);
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon">
          <Languages className="h-4 w-4" />
          <span className="sr-only">{t('language.change')}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuRadioGroup
          value={currentLanguage}
          onValueChange={(value) => changeLanguage(value as 'es' | 'en')}
        >
          <DropdownMenuRadioItem value="es">{t('language.es')}</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="en">{t('language.en')}</DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
