import { authedRequest } from "@/shared/api/authed-client";
import { pathParam } from "@/shared/api/http-client";
import type {
  AddProjectKnowledgeDocumentsRequest,
  DeleteProjectKnowledgeDocumentResult,
  MarkProjectKnowledgeIndexRequest,
  ProjectKnowledgeDocumentDTO,
} from "@/shared/api/project-knowledge.types";

export async function listProjectKnowledgeDocuments(
  accessToken: string,
  projectPublicID: string,
): Promise<ProjectKnowledgeDocumentDTO[]> {
  return authedRequest<ProjectKnowledgeDocumentDTO[]>(
    `/api/v1/conversation-projects/${pathParam(projectPublicID)}/documents`,
    {
      accessToken,
    },
    true,
  );
}

export async function addProjectKnowledgeDocuments(
  accessToken: string,
  projectPublicID: string,
  payload: AddProjectKnowledgeDocumentsRequest,
): Promise<ProjectKnowledgeDocumentDTO[]> {
  return authedRequest<ProjectKnowledgeDocumentDTO[]>(
    `/api/v1/conversation-projects/${pathParam(projectPublicID)}/documents`,
    {
      method: "POST",
      accessToken,
      body: payload,
    },
    true,
  );
}

export async function deleteProjectKnowledgeDocument(
  accessToken: string,
  projectPublicID: string,
  fileID: string,
): Promise<DeleteProjectKnowledgeDocumentResult> {
  return authedRequest<DeleteProjectKnowledgeDocumentResult>(
    `/api/v1/conversation-projects/${pathParam(projectPublicID)}/documents/${pathParam(fileID)}`,
    {
      method: "DELETE",
      accessToken,
    },
    true,
  );
}

export async function reindexProjectKnowledgeDocument(
  accessToken: string,
  projectPublicID: string,
  fileID: string,
): Promise<ProjectKnowledgeDocumentDTO> {
  return authedRequest<ProjectKnowledgeDocumentDTO>(
    `/api/v1/conversation-projects/${pathParam(projectPublicID)}/documents/${pathParam(fileID)}/reindex`,
    {
      method: "POST",
      accessToken,
    },
    true,
  );
}

export async function markProjectKnowledgeIndexStatus(
  accessToken: string,
  projectPublicID: string,
  fileID: string,
  payload: MarkProjectKnowledgeIndexRequest,
): Promise<ProjectKnowledgeDocumentDTO> {
  return authedRequest<ProjectKnowledgeDocumentDTO>(
    `/api/v1/conversation-projects/${pathParam(projectPublicID)}/documents/${pathParam(fileID)}/index`,
    {
      method: "PATCH",
      accessToken,
      body: payload,
    },
    true,
  );
}
