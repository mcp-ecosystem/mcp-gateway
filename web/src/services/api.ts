import axios from 'axios';
import { t } from 'i18next';

import { toast } from '../utils/toast';

// Create an axios instance with default config
const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add response interceptor
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Clear any existing token
      window.localStorage.removeItem('token');
      // Only redirect if not already on login page
      if (window.location.pathname !== '/login') {
        window.location.href = '/login';
      }
      // If already on login page, do not redirect, just clear token
    }
    return Promise.reject(error);
  }
);

// Add request interceptor to add token to headers
api.interceptors.request.use(
  (config) => {
    const token = window.localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// API endpoints
export const getMCPServers = async (tenantId?: number) => {
  try {
    const params = tenantId ? { tenantId } : {};
    const response = await api.get('/mcp-servers', { params });
    return response.data;
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.data?.error) {
      toast.error(t('errors.fetch_mcp_servers'), {
        duration: 3000,
      });
    } else {
      toast.error(t('errors.fetch_mcp_servers'), {
        duration: 3000,
      });
    }
    throw error;
  }
};

export const getMCPServer = async (name: string) => {
  try {
    const response = await api.get(`/mcp-servers/${name}`);
    return response.data;
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.data?.error) {
      toast.error(t('errors.fetch_mcp_server'), {
        duration: 3000,
      });
    } else {
      toast.error(t('errors.fetch_mcp_server'), {
        duration: 3000,
      });
    }
    throw error;
  }
};

// Helper function to map backend error messages to i18n keys
const mapErrorToI18nKey = (errorMessage: string): string => {
  // Direct error codes returned from backend
  if (errorMessage.startsWith('errors.')) {
    return errorMessage;
  }

  // Legacy error message mapping
  if (errorMessage.includes('Tenant field is required')) {
    return 'errors.tenant_required';
  }
  if (errorMessage.includes('Tenant with prefix') && errorMessage.includes('does not exist')) {
    return 'errors.tenant_not_found';
  }
  if (errorMessage.includes('User does not have permission to configure')) {
    return 'errors.tenant_permission_error';
  }
  if (errorMessage.includes('router prefix') && errorMessage.includes('must start with tenant prefix')) {
    return 'errors.router_prefix_error';
  }
  if (errorMessage === 'errors.namespace_permission_error') {
    return 'errors.namespace_permission_error';
  }
  if (errorMessage === 'errors.tenant_permission_error') {
    return 'errors.tenant_permission_error';
  }
  return '';
};

export const createMCPServer = async (config: string) => {
  try {
    const response = await api.post('/mcp-servers', config, {
      headers: {
        'Content-Type': 'application/yaml',
      },
    });
    return response.data;
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.data?.error) {
      const errorMessage = error.response.data.error;
      const i18nKey = mapErrorToI18nKey(errorMessage);
      
      if (i18nKey) {
        toast.error(t(i18nKey), { duration: 3000 });
      } else {
        toast.error(t('errors.create_mcp_server'), { duration: 3000 });
      }
    } else {
      toast.error(t('errors.create_mcp_server'), { duration: 3000 });
    }
    throw error;
  }
};

export const updateMCPServer = async (name: string, config: string) => {
  try {
    const response = await api.put(`/mcp-servers/${name}`, config, {
      headers: {
        'Content-Type': 'application/yaml',
      },
    });
    return response.data;
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.data?.error) {
      const errorMessage = error.response.data.error;
      const i18nKey = mapErrorToI18nKey(errorMessage);
      
      if (i18nKey) {
        toast.error(t(i18nKey), { duration: 3000 });
      } else {
        toast.error(t('errors.update_mcp_server'), { duration: 3000 });
      }
    } else {
      toast.error(t('errors.update_mcp_server'), { duration: 3000 });
    }
    throw error;
  }
};

export const deleteMCPServer = async (name: string) => {
  try {
    const response = await api.delete(`/mcp-servers/${name}`);
    return response.data;
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.data?.error) {
      toast.error(t('errors.delete_mcp_server'), {
        duration: 3000,
      });
    } else {
      toast.error(t('errors.delete_mcp_server'), {
        duration: 3000,
      });
    }
    throw error;
  }
};

export const syncMCPServers = async () => {
  try {
    const response = await api.post('/mcp-servers/sync');
    return response.data;
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.data?.error) {
      toast.error(t('errors.sync_mcp_server'), {
        duration: 3000,
      });
    } else {
      toast.error(t('errors.sync_mcp_server'), {
        duration: 3000,
      });
    }
    throw error;
  }
};

export const getChatMessages = async (sessionId: string, page: number = 1, pageSize: number = 20) => {
  try {
    const response = await api.get(`/chat/sessions/${sessionId}/messages`, {
      params: {
        page,
        pageSize,
      },
    });
    return response.data;
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.data?.error) {
      toast.error(t('errors.fetch_chat_messages'), {
        duration: 3000,
      });
    } else {
      toast.error(t('errors.fetch_chat_messages'), {
        duration: 3000,
      });
    }
    throw error;
  }
};

export const getChatSessions = async () => {
  try {
    const response = await api.get('/chat/sessions');
    return response.data;
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.data?.error) {
      toast.error(t('errors.fetch_chat_sessions'), {
        duration: 3000,
      });
    } else {
      toast.error(t('errors.fetch_chat_sessions'), {
        duration: 3000,
      });
    }
    throw error;
  }
};

export const importOpenAPI = async (file: File) => {
  try {
    const formData = new globalThis.FormData();
    formData.append('file', file);

    const response = await api.post('/openapi/import', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    return response.data;
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.data?.error) {
      toast.error(t('errors.import_openapi'), {
        duration: 3000,
      });
    } else {
      toast.error(t('errors.import_openapi'), {
        duration: 3000,
      });
    }
    throw error;
  }
};

// Tenant related API functions
export const getTenants = async () => {
  try {
    const response = await api.get('/auth/tenants');
    return response.data;
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.data?.error) {
      toast.error(t('errors.fetch_tenants'), {
        duration: 3000,
      });
    } else {
      toast.error(t('errors.fetch_tenants'), {
        duration: 3000,
      });
    }
    throw error;
  }
};

export const getTenant = async (name: string) => {
  try {
    const response = await api.get(`/auth/tenants/${name}`);
    return response.data;
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.data?.error) {
      toast.error(t('errors.fetch_tenant'), {
        duration: 3000,
      });
    } else {
      toast.error(t('errors.fetch_tenant'), {
        duration: 3000,
      });
    }
    throw error;
  }
};

interface Tenant {
  id: number;
  name: string;
  prefix: string;
  description: string;
  isActive: boolean;
}

export const createTenant = async (data: { name: string; prefix: string; description: string }) => {
  try {
    const { name, prefix, description } = data;
    
    // Check if prefix conflicts with existing ones
    const tenants = await getTenants();
    if (checkPrefixConflict(prefix, tenants.map((t: Tenant) => t.prefix))) {
      toast.error(t('errors.prefix_conflict'), {
        duration: 3000,
      });
      throw new Error('Prefix conflict');
    }
    
    // Ensure prefix starts with /
    let prefixed = prefix;
    if (prefixed && !prefixed.startsWith('/')) {
      prefixed = `/${prefixed}`;
    }

    // Check if it's a root level directory
    if (prefixed === '/') {
      toast.error(t('tenants.root_prefix_not_allowed'), {
        duration: 3000,
      });
      throw new Error('Root prefix not allowed');
    }

    // First get all tenants, check for prefix conflicts
    if (checkPrefixConflict(prefixed, tenants.map((t: Tenant) => t.prefix))) {
      toast.error(t('tenants.prefix_path_conflict'), {
        duration: 3000,
      });
      throw new Error('Prefix path conflict');
    }

    const response = await api.post('/auth/tenants', {
      name,
      prefix: prefixed,
      description,
    });
    toast.success(t('tenants.add_success'), {
      duration: 3000,
    });
    return response.data;
  } catch (error) {
    if (
      axios.isAxiosError(error) &&
      error.response?.data?.error &&
      !error.message.includes('Prefix')
    ) {
      toast.error(t('tenants.add_failed'), {
        duration: 3000,
      });
    }
    throw error;
  }
};

// Check if prefix conflicts with existing prefixes (same, parent path or child path)
const checkPrefixConflict = (prefix: string, existingPrefixes: string[], excludePrefix?: string): boolean => {
  for (const existingPrefix of existingPrefixes) {
    // Skip the prefix being edited (only used when updating)
    if (excludePrefix && existingPrefix === excludePrefix) {
      continue;
    }

    // Check if it's a parent path - e.g., /a is the parent path of /a/b
    if (prefix.startsWith(existingPrefix + '/') || existingPrefix === prefix) {
      return true;
    }

    // Check if it's a child path - e.g., /a/b is a child path of /a
    if (existingPrefix.startsWith(prefix + '/')) {
      return true;
    }
  }
  return false;
};

export const updateTenant = async (data: { name: string; prefix?: string; description?: string; isActive?: boolean }) => {
  try {
    const { name, prefix } = data;
    
    if (prefix) {
      // Get current tenant information
      const currentTenant = await getTenant(name);
      
      // Check for conflicts if prefix has changed
      if (currentTenant.prefix !== prefix) {
        const tenants = await getTenants();
        if (checkPrefixConflict(prefix, tenants.map((t: Tenant) => t.prefix), currentTenant.prefix)) {
          toast.error(t('errors.prefix_conflict'), {
            duration: 3000,
          });
          throw new Error('Prefix conflict');
        }
      }
    }

    const response = await api.put('/auth/tenants', data);
    toast.success(t('tenants.edit_success'), {
      duration: 3000,
    });
    return response.data;
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.status === 409) {
      // Check specific error message to distinguish between name conflict and prefix conflict
      const errorMessage = error.response.data?.error;
      if (errorMessage === "Tenant name already exists") {
        toast.error(t('tenants.name_conflict'), {
          duration: 3000,
        });
      } else {
        toast.error(t('tenants.prefix_conflict'), {
          duration: 3000,
        });
      }
    } else if (!(error instanceof Error && 
               (error.message === 'Root prefix not allowed' || 
                error.message === 'Prefix path conflict'))) {
      toast.error(t('errors.update_tenant'), {
        duration: 3000,
      });
    }
    throw error;
  }
};

export const deleteTenant = async (name: string) => {
  try {
    const response = await api.delete(`/auth/tenants/${name}`);
    toast.success(t('tenants.delete_success'), {
      duration: 3000,
    });
    return response.data;
  } catch (error) {
    if (axios.isAxiosError(error) && error.response?.data?.error) {
      toast.error(t('errors.delete_tenant'), {
        duration: 3000,
      });
    } else {
      toast.error(t('tenants.delete_failed'), {
        duration: 3000,
      });
    }
    throw error;
  }
};

// User related API functions
export const getUsers = async () => {
  try {
    const response = await api.get('/auth/users');
    return response.data;
  } catch (error) {
    toast.error(t('errors.fetch_users'), {
      duration: 3000,
    });
    throw error;
  }
};

export const getUser = async (username: string) => {
  try {
    const response = await api.get(`/auth/users/${username}`);
    return response.data;
  } catch (error) {
    toast.error(t('errors.fetch_user'), {
      duration: 3000,
    });
    throw error;
  }
};

export const createUser = async (data: { 
  username: string; 
  password: string; 
  role: 'admin' | 'normal';
  tenantIds?: number[];
}) => {
  try {
    const response = await api.post('/auth/users', data);
    toast.success(t('users.add_success'), {
      duration: 3000,
    });
    return response.data;
  } catch (error) {
    toast.error(t('users.add_failed'), {
      duration: 3000,
    });
    throw error;
  }
};

export const updateUser = async (data: { 
  username: string; 
  password?: string; 
  role?: 'admin' | 'normal'; 
  isActive?: boolean;
  tenantIds?: number[];
}) => {
  try {
    const response = await api.put('/auth/users', data);
    toast.success(t('users.edit_success'), {
      duration: 3000,
    });
    return response.data;
  } catch (error) {
    toast.error(t('users.edit_failed'), {
      duration: 3000,
    });
    throw error;
  }
};

export const deleteUser = async (username: string) => {
  try {
    const response = await api.delete(`/auth/users/${username}`);
    toast.success(t('users.delete_success'), {
      duration: 3000,
    });
    return response.data;
  } catch (error) {
    toast.error(t('users.delete_failed'), {
      duration: 3000,
    });
    throw error;
  }
};

export const toggleUserStatus = async (username: string, isActive: boolean) => {
  try {
    const response = await api.put('/auth/users', {
      username,
      isActive,
    });
    toast.success(isActive ? t('users.enable_success') : t('users.disable_success'), {
      duration: 3000,
    });
    return response.data;
  } catch (error) {
    toast.error(isActive ? t('users.enable_failed') : t('users.disable_failed'), {
      duration: 3000,
    });
    throw error;
  }
};

// Get user details and associated tenants
export const getUserWithTenants = async (username: string) => {
  try {
    const response = await api.get(`/auth/users/${username}`);
    return response.data;
  } catch (error) {
    toast.error(t('errors.fetch_user'), {
      duration: 3000,
    });
    throw error;
  }
};

// Get current user's authorized tenants
export const getUserAuthorizedTenants = async () => {
  try {
    const response = await api.get('/auth/user');
    return response.data.tenants || [];
  } catch (error) {
    toast.error(t('errors.fetch_authorized_tenants'), {
      duration: 3000,
    });
    throw error;
  }
};

// Update user tenant associations
export const updateUserTenants = async (userId: number, tenantIds: number[]) => {
  try {
    const response = await api.put('/auth/users/tenants', {
      userId,
      tenantIds
    });
    toast.success(t('users.update_tenants_success'), {
      duration: 3000,
    });
    return response.data;
  } catch (error) {
    toast.error(t('users.update_tenants_failed'), {
      duration: 3000,
    });
    throw error;
  }
};

export default api;
