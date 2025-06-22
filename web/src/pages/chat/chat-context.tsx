import React from 'react';

import { Message } from '@/types/message';

interface ChatContextType {
  messages: Message[];
}

export const ChatContext = React.createContext<ChatContextType>({
  messages: [],
});

export function ChatProvider({ children, messages }: { children: React.ReactNode; messages: Message[] }) {
  return (
    <ChatContext.Provider value={{ messages }}>
      {children}
    </ChatContext.Provider>
  );
} 