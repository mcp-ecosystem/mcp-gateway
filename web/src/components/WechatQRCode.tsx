import {
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Button,
  Modal,
} from "@heroui/react";
import { useTranslation } from 'react-i18next';

import wechatQrcode from '/wechat-qrcode.png';

interface WechatQRCodeProps {
  isOpen: boolean;
  onOpenChange: (isOpen: boolean) => void;
}

export function WechatQRCode({ isOpen, onOpenChange }: WechatQRCodeProps) {
  const { t } = useTranslation();
  
  return (
    <Modal isOpen={isOpen} onOpenChange={onOpenChange} size="sm">
      <ModalContent>
        <ModalHeader>{t('common.join_wechat')}</ModalHeader>
        <ModalBody>
          <div className="flex flex-col items-center justify-center">
            <img
              src={wechatQrcode}
              alt="WeChat QR Code"
              className="w-64 h-64 object-contain"
            />
            <p className="mt-4 text-center text-muted-foreground">
              {t('common.scan_qrcode')}
            </p>
            <p className="mt-4 text-center text-muted-foreground">
              {t('common.add_wechat_note')}
            </p>
          </div>
        </ModalBody>
        <ModalFooter>
          <Button color="primary" onPress={() => onOpenChange(false)}>
            {t('common.close')}
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}
