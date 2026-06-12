export type ModerationEventDTO = {
  id: number;
  userID: number;
  conversationID: number;
  messageID: number;
  runID: string;
  direction: string;
  action: string;
  model: string;
  score: number;
  threshold: number;
  flagged: boolean;
  categoriesJSON: string;
  reason: string;
  createdAt: string;
  updatedAt: string;
};
