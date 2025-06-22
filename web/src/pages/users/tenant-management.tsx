import {
  Table,
  TableHeader,
  TableColumn,
  TableBody,
  TableRow,
  TableCell,
  Button,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  useDisclosure,
  Input,
  Switch,
  Textarea,
  Modal,
} from '@heroui/react';
import { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';

import LocalIcon from '@/components/LocalIcon';
import { getTenants, createTenant, updateTenant, deleteTenant } from '@/services/api';
import {Tenant, CreateTenantForm, UpdateTenantForm} from '@/types/user';

export function TenantManagement() {
  const { t } = useTranslation();
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedTenant, setSelectedTenant] = useState<Tenant | null>(null);
  const [tenantToDelete, setTenantToDelete] = useState<string>('');
  const [createForm, setCreateForm] = useState<CreateTenantForm>({
    name: '',
    prefix: '',
    description: '',
  });
  const [updateForm, setUpdateForm] = useState<UpdateTenantForm>({
    name: '',
  });

  const {
    isOpen: isCreateOpen,
    onOpen: onCreateOpen,
    onClose: onCreateClose,
  } = useDisclosure();
  const {
    isOpen: isUpdateOpen,
    onOpen: onUpdateOpen,
    onClose: onUpdateClose,
  } = useDisclosure();
  const {
    isOpen: isDeleteOpen,
    onOpen: onDeleteOpen,
    onClose: onDeleteClose,
  } = useDisclosure();

  const fetchTenants = useCallback(async () => {
    try {
      const data = await getTenants();
      setTenants(data);
    } catch {
      // Error already handled in the API function
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchTenants();
  }, [fetchTenants]);

  const handleCreate = async () => {
    try {
      await createTenant(createForm);
      onCreateClose();
      fetchTenants();
      setCreateForm({ name: '', prefix: '', description: '' });
    } catch {
      // Error already handled in the API function
    }
  };

  const handleUpdate = async () => {
    if (!selectedTenant) return;
    try {
      await updateTenant(updateForm);
      onUpdateClose();
      fetchTenants();
      setSelectedTenant(null);
      setUpdateForm({ name: '' });
    } catch {
      // Error already handled in the API function
    }
  };

  const openDeleteConfirm = (name: string) => {
    setTenantToDelete(name);
    onDeleteOpen();
  };

  const handleDelete = async () => {
    try {
      await deleteTenant(tenantToDelete);
      onDeleteClose();
      fetchTenants();
    } catch {
      // Error already handled in the API function
    }
  };

  const handleEdit = (tenant: Tenant) => {
    setSelectedTenant(tenant);
    setUpdateForm({
      name: tenant.name,
      prefix: tenant.prefix,
      description: tenant.description,
      isActive: tenant.isActive,
    });
    onUpdateOpen();
  };

  const handleToggleStatus = async (tenant: Tenant) => {
    try {
      await updateTenant({
        name: tenant.name,
        isActive: !tenant.isActive,
      });
      fetchTenants();
    } catch {
      // Error already handled in the API function
    }
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">{t('tenants.title')}</h1>
        <Button
          color="primary"
          startContent={<LocalIcon icon="lucide:plus" />}
          onPress={onCreateOpen}
        >
          {t('tenants.add')}
        </Button>
      </div>

      <Table aria-label={t('tenants.title')}>
        <TableHeader>
          <TableColumn>{t('tenants.name')}</TableColumn>
          <TableColumn>{t('tenants.prefix')}</TableColumn>
          <TableColumn>{t('tenants.description')}</TableColumn>
          <TableColumn>{t('tenants.status')}</TableColumn>
          <TableColumn>{t('tenants.created_at')}</TableColumn>
          <TableColumn>{t('tenants.actions')}</TableColumn>
        </TableHeader>
        <TableBody
          loadingContent={<div>{t('common.loading')}</div>}
          loadingState={loading ? 'loading' : 'idle'}
        >
          {tenants.map((tenant) => (
            <TableRow key={tenant.id}>
              <TableCell>{tenant.name}</TableCell>
              <TableCell>{tenant.prefix}</TableCell>
              <TableCell>
                <div className="max-w-md truncate">
                  {tenant.description || t('tenants.no_description')}
                </div>
              </TableCell>
              <TableCell>
                <span
                  className={`px-2 py-1 rounded-full text-xs ${
                    tenant.isActive
                      ? 'bg-success-100 text-success-800'
                      : 'bg-danger-100 text-danger-800'
                  }`}
                >
                  {tenant.isActive ? t('tenants.status_enabled') : t('tenants.status_disabled')}
                </span>
              </TableCell>
              <TableCell>
                {new Date(tenant.createdAt).toLocaleString()}
              </TableCell>
              <TableCell>
                <div className="flex items-center gap-4">
                  <div className="flex items-center gap-2">
                    <Switch
                      size="sm"
                      isSelected={tenant.isActive}
                      onValueChange={() => handleToggleStatus(tenant)}
                    />
                    <span className="text-sm text-gray-600">
                      {tenant.isActive ? t('tenants.status_enabled') : t('tenants.status_disabled')}
                    </span>
                  </div>
                  <div className="flex gap-2">
                    <Button
                      size="sm"
                      variant="light"
                      onPress={() => handleEdit(tenant)}
                    >
                      {t('tenants.edit')}
                    </Button>
                    <Button
                      size="sm"
                      color="danger"
                      variant="light"
                      onPress={() => openDeleteConfirm(tenant.name)}
                    >
                      {t('tenants.delete')}
                    </Button>
                  </div>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      {/* Create Tenant Modal */}
      <Modal isOpen={isCreateOpen} onClose={onCreateClose}>
        <ModalContent>
          <ModalHeader>{t('tenants.add')}</ModalHeader>
          <ModalBody>
            <div className="flex flex-col gap-4">
              <Input
                label={t('tenants.name')}
                placeholder={t('tenants.name_placeholder')}
                value={createForm.name}
                onChange={(e) =>
                  setCreateForm({ ...createForm, name: e.target.value })
                }
              />
              <Input
                label={t('tenants.prefix')}
                placeholder={t('tenants.prefix_placeholder')}
                value={createForm.prefix}
                onChange={(e) => {
                  let prefix = e.target.value;
                  if (prefix === '/') {
                    setCreateForm({ ...createForm, prefix });
                    return;
                  }
                  if (prefix && !prefix.startsWith('/') && prefix.trim() !== '') {
                    prefix = `/${prefix}`;
                  }
                  setCreateForm({ ...createForm, prefix });
                }}
              />
              <Textarea
                label={t('tenants.description')}
                value={createForm.description}
                onChange={(e) =>
                  setCreateForm({ ...createForm, description: e.target.value })
                }
              />
            </div>
          </ModalBody>
          <ModalFooter>
            <Button variant="light" onPress={onCreateClose}>
              {t('common.cancel')}
            </Button>
            <Button color="primary" onPress={handleCreate}>
              {t('tenants.add')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Update Tenant Modal */}
      <Modal isOpen={isUpdateOpen} onClose={onUpdateClose}>
        <ModalContent>
          <ModalHeader>{t('tenants.edit')}</ModalHeader>
          <ModalBody>
            <div className="flex flex-col gap-4">
              <Input
                label={t('tenants.name')}
                value={updateForm.name}
                isReadOnly
              />
              <Input
                label={t('tenants.prefix')}
                placeholder={t('tenants.prefix_placeholder')}
                value={updateForm.prefix}
                onChange={(e) => {
                  let prefix = e.target.value;
                  if (prefix === '/') {
                    setUpdateForm({ ...updateForm, prefix });
                    return;
                  }
                  if (prefix && !prefix.startsWith('/') && prefix.trim() !== '') {
                    prefix = `/${prefix}`;
                  }
                  setUpdateForm({ ...updateForm, prefix });
                }}
              />
              <Textarea
                label={t('tenants.description')}
                value={updateForm.description}
                onChange={(e) =>
                  setUpdateForm({ ...updateForm, description: e.target.value })
                }
              />
              <div className="flex items-center gap-2">
                <Switch
                  isSelected={updateForm.isActive}
                  onValueChange={(selected) =>
                    setUpdateForm({ ...updateForm, isActive: selected })
                  }
                />
                <span>
                  {updateForm.isActive ? t('tenants.status_enabled') : t('tenants.status_disabled')}
                </span>
              </div>
            </div>
          </ModalBody>
          <ModalFooter>
            <Button variant="light" onPress={onUpdateClose}>
              {t('common.cancel')}
            </Button>
            <Button color="primary" onPress={handleUpdate}>
              {t('tenants.save')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal isOpen={isDeleteOpen} onClose={onDeleteClose}>
        <ModalContent>
          <ModalHeader className="flex flex-col gap-1">{t('tenants.delete_title')}</ModalHeader>
          <ModalBody>
            <p>{t('tenants.confirm_delete')}</p>
            <p className="text-danger font-semibold">{tenantToDelete}</p>
          </ModalBody>
          <ModalFooter>
            <Button variant="light" onPress={onDeleteClose}>
              {t('common.cancel')}
            </Button>
            <Button color="danger" onPress={handleDelete}>
              {t('tenants.delete')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </div>
  );
} 