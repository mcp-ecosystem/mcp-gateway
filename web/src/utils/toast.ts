import { addToast } from "@heroui/react";

export const toast = {
  success: (message: string, options?: { description?: string; duration?: number }) => {
    addToast({
      title: message,
      description: options?.description,
      // color: "success",
      timeout: options?.duration,
    });
  },
  error: (message: string, options?: { description?: string; duration?: number }) => {
    addToast({
      title: message,
      description: options?.description,
      color: "danger",
      timeout: options?.duration,
    });
  },
  warning: (message: string, options?: { description?: string; duration?: number }) => {
    addToast({
      title: message,
      description: options?.description,
      color: "warning",
      timeout: options?.duration,
    });
  },
  info: (message: string, options?: { description?: string; duration?: number }) => {
    addToast({
      title: message,
      description: options?.description,
      color: "primary",
      timeout: options?.duration,
    });
  },
};
