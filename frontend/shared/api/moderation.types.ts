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
  eventType: string;
  contentSnapshot: string;
  contentHash: string;
  snapshotTruncated: boolean;
  reviewStatus: string;
  reviewedBy: number;
  reviewedAt: string | null;
  reviewNote: string;
  disposition: string;
  dispositionAppliedAt: string | null;
  dispositionReleasedAt: string | null;
  dispositionReleasedBy: number;
  createdAt: string;
  updatedAt: string;
};

export type ModerationReviewRequest = {
  status: "pending" | "false_positive" | "confirmed" | "resolved";
  note?: string;
};
