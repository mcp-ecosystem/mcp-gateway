import {
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Button,
  Spinner,
} from "@heroui/react";
import { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import ReactMarkdown from 'react-markdown';
import rehypeHighlight from "rehype-highlight";
import rehypeKatex from 'rehype-katex';
import remarkGfm from 'remark-gfm';

import { AccessibleModal } from "./AccessibleModal";

interface ChangelogModalProps {
  isOpen: boolean;
  onOpenChange: (isOpen: boolean) => void;
  version: string;
}

export function ChangelogModal({ isOpen, onOpenChange, version }: ChangelogModalProps) {
  const { t } = useTranslation();
  const [changelogContent, setChangelogContent] = useState<string>('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  const loadChangelog = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await fetch(
        `https://raw.githubusercontent.com/mcp-ecosystem/mcp-gateway/main/changelog/v${version}.md`
      );
      
      if (!response.ok) {
        throw new Error(`Failed to load changelog: ${response.status}`);
      }
      
      const content = await response.text();
      setChangelogContent(content);
    } catch (err) {
      console.error('Error loading changelog:', err);
      setError(t('errors.load_changelog', { error: (err as Error).message }));
    } finally {
      setIsLoading(false);
    }
  }, [version, t]);
  
  useEffect(() => {
    if (isOpen) {
      loadChangelog();
    }
  }, [isOpen, loadChangelog]);
  
  return (
    <AccessibleModal isOpen={isOpen} onOpenChange={onOpenChange} size="2xl" scrollBehavior="inside">
      <ModalContent>
        <ModalHeader>{t('common.changelog')}: v{version}</ModalHeader>
        <ModalBody>
          {isLoading ? (
            <div className="flex justify-center items-center py-10">
              <Spinner size="lg" />
            </div>
          ) : error ? (
            <div className="text-danger py-4">
              {error}
            </div>
          ) : (
            <div className="prose dark:prose-invert max-w-none">
              <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                rehypePlugins={[rehypeHighlight, rehypeKatex]}
              >
                {changelogContent}
              </ReactMarkdown>
            </div>
          )}
        </ModalBody>
        <ModalFooter>
          <Button color="primary" onPress={() => onOpenChange(false)}>
            {t('common.close')}
          </Button>
        </ModalFooter>
      </ModalContent>
    </AccessibleModal>
  );
}
