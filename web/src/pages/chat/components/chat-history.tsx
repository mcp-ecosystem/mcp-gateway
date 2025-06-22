
import { Card, CardBody, Button, Dropdown, DropdownTrigger, DropdownMenu, DropdownItem, Modal, ModalContent, ModalHeader, ModalBody, ModalFooter, Input } from '@heroui/react';
import React from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import LocalIcon from '@/components/LocalIcon';
import { getChatSessions, deleteChatSession, updateChatSessionTitle } from '@/services/api';
import { toast } from '@/utils/toast';

interface ChatHistoryProps {
  selectedChat: string | null;
  onSelectChat: (id: string) => void;
  isCollapsed: boolean;
}

interface Session {
  id: string;
  createdAt: string;
  title: string;
}

export function ChatHistory({ selectedChat, onSelectChat, isCollapsed }: ChatHistoryProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [sessions, setSessions] = React.useState<Session[]>([]);
  const [loading, setLoading] = React.useState(true);
  const loadedRef = React.useRef(false);
  const [isRenameModalOpen, setIsRenameModalOpen] = React.useState(false);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = React.useState(false);
  const [selectedSession, setSelectedSession] = React.useState<Session | null>(null);
  const [newTitle, setNewTitle] = React.useState('');

  const fetchSessions = React.useCallback(async () => {
    try {
      const data = await getChatSessions();
      // Ensure data is an array and each session has required properties
      const validSessions = Array.isArray(data)
        ? data.filter(session =>
          session &&
          typeof session.id === 'string' &&
          typeof session.createdAt === 'string' &&
          typeof session.title === 'string'
        )
        : [];
      setSessions(validSessions);
    } catch (error) {
      toast.error(t('errors.fetch_chat_history', { error }), {
        duration: 3000,
      });
      setSessions([]); // Set empty array on error
    } finally {
      setLoading(false);
    }
  }, [t]);

  React.useEffect(() => {
    // Skip if we've already loaded sessions
    if (loadedRef.current) {
      return;
    }

    fetchSessions();
    loadedRef.current = true;
  }, [fetchSessions]);

  const handleNewChat = () => {
    // Generate new session ID and navigate to new chat
    const newSessionId = globalThis.crypto?.randomUUID?.() || 
      'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
        const r = Math.random() * 16 | 0;
        const v = c === 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
      });
    navigate(`/chat/${newSessionId}`);
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    });
  };

  const handleSessionSelect = (sessionId: string) => {
    onSelectChat(sessionId);
    navigate(`/chat/${sessionId}`);
  };

  const handleRename = async () => {
    if (!selectedSession || !newTitle) {
      return;
    }

    try {
      await updateChatSessionTitle(selectedSession.id, newTitle);
      toast.success(t('chat.rename_success'));
      fetchSessions();
      setIsRenameModalOpen(false);
      setSelectedSession(null);
      setNewTitle('');
    } catch {
      toast.error(t('chat.rename_failed'));
    }
  };

  const handleDelete = async () => {
    if (!selectedSession) {
      return;
    }

    try {
      await deleteChatSession(selectedSession.id);
      toast.success(t('chat.delete_success'));
      fetchSessions();
      setIsDeleteModalOpen(false);
      setSelectedSession(null);
    } catch {
      toast.error(t('chat.delete_failed'));
    }
  };

  if (isCollapsed) return null;

  return (
    <Card className="w-64">
      <CardBody className="p-0">
        <div className="p-4">
          <Button
            color="primary"
            className="w-full"
            startContent={<LocalIcon icon="lucide:plus" />}
            onPress={handleNewChat}
          >
            {t('chat.new_chat')}
          </Button>
        </div>
        <div className="space-y-1 px-4">
          {loading ? (
            <div className="p-4 text-center text-default-500">{t('common.loading')}</div>
          ) : !sessions || sessions.length === 0 ? (
            <div className="p-4 text-center text-default-500">{t('chat.no_history')}</div>
          ) : (
            sessions.map((session) => (
              <div
                key={session.id}
                className={`group flex items-center justify-between px-4 py-2 rounded-lg cursor-pointer hover:bg-default-100 transition-all ${
                  selectedChat === session.id ? 'bg-primary-100' : ''
                }`}
                onClick={() => handleSessionSelect(session.id)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    handleSessionSelect(session.id);
                  }
                }}
                role="button"
                tabIndex={0}
              >
                <div className="flex flex-col items-start flex-1 min-w-0 mr-2 relative">
                  <span className="text-sm font-medium truncate w-full">
                    {session.title || t('chat.untitled')}
                  </span>
                  <span className="text-xs text-default-400 truncate w-full">
                    {formatDate(session.createdAt)}
                  </span>
                  <div className="absolute inset-y-0 right-0 w-16 bg-gradient-to-l from-default-100 to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
                </div>
                <div className="flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity absolute right-6">
                  <Dropdown className='min-w-0 w-fit'>
                    <DropdownTrigger>
                      <Button
                        isIconOnly
                        variant="light"
                        size="sm"
                        className="ml-2"
                      >
                        <LocalIcon icon="lucide:more-vertical" />
                      </Button>
                    </DropdownTrigger>
                    <DropdownMenu
                      aria-label="Session actions"
                      onAction={(key) => {
                        setSelectedSession(session);
                        if (key === 'rename') {
                          setNewTitle(session.title);
                          setIsRenameModalOpen(true);
                        } else if (key === 'delete') {
                          setIsDeleteModalOpen(true);
                        }
                      }}
                    >
                      <DropdownItem key="rename" startContent={<LocalIcon icon="lucide:edit" />}>
                        {t('chat.rename')}
                      </DropdownItem>
                      <DropdownItem
                        key="delete"
                        className="text-danger"
                        color="danger"
                        startContent={<LocalIcon icon="lucide:trash" />}
                      >
                        {t('chat.delete')}
                      </DropdownItem>
                    </DropdownMenu>
                  </Dropdown>
                </div>
              </div>
            ))
          )}
        </div>
      </CardBody>

      {/* Rename Modal */}
      <Modal isOpen={isRenameModalOpen} onClose={() => setIsRenameModalOpen(false)}>
        <ModalContent>
          <ModalHeader>{t('chat.rename_session')}</ModalHeader>
          <ModalBody>
            <Input
              value={newTitle}
              onChange={(e) => setNewTitle(e.target.value)}
              placeholder={t('chat.session_title_placeholder')}
            />
          </ModalBody>
          <ModalFooter>
            <Button variant="light" onPress={() => setIsRenameModalOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button color="primary" onPress={handleRename}>
              {t('common.save')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal isOpen={isDeleteModalOpen} onClose={() => setIsDeleteModalOpen(false)}>
        <ModalContent>
          <ModalHeader>{t('chat.delete_session')}</ModalHeader>
          <ModalBody>
            {t('chat.delete_session_confirm')}
          </ModalBody>
          <ModalFooter>
            <Button variant="light" onPress={() => setIsDeleteModalOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button color="danger" onPress={handleDelete}>
              {t('common.delete')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Card>
  );
}
