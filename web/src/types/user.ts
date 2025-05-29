export interface User {
  id: number;
  username: string;
  role: 'admin' | 'normal';
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
  tenants?: Tenant[];
}

export interface Tenant {
  id: number;
  name: string;
  prefix: string;
  description: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CreateUserForm {
  username: string;
  password: string;
  role: 'admin' | 'normal';
  tenantIds?: number[];
}

export interface UpdateUserForm {
  username: string;
  password?: string;
  role?: 'admin' | 'normal';
  isActive?: boolean;
  tenantIds?: number[];
}

export interface CreateTenantForm {
  name: string;
  prefix: string;
  description: string;
}

export interface UpdateTenantForm {
  name: string;
  prefix?: string;
  description?: string;
  isActive?: boolean;
} 