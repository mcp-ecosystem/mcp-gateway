import {
  Table,
  TableHeader,
  TableColumn,
  TableBody,
  TableRow,
  TableCell,
  Button,
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  useDisclosure,
  Input,
  Select,
  SelectItem,
  Switch,
} from '@heroui/react';
import { Icon } from '@iconify/react';
import { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';

import api from '../../services/api';
import { toast } from '../../utils/toast';

interface User {
  id: number;
  username: string;
  role: 'admin' | 'normal';
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

interface CreateUserForm {
  username: string;
  password: string;
  role: 'admin' | 'normal';
}

interface UpdateUserForm {
  username: string;
  password?: string;
  role?: 'admin' | 'normal';
  isActive?: boolean;
}

export function UserManagement() {
  const { t } = useTranslation();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [createForm, setCreateForm] = useState<CreateUserForm>({
    username: '',
    password: '',
    role: 'normal',
  });
  const [updateForm, setUpdateForm] = useState<UpdateUserForm>({
    username: '',
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

  const fetchUsers = useCallback(async () => {
    try {
      const response = await api.get('/auth/users');
      setUsers(response.data);
    } catch {
      toast.error(t('errors.fetch_users'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  const handleCreate = async () => {
    try {
      await api.post('/auth/users', createForm);
      toast.success(t('users.add_success'));
      onCreateClose();
      fetchUsers();
      setCreateForm({ username: '', password: '', role: 'normal' });
    } catch {
      toast.error(t('users.add_failed'));
    }
  };

  const handleUpdate = async () => {
    if (!selectedUser) return;
    try {
      await api.put('/auth/users', updateForm);
      toast.success(t('users.edit_success'));
      onUpdateClose();
      fetchUsers();
      setSelectedUser(null);
      setUpdateForm({ username: '' });
    } catch {
      toast.error(t('users.edit_failed'));
    }
  };

  const handleDelete = async (username: string) => {
    if (!window.confirm(t('users.confirm_delete'))) return;
    try {
      await api.delete(`/auth/users/${username}`);
      toast.success(t('users.delete_success'));
      fetchUsers();
    } catch {
      toast.error(t('users.delete_failed'));
    }
  };

  const handleEdit = (user: User) => {
    setSelectedUser(user);
    setUpdateForm({
      username: user.username,
      role: user.role,
      isActive: user.isActive,
    });
    onUpdateOpen();
  };

  const handleToggleStatus = async (user: User) => {
    try {
      await api.put('/auth/users', {
        username: user.username,
        isActive: !user.isActive,
      });
      toast.success(t(user.isActive ? 'users.disable_success' : 'users.enable_success'));
      fetchUsers();
    } catch {
      toast.error(t(user.isActive ? 'users.disable_failed' : 'users.enable_failed'));
    }
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">{t('users.title')}</h1>
        <Button
          color="primary"
          startContent={<Icon icon="lucide:plus" />}
          onPress={onCreateOpen}
        >
          {t('users.add')}
        </Button>
      </div>

      <Table aria-label={t('users.title')}>
        <TableHeader>
          <TableColumn>{t('users.username')}</TableColumn>
          <TableColumn>{t('users.role')}</TableColumn>
          <TableColumn>{t('users.status')}</TableColumn>
          <TableColumn>{t('users.created_at')}</TableColumn>
          <TableColumn>{t('users.actions')}</TableColumn>
        </TableHeader>
        <TableBody
          loadingContent={<div>{t('common.loading')}</div>}
          loadingState={loading ? 'loading' : 'idle'}
        >
          {users.map((user) => (
            <TableRow key={user.id}>
              <TableCell>{user.username}</TableCell>
              <TableCell>{user.role === 'admin' ? t('users.role_admin') : t('users.role_normal')}</TableCell>
              <TableCell>
                <span
                  className={`px-2 py-1 rounded-full text-xs ${
                    user.isActive
                      ? 'bg-success-100 text-success-800'
                      : 'bg-danger-100 text-danger-800'
                  }`}
                >
                  {user.isActive ? t('users.status_enabled') : t('users.status_disabled')}
                </span>
              </TableCell>
              <TableCell>
                {new Date(user.createdAt).toLocaleString()}
              </TableCell>
              <TableCell>
                <div className="flex items-center gap-4">
                  <div className="flex items-center gap-2">
                    <Switch
                      size="sm"
                      isSelected={user.isActive}
                      onValueChange={() => handleToggleStatus(user)}
                    />
                    <span className="text-sm text-gray-600">
                      {user.isActive ? t('users.status_enabled') : t('users.status_disabled')}
                    </span>
                  </div>
                  <div className="flex gap-2">
                    <Button
                      size="sm"
                      variant="light"
                      onPress={() => handleEdit(user)}
                    >
                      {t('users.edit')}
                    </Button>
                    <Button
                      size="sm"
                      color="danger"
                      variant="light"
                      onPress={() => handleDelete(user.username)}
                    >
                      {t('users.delete')}
                    </Button>
                  </div>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      {/* Create User Modal */}
      <Modal isOpen={isCreateOpen} onClose={onCreateClose}>
        <ModalContent>
          <ModalHeader>{t('users.add')}</ModalHeader>
          <ModalBody>
            <div className="flex flex-col gap-4">
              <Input
                label={t('users.username')}
                value={createForm.username}
                onChange={(e) =>
                  setCreateForm({ ...createForm, username: e.target.value })
                }
              />
              <Input
                label={t('users.password')}
                type="password"
                value={createForm.password}
                onChange={(e) =>
                  setCreateForm({ ...createForm, password: e.target.value })
                }
              />
              <Select
                label={t('users.role')}
                selectedKeys={[createForm.role]}
                onSelectionChange={(keys) =>
                  setCreateForm({
                    ...createForm,
                    role: Array.from(keys)[0] as 'admin' | 'normal',
                  })
                }
              >
                <SelectItem key="admin">{t('users.role_admin')}</SelectItem>
                <SelectItem key="normal">{t('users.role_normal')}</SelectItem>
              </Select>
            </div>
          </ModalBody>
          <ModalFooter>
            <Button variant="light" onPress={onCreateClose}>
              {t('common.cancel')}
            </Button>
            <Button color="primary" onPress={handleCreate}>
              {t('users.add')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Update User Modal */}
      <Modal isOpen={isUpdateOpen} onClose={onUpdateClose}>
        <ModalContent>
          <ModalHeader>{t('users.edit')}</ModalHeader>
          <ModalBody>
            <div className="flex flex-col gap-4">
              <Input
                label={t('users.username')}
                value={updateForm.username}
                isReadOnly
              />
              <Input
                label={t('users.password')}
                type="password"
                placeholder={t('users.password_placeholder')}
                value={updateForm.password || ''}
                onChange={(e) =>
                  setUpdateForm({ ...updateForm, password: e.target.value })
                }
              />
              <Select
                label={t('users.role')}
                selectedKeys={[updateForm.role || '']}
                onSelectionChange={(keys) =>
                  setUpdateForm({
                    ...updateForm,
                    role: Array.from(keys)[0] as 'admin' | 'normal',
                  })
                }
              >
                <SelectItem key="admin">{t('users.role_admin')}</SelectItem>
                <SelectItem key="normal">{t('users.role_normal')}</SelectItem>
              </Select>
            </div>
          </ModalBody>
          <ModalFooter>
            <Button variant="light" onPress={onUpdateClose}>
              {t('common.cancel')}
            </Button>
            <Button color="primary" onPress={handleUpdate}>
              {t('common.save')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </div>
  );
}
