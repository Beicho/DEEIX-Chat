"use client";

import * as React from "react";
import { toast } from "sonner";

import {
  type ConversationImageExportLabels,
  copyConversationMarkdownExport,
  downloadConversationExport,
  downloadConversationImageExport,
  downloadConversationMarkdownExport,
} from "@/features/chat/model/conversation-export";
import { exportConversation } from "@/shared/api/conversation";
import { resolveAccessToken } from "@/shared/auth/resolve-access-token";

type UseChatConversationExportOptions = {
  successMessage: string;
  failureMessage: string;
  format?: "json" | "markdown" | "image";
  action?: "download" | "copy";
  imageLabels?: ConversationImageExportLabels;
};

export function useChatConversationExport({
  successMessage,
  failureMessage,
  format = "json",
  action = "download",
  imageLabels,
}: UseChatConversationExportOptions) {
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
