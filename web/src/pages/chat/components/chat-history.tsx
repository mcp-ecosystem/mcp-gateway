import { Card, CardBody, Button } from '@heroui/react';
import { Icon } from '@iconify/react';
import React from 'react';
import {toast} from "react-hot-toast";
import { useNavigate } from 'react-router-dom';

import { getChatSessions } from '../../../services/api';
import { wsService } from '../../../services/websocket';

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
  const navigate = useNavigate();
  const [sessions, setSessions] = React.useState<Session[]>([]);
  const [loading, setLoading] = React.useState(true);
  const loadedRef = React.useRef(false);

  React.useEffect(() => {
    // Skip if we've already loaded sessions
    if (loadedRef.current) {
      return;
    }

    fetchSessions();
    loadedRef.current = true;
  }, []);

  const fetchSessions = async () => {
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
      toast.error(`Failed to fetch chat history: ${error}`, {
        duration: 3000,
        position: 'bottom-right',
      });
      setSessions([]); // Set empty array on error
    } finally {
      setLoading(false);
    }
  };

  const handleNewChat = () => {
    wsService.cleanup();
    const newSessionId = wsService.getSessionId();
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

  if (isCollapsed) return null;
  return (
    <Card className="w-64 relative group bg-card transition-all duration-200">
      {/* Resize bar 仅在展开时显示 */}
      <button
        type="button"
        aria-label="Resize chat history"
        className="absolute right-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-primary/20 transition-colors"
        onMouseDown={(e: React.MouseEvent) => {
          const startX = e.pageX;
          const startWidth = e.currentTarget.parentElement?.offsetWidth || 0;
          const handleMouseMove = (e: MouseEvent) => {
            const delta = e.pageX - startX;
            const newWidth = Math.max(200, Math.min(400, startWidth + delta));
            const parent = (e.target as HTMLElement).parentElement;
            if (parent) {
              parent.style.width = `${newWidth}px`;
            }
          };
          const handleMouseUp = () => {
            document.removeEventListener('mousemove', handleMouseMove);
            document.removeEventListener('mouseup', handleMouseUp);
          };
          document.addEventListener('mousemove', handleMouseMove);
          document.addEventListener('mouseup', handleMouseUp);
        }}
        onKeyDown={(e: React.KeyboardEvent) => {
          if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
            e.preventDefault();
            const parent = e.currentTarget.parentElement;
            if (parent) {
              const currentWidth = parent.offsetWidth;
              const delta = e.key === 'ArrowLeft' ? -10 : 10;
              const newWidth = Math.max(200, Math.min(400, currentWidth + delta));
              parent.style.width = `${newWidth}px`;
            }
          }
        }}
      />
      <CardBody className="p-0">
        <div className="p-4">
          <Button
            color="primary"
            className="w-full"
            startContent={<Icon icon="lucide:plus" />}
            onPress={handleNewChat}
          >
            New Chat
          </Button>
        </div>
        <div className="space-y-1 px-4">
          {loading ? (
            <div className="p-4 text-center text-default-500">Loading...</div>
          ) : !sessions || sessions.length === 0 ? (
            <div className="p-4 text-center text-default-500">No chat history</div>
          ) : (
            sessions.map((session) => (
              <Button
                key={session.id}
                variant="light"
                className={`w-full justify-start px-4 py-2 ${
                  selectedChat === session.id ? 'bg-primary-100' : ''
                }`}
                onPress={() => handleSessionSelect(session.id)}
              >
                <div className="flex flex-col items-start">
                  <span className="text-sm font-medium truncate max-w-full">
                    {session.title || 'Untitled Chat'}
                  </span>
                  <span className="text-xs text-default-400">
                    {formatDate(session.createdAt)}
                  </span>
                </div>
              </Button>
            ))
          )}
        </div>
      </CardBody>
    </Card>
  );
}
