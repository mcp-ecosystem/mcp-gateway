import { Avatar, Button, Accordion, AccordionItem } from "@heroui/react";
import { Icon } from "@iconify/react";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import ReactMarkdown from 'react-markdown';
import rehypeHighlight from "rehype-highlight";
import rehypeKatex from 'rehype-katex';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import 'katex/dist/katex.min.css';
import 'highlight.js/styles/github.css';

import { mcpService } from "../../../services/mcp";
import { wsService } from "../../../services/websocket";
import {Message, ToolCall, ToolResult} from "../../../types/message";
import { toast } from '../../../utils/toast';
import { ChatContext } from "../chat-context";

interface ChatMessageProps {
  message: Message;
}

export function ChatMessage({ message }: ChatMessageProps) {
  const { t } = useTranslation();
  const isBot = message.sender === 'bot';
  const { messages } = useContext(ChatContext);

  const findToolResult = (toolId: string): ToolResult | undefined => {
    return messages.find((m: Message) => m.toolResult?.toolCallId === toolId)?.toolResult;
  };

  const handleRunTool = async (tool: ToolCall) => {
    try {
      if (!tool?.function?.name) {
        toast.error(t('errors.invalid_tool_name'), {
          duration: 3000,
        });
        return;
      }

      // 解析 serverName:toolName 格式
      const [serverName, toolName] = tool.function.name.split(':');
      if (!serverName || !toolName) {
        toast.error(t('errors.invalid_tool_name'), {
          duration: 3000,
        });
        return;
      }

      const sessionId = mcpService.getSessionId(serverName);

      if (!sessionId) {
        toast.error(t('errors.server_not_connected', { server: serverName }), {
          duration: 3000,
        });
        return;
      }

      // 解析 arguments 字符串为对象
      const args = JSON.parse(tool.function.arguments);
      const result = await mcpService.callTool(serverName, toolName, args);

      // 显示工具调用结果
      toast.success(t('chat.tool_call_success', { result }), {
        duration: 3000,
      });

      // 将工具调用结果作为新消息发送
      await wsService.sendToolResult(tool.function.name, tool.id, result);
    } catch (error) {
      toast.error(t('errors.tool_call_failed', { error: (error as Error).message }), {
        duration: 3000,
      });
    }
  };

  return (
    <div className={`flex gap-3 mb-4 ${isBot ? 'flex-row' : 'flex-row-reverse'}`}>
      <Avatar
        size="sm"
        src={isBot ? "https://img.heroui.chat/image/avatar?w=32&h=32&u=1" : undefined}
        name={isBot ? "MCP" : t('chat.you')}
      />
      <div
        className={`px-4 py-2 rounded-lg max-w-[80%] ${
          isBot ? 'bg-secondary' : 'bg-primary text-primary-foreground'
        }`}
      >
        <div className="prose prose-sm dark:prose-invert max-w-none">
          <ReactMarkdown
            remarkPlugins={[remarkGfm, remarkMath]}
            rehypePlugins={[rehypeHighlight, rehypeKatex]}
            components={{
              code({className, children, ...props}) {
                const match = /language-(\w+)/.exec(className || '');
                return match ? (
                  <code className={className} {...props}>
                    {children}
                  </code>
                ) : (
                  <code className="bg-gray-100 dark:bg-gray-800 rounded px-1" {...props}>
                    {children}
                  </code>
                );
              }
            }}
          >
            {message.content}
          </ReactMarkdown>
        </div>
        {message.isStreaming && (
          <span className="inline-block w-2 h-4 ml-1 bg-current animate-pulse" />
        )}
        {message.toolCalls?.map((tool, index) => {
          const toolResult = findToolResult(tool.id);
          return tool?.function?.name ? (
            <div key={index} className="mt-2 p-2 border rounded bg-card">
              <div className="font-medium mb-2">{tool.function.name}</div>
              <Accordion selectionMode="multiple">
                <AccordionItem
                  key={`${tool.id}-args`}
                  title={t('chat.arguments')}
                  className="px-0"
                >
                  <pre className="text-sm p-2 bg-secondary rounded overflow-auto">
                    {JSON.stringify(JSON.parse(tool.function.arguments), null, 2)}
                  </pre>
                </AccordionItem>
                {toolResult ? (
                  <AccordionItem
                    key={`${tool.id}-result`}
                    title={t('chat.result')}
                    className="px-0"
                  >
                    <pre className="text-sm p-2 bg-secondary rounded overflow-auto">
                      {(() => {
                        try {
                          return JSON.stringify(JSON.parse(toolResult.result), null, 2);
                        } catch {
                          return toolResult.result;
                        }
                      })()}
                    </pre>
                  </AccordionItem>
                ) : null}
              </Accordion>
              {!toolResult && (
                <Button
                  size="sm"
                  color="primary"
                  className="mt-2"
                  startContent={<Icon icon="lucide:play" />}
                  onPress={() => handleRunTool(tool)}
                >
                  {t('chat.run_tool')}
                </Button>
              )}
            </div>
          ) : null;
        })}
      </div>
    </div>
  );
}
