import { Languages } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useLocaleStore } from '@/stores/locale-store';

export function LanguageToggle() {
  const { i18n, t } = useTranslation();
  const { setLocale } = useLocaleStore();

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
        <DropdownMenuItem onClick={() => changeLanguage('es')}>{t('language.es')}</DropdownMenuItem>
        <DropdownMenuItem onClick={() => changeLanguage('en')}>{t('language.en')}</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
