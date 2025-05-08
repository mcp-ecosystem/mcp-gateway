import i18n from '../i18n';

export const t = (key: string, options?: Record<string, string | number | boolean>) => {
  return i18n.t(key, options);
}; 