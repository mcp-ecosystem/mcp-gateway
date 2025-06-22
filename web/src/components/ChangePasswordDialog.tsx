import { Button, Input, ModalContent, ModalHeader, ModalBody, ModalFooter, Modal } from "@heroui/react";
import axios from 'axios';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';


import api from '../services/api';
import { toast } from '../utils/toast';

import LocalIcon from './LocalIcon';

interface ChangePasswordDialogProps {
  isOpen: boolean;
  onOpenChange: () => void;
}

export function ChangePasswordDialog({ isOpen, onOpenChange }: ChangePasswordDialogProps) {
  const { t } = useTranslation();
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async () => {
    if (newPassword !== confirmPassword) {
      toast.error(t('auth.password_mismatch'));
      return;
    }

    setLoading(true);
    try {
      await api.post('/auth/change-password', {
        oldPassword,
        newPassword,
      });
      toast.success(t('auth.password_change_success'));
      onOpenChange();
      // Clear form
      setOldPassword('');
      setNewPassword('');
      setConfirmPassword('');
    } catch (error) {
      if (axios.isAxiosError(error) && error.response?.data?.error) {
        toast.error(error.response.data.error);
      } else {
        toast.error(t('auth.password_change_failed'));
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onOpenChange={onOpenChange}>
      <ModalContent>
        <ModalHeader>{t('auth.change_password')}</ModalHeader>
        <ModalBody>
          <div className="flex flex-col gap-4">
            <Input
              label={t('auth.current_password')}
              type="password"
              placeholder={t('auth.current_password_placeholder')}
              value={oldPassword}
              onChange={(e) => setOldPassword(e.target.value)}
              startContent={<LocalIcon icon="lucide:lock" className="text-default-400" />}
            />
            <Input
              label={t('auth.new_password')}
              type="password"
              placeholder={t('auth.new_password_placeholder')}
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              startContent={<LocalIcon icon="lucide:lock" className="text-default-400" />}
            />
            <Input
              label={t('auth.confirm_password')}
              type="password"
              placeholder={t('auth.confirm_password_placeholder')}
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              startContent={<LocalIcon icon="lucide:lock" className="text-default-400" />}
            />
          </div>
        </ModalBody>
        <ModalFooter>
          <Button color="danger" variant="light" onPress={onOpenChange}>
            {t('common.cancel')}
          </Button>
          <Button color="primary" onPress={handleSubmit} isLoading={loading}>
            {t('auth.confirm_change')}
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}
