"use client";

import * as React from "react";
import { toast } from "sonner";

import {
  copyConversationMarkdownExport,
  downloadConversationExport,
  downloadConversationMarkdownExport,
} from "@/features/chat/model/conversation-export";
import { exportConversation } from "@/shared/api/conversation";
import { resolveAccessToken } from "@/shared/auth/resolve-access-token";

type UseConversationExportActionOptions = {
  successMessage: string;
  failureMessage: string;
  format?: "json" | "markdown";
  action?: "download" | "copy";
};

export function useConversationExportAction({
  successMessage,
  failureMessage,
  format = "json",
  action = "download",
}: UseConversationExportActionOptions) {
  return React.useCallback(
    async (conversationPublicID: string) => {
      const token = await resolveAccessToken();
      if (!token) {
        return;
      }

      try {
        const data = await exportConversation(token, conversationPublicID);
        if (format === "markdown") {
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
          description: error instanceof Error ? error.message : undefined,
        });
      }
    },
    [action, failureMessage, format, successMessage],
  );
}
