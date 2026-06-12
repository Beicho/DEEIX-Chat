export type ProjectKnowledgeDocumentDTO = {
  fileID: string;
  fileName: string;
  fileSize: number;
  fileCategory: string;
  extractStatus: string;
  embedStatus: string;
  indexStatus: "pending" | "indexing" | "ready" | "failed" | "stale" | string;
  status: string;
  createdAt: string;
  updatedAt: string;
};

export type AddProjectKnowledgeDocumentsRequest = {
  fileIDs: string[];
};

export type MarkProjectKnowledgeIndexRequest = {
  indexStatus: "pending" | "indexing" | "ready" | "failed" | "stale";
};

export type DeleteProjectKnowledgeDocumentResult = {
  deleted: boolean;
  fileID: string;
};
