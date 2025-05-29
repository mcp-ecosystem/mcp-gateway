import {Button, Card, CardBody, Divider, Input, Select, SelectItem, Tab, Tabs} from '@heroui/react';
import {Icon} from '@iconify/react';
import yaml from 'js-yaml';
import React from 'react';
import {useTranslation} from 'react-i18next';
import {useNavigate, useParams} from 'react-router-dom';
import {v4 as uuidv4} from 'uuid';

import {getChatMessages, getMCPServers} from '../../services/api';
import {mcpService} from '../../services/mcp';
import {WebSocketMessage, wsService} from '../../services/websocket';
import type { Gateway } from '../../types/gateway';
import {Tool} from '../../types/mcp';
import {Message as MessageType, ToolCall, ToolResult, BackendMessage} from '../../types/message';
import {toast} from '../../utils/toast';

import {ChatProvider} from './chat-context';
import {ChatHistory} from './components/chat-history';
import {ChatMessage} from './components/chat-message';

export function ChatInterface() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { sessionId } = useParams();
  const messagesEndRef = React.useRef<HTMLDivElement>(null);
  const messagesContainerRef = React.useRef<HTMLDivElement>(null);
  const [messages, setMessages] = React.useState<MessageType[]>([]);
  const [input, setInput] = React.useState('');
  const [selectedChat, setSelectedChat] = React.useState<string | null>(null);
  const [activeServices, setActiveServices] = React.useState<string[]>([]);
  const [mcpServers, setMcpServers] = React.useState<Gateway[]>([]);
  const [tools, setTools] = React.useState<Record<string, Tool[]>>({});
  const [isNearBottom, setIsNearBottom] = React.useState(true);
  const [page, setPage] = React.useState(1);
  const [hasMore, setHasMore] = React.useState(true);
  const [loading, setLoading] = React.useState(false);
  const [lastScrollTop, setLastScrollTop] = React.useState(0);
  const [isHistoryCollapsed, setIsHistoryCollapsed] = React.useState(false);
  const [isToolsCollapsed, setIsToolsCollapsed] = React.useState(false);

  // Parse configuration
  const parseConfig = React.useCallback((config: string) => {
    try {
      return yaml.load(config) as Gateway;
    } catch (error) {
      toast.error(t('errors.parse_config', { error: error instanceof Error ? error.message : 'Unknown error' }));
      return undefined;
    }
  }, [t]);

  // Get MCP servers list and parse configuration
  React.useEffect(() => {
    const fetchMCPServers = async () => {
      try {
        const servers = await getMCPServers();
        const parsedServers = servers.map((server: { config: string; }) => ({
          ...server,
          parsedConfig: parseConfig(server.config)
        }));
        setMcpServers(parsedServers);
      } catch {
        toast.error(t('errors.fetch_mcp_servers'), {
          duration: 3000,
        });
      }
    };

    void fetchMCPServers();
  }, [t, parseConfig]);

  // When selected server changes, reload tool list
  React.useEffect(() => {
    const loadToolsForActiveServers = async () => {
      for (const serverName of activeServices) {
        const server = mcpServers.find((s: Gateway) => s.name === serverName);
        if (!server) continue;

        for (const router of server.routers || []) {
          try {
            // First establish connection
            await mcpService.connect({
              name: serverName,
              prefix: router.prefix,
              onError: (error) => {
                toast.error(t('errors.mcp_server_error', { server: serverName, error: error.message }), {
                  duration: 3000,
                });
              },
              onNotification: (notification) => {
                toast.success(t('chat.notification_received', { server: serverName, message: notification }), {
                  duration: 3000,
                });
              }
            });

            // Then get tool list
            const toolsList = await mcpService.getTools(serverName);
            setTools(prev => ({
              ...prev,
              [serverName]: toolsList
            }));
          } catch (error) {
            toast.error(t('errors.fetch_tools', { error }), {
              duration: 3000,
            });
          }
        }
      }
    };

    if (activeServices.length > 0) {
      void loadToolsForActiveServers();
    }
  }, [activeServices, mcpServers, t, parseConfig]);

  const loadMessages = React.useCallback(async (sessionId: string, pageNum: number = 1) => {
    setLoading(true);
    try {
      const data = await getChatMessages(sessionId, pageNum);

      // Check if data exists
      if (!data) {
        const welcomeMessage: MessageType = {
          id: uuidv4(),
          session_id: sessionId,
          content: t('chat.welcome_message'),
          sender: 'bot',
          timestamp: new Date().toISOString(),
        };
        setMessages([welcomeMessage]);
        return;
      }

      // Convert backend message format to frontend format
      const newMessages = data.map((msg: BackendMessage) => ({
        id: msg.id,
        session_id: sessionId,
        content: msg.content,
        sender: msg.sender as 'user' | 'bot',
        timestamp: msg.timestamp,
        toolCalls: msg.toolCalls ? JSON.parse(msg.toolCalls) as ToolCall[] : undefined,
        toolResult: msg.toolResult ? JSON.parse(msg.toolResult) as ToolResult : undefined,
      }));

      if (pageNum === 1) {
        setMessages(newMessages);
      } else {
        setMessages(prev => [...newMessages, ...prev]);
      }

      // Since we don't have hasMore in the response, we'll assume there are more messages
      // if we got a full page of messages
      setHasMore(newMessages.length === 20);
      setPage(pageNum);
    } catch (error) {
      toast.error(t('errors.load_messages', { error: error instanceof Error ? error.message : 'Unknown error' }), {
        duration: 3000,
      });
      setMessages([]);
      setHasMore(false);
    } finally {
      setLoading(false);
    }
  }, [t]);

  React.useEffect(() => {
    if (!sessionId) {
      // If no session ID in URL, create a new one and redirect
      wsService.cleanup();
      const newSessionId = wsService.getSessionId();
      navigate(`/chat/${newSessionId}`);
      return;
    }

    if (!sessionId) return;

    // Clear old messages first
    setMessages([]);
    setPage(1);
    setHasMore(true);
    setSelectedChat(sessionId);

    // Set up message handler for regular messages
    const unsubscribe = wsService.onMessage((message: WebSocketMessage) => {
      if (message.type === 'stream') return;

      switch (message.type) {
        case "message":
          setMessages(prev => {
            // Check if message already exists to prevent duplicates
            if (prev.some(m => m.id === message.id)) {
              return prev;
            }

            const newMessage: MessageType = {
              id: message.id,
              session_id: sessionId,
              content: message.content,
              sender: message.sender,
              timestamp: new Date(message.timestamp).toISOString(),
              toolCalls: message.toolCalls
            };
            return [...prev, newMessage];
          });
          break
        case "tool_call":
          setMessages(prev => {
            // Check if message already exists to prevent duplicates
            if (prev.some(m => m.id === message.id)) {
              return prev;
            }

            const newMessage: MessageType = {
              id: message.id,
              session_id: sessionId,
              content: '',
              sender: message.sender,
              timestamp: new Date(message.timestamp).toISOString(),
              toolCalls: message.toolCalls
            };
            return [...prev, newMessage];
          });
          break
      }
    });

    // Set up stream handler
    const unsubscribeStream = wsService.onStream((chunk: string) => {
      setMessages(prev => {
        const lastMessage = prev[prev.length - 1];
        // If the last message is from bot and is streaming, append to it
        if (lastMessage && lastMessage.sender === 'bot' && lastMessage.isStreaming) {
          const updatedMessages = [...prev];
          updatedMessages[prev.length - 1] = {
            ...lastMessage,
            content: lastMessage.content + chunk,
            isStreaming: true
          };
          return updatedMessages;
        }
        // If no bot message exists or last message is from user, create new one
        const newMessage: MessageType = {
          id: uuidv4(),
          session_id: sessionId,
          content: chunk,
          sender: 'bot',
          timestamp: new Date().toISOString(),
          isStreaming: true
        };
        return [...prev, newMessage];
      });
    });

    // Switch to new session and load history
    wsService.switchChat(sessionId).then(async () => {
      // Load chat history
      await loadMessages(sessionId);
    });

    // Cleanup on unmount or session change
    return () => {
      unsubscribe();
      unsubscribeStream();
    };
  }, [sessionId, navigate, loadMessages]);

  // Add scroll position check and load more messages when scrolling up
  React.useEffect(() => {
    const container = messagesContainerRef.current;
    if (!container) return;

    const handleScroll = () => {
      const { scrollTop, scrollHeight, clientHeight } = container;
      const isNearBottom = scrollHeight - scrollTop - clientHeight < 100;
      setIsNearBottom(isNearBottom);

      // Only load more messages when user actively scrolls up and not loading
      if (scrollTop < lastScrollTop && scrollTop < 100 && hasMore && !loading && sessionId) {
        loadMessages(sessionId, page + 1);
      }
      setLastScrollTop(scrollTop);
    };

    container.addEventListener('scroll', handleScroll);
    return () => container.removeEventListener('scroll', handleScroll);
  }, [sessionId, page, hasMore, loading, lastScrollTop, loadMessages]);

  // Modify auto-scroll effect to only scroll when appropriate
  React.useEffect(() => {
    if (isNearBottom) {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages, isNearBottom]);

  const handleSend = async () => {
    if (!input.trim()) return;

    const newMessage: MessageType = {
      id: uuidv4(),
      session_id: sessionId!,
      content: input,
      sender: 'user',
      timestamp: new Date().toISOString(),
    };

    setMessages(prev => [...prev, newMessage]);

    // Convert active tools to the required format
    const activeTools = Object.entries(tools)
      .filter(([serverName]) => activeServices.includes(serverName))
      .flatMap(([serverName, serverTools]) => serverTools.map(tool => ({
        name: `${serverName}:${tool.name}`,
        description: tool.description || tool.name, // Fallback to name if description is not provided
        parameters: {
          properties: tool.inputSchema.properties || {},
          required: tool.inputSchema.required as string[] || []
        }
      })));

    await wsService.sendMessage(input, activeTools.length > 0 ? activeTools : undefined);
    setInput('');
  };

  return (
    <div className="flex h-[calc(100vh-8rem)]">
      <ChatHistory
        selectedChat={selectedChat}
        onSelectChat={(id) => {
          setSelectedChat(id);
          navigate(`/chat/${id}`);
        }}
        isCollapsed={isHistoryCollapsed}
      />
      <div className={isHistoryCollapsed ? "flex-1 ml-2" : "flex-1 ml-4"}>
        <Card className="h-full bg-card">
          <CardBody className="p-0 h-full flex flex-col">
            <div
              ref={messagesContainerRef}
              className="flex-1 overflow-auto p-4 scrollbar-hide"
            >
              {loading && page > 1 && (
                <div className="text-center text-default-500 py-2">
                  Loading more messages...
                </div>
              )}
              <ChatProvider messages={messages}>
                {messages
                  .filter(message => !(message.sender === 'user' && !message.content && message.toolResult))
                  .map((message) => (
                    <ChatMessage key={message.id} message={message} />
                  ))}
              </ChatProvider>
              <div ref={messagesEndRef} />
            </div>
            <Divider />
            <div className="p-4 flex flex-col gap-4">
              <div className="flex items-center gap-2 mb-2">
                <Button
                  isIconOnly
                  variant="light"
                  aria-label={isHistoryCollapsed ? "Expand chat history" : "Collapse chat history"}
                  onPress={() => setIsHistoryCollapsed(v => !v)}
                  className="mr-1"
                >
                  <Icon icon={isHistoryCollapsed ? "ri:menu-unfold-line" : "ri:menu-unfold-2-line"} className="text-lg" />
                </Button>
                <Button
                  isIconOnly
                  variant="light"
                  aria-label={isToolsCollapsed ? "Expand tools area" : "Collapse tools area"}
                  onPress={() => setIsToolsCollapsed(v => !v)}
                  className="mr-2"
                >
                  <Icon icon={isToolsCollapsed ? "ic:round-unfold-more" : "ic:round-unfold-less"} className="text-lg" />
                </Button>
                <Input
                  value={input}
                  onValueChange={setInput}
                  placeholder="Type your message..."
                  onKeyPress={(e) => e.key === 'Enter' && handleSend()}
                  endContent={
                    <Button
                      isIconOnly
                      color="primary"
                      variant="light"
                      onPress={handleSend}
                    >
                      <Icon icon="lucide:send" className="text-lg" />
                    </Button>
                  }
                />
              </div>
              {!isToolsCollapsed && (
                <>
                  <Select
                    label="MCP Servers"
                    selectionMode="multiple"
                    selectedKeys={activeServices}
                    onSelectionChange={(keys) => setActiveServices(Array.from(keys) as string[])}
                  >
                    {mcpServers.map((server) => (
                      <SelectItem key={server.name}>
                        {server.name}
                      </SelectItem>
                    ))}
                  </Select>

                  {activeServices.length > 0 && (
                    <div className="flex flex-col gap-2">
                      <h3 className="text-lg font-semibold">Available Tools</h3>
                      <Tabs aria-label="Server tools">
                        {activeServices.map(serverName => {
                          const serverTools = tools[serverName] || [];
                          return (
                            <Tab key={serverName} title={serverName}>
                              <div className="flex flex-wrap gap-2 mt-2">
                                {serverTools.map(tool => (
                                  <div key={tool.name} className="p-2 border rounded min-w-[200px] flex-1">
                                    <div className="font-medium">{tool.name}</div>
                                    <div className="text-sm text-default-500">{tool.description}</div>
                                  </div>
                                ))}
                              </div>
                            </Tab>
                          );
                        })}
                      </Tabs>
                    </div>
                  )}
                </>
              )}
            </div>
          </CardBody>
        </Card>
      </div>
    </div>
  );
}
