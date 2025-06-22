import { Button, Dropdown, DropdownTrigger, DropdownMenu, DropdownItem } from '@heroui/react';
import { useTranslation } from 'react-i18next';

import LocalIcon from './LocalIcon';

const languages = [
  { code: 'en', name: 'English' },
  { code: 'zh', name: '中文' }
];

/**
 * Language switching component that allows users to change the application language.
 * When language is changed, it automatically updates i18n context and all
 * subsequent API requests will include the selected language in X-Lang header.
 */
export function LanguageSwitcher() {
  const { i18n, t } = useTranslation();

  /**
   * Change the application language
   * This automatically affects API requests through the axios interceptor
   * which adds the X-Lang header to all requests
   */
  const handleLanguageChange = (languageCode: string) => {
    i18n.changeLanguage(languageCode);
    // Explicitly save to localStorage to ensure persistence
    window.localStorage.setItem('i18nextLng', languageCode);
  };

  const currentLanguage = languages.find(lang => lang.code === i18n.language) || languages[0];

  return (
    <Dropdown>
      <DropdownTrigger>
        <Button
          variant="light"
          startContent={<LocalIcon icon="lucide:languages" className="text-lg" />}
          aria-label={t('common.switch_language')}
        >
          {currentLanguage.name}
        </Button>
      </DropdownTrigger>
      <DropdownMenu aria-label="Language selection">
        {languages.map((lang) => (
          <DropdownItem
            key={lang.code}
            onPress={() => handleLanguageChange(lang.code)}
            className={i18n.language === lang.code ? 'bg-primary-100' : ''}
          >
            {lang.name}
          </DropdownItem>
        ))}
      </DropdownMenu>
    </Dropdown>
  );
} 