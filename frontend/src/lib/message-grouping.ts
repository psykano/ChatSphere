import type { ChatMessage } from "@/hooks/use-chat";

function isSystemMessage(msg: ChatMessage): boolean {
  return msg.type === "system" || msg.type === "join" || msg.type === "leave";
}

export function isSameUserAsPrevious(messages: ChatMessage[], index: number): boolean {
  if (index === 0) return false;
  const msg = messages[index];
  const prev = messages[index - 1];
  if (isSystemMessage(msg) || isSystemMessage(prev)) return false;
  return prev.user_id === msg.user_id;
}
