import { Button, Dropdown, DropdownTrigger, DropdownMenu, DropdownItem } from '@heroui/react';
import { Icon } from '@iconify/react';
import { useTranslation } from 'react-i18next';

const languages = [
  { code: 'en', name: 'English' },
  { code: 'zh', name: '中文' }
];

export function LanguageSwitcher() {
  const { i18n, t } = useTranslation();

  const handleLanguageChange = (languageCode: string) => {
    i18n.changeLanguage(languageCode);
  };

  const currentLanguage = languages.find(lang => lang.code === i18n.language) || languages[0];

  return (
    <Dropdown>
      <DropdownTrigger>
        <Button
          variant="light"
          startContent={<Icon icon="lucide:languages" className="text-lg" />}
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