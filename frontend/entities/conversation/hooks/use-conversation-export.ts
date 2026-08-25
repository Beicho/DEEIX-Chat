"use client";

import * as React from "react";
import { toast } from "sonner";

import { downloadConversationExport } from "@/entities/conversation/lib/conversation-export";
import { exportConversation } from "@/shared/api/conversation";
import { resolveAccessToken } from "@/shared/auth/resolve-access-token";

type UseConversationExportOptions = {
  successMessage: string;
  failureMessage: string;
  format?: "json" | "markdown" | "image";
  action?: "download" | "copy";
  imageLabels?: ConversationImageExportLabels;
};

export function useConversationExport({
  successMessage,
  failureMessage,
}: UseConversationExportOptions) {
  return React.useCallback(
    async (conversationPublicID: string) => {
      const token = await resolveAccessToken();
      if (!token) {
        return;
      }

      try {
        const data = await exportConversation(token, conversationPublicID);
        if (format === "image") {
          if (!imageLabels) {
            return;
          }
          await downloadConversationImageExport(data, imageLabels);
        } else if (format === "markdown") {
          if (action === "copy") {
            await copyConversationMarkdownExport(data);
          } else {
            downloadConversationMarkdownExport(data);
          }
        } else {
          downloadConversationExport(data);
        }
        toast.success(successMessage);
      } catch (error) {
        toast.error(failureMessage, {
          description: format === "image" ? undefined : error instanceof Error ? error.message : undefined,
        });
      }
    },
    [action, failureMessage, format, imageLabels, successMessage],
  );
}

export const useConversationExportAction = useChatConversationExport;
