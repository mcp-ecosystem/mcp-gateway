import axios, { AxiosError } from 'axios';
import { t } from 'i18next';

import { toast } from './toast';

/**
 * Handle API errors and display user-friendly error messages
 * 
 * @param error - The error object
 * @param fallbackMessage - Default message to display if specific error information is not available
 * @returns The error message that was displayed
 */
export const handleApiError = (error: unknown, fallbackMessage: string): string => {
  // Handle standard error responses
  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError;
    const serverError = axiosError.response?.data as { error?: string };
    
    if (serverError?.error) {
      // Server error message is already i18n translated
      toast.error(serverError.error, { duration: 3000 });
      return serverError.error;
    }
  }
  
  // Handle general or network errors
  toast.error(t(fallbackMessage), { duration: 3000 });
  return t(fallbackMessage);
}; 