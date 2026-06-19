import { authedRequest } from "@/shared/api/authed-client";
import { readAccessToken } from "@/shared/auth/session";
import type {
  ArenaVoteRequest,
  ArenaVoteResult,
  ArenaLeaderboardEntry,
} from "../types/arena";

export async function submitArenaVote(
  conversationID: string,
  payload: ArenaVoteRequest
): Promise<ArenaVoteResult> {
  const accessToken = readAccessToken();
  if (!accessToken) throw new Error("Not authenticated");

  const response = await authedRequest<ArenaVoteResult | { vote: ArenaVoteResult }>(
    `/api/v1/conversations/${conversationID}/arena-vote`,
    { method: "POST", accessToken, body: payload }
  );
  return "vote" in response ? response.vote : response;
}

export async function getArenaLeaderboard(): Promise<ArenaLeaderboardEntry[]> {
  const accessToken = readAccessToken();
  if (!accessToken) throw new Error("Not authenticated");

  const response = await authedRequest<{ leaderboard: ArenaLeaderboardEntry[] }>(
    "/api/v1/admin/arena/leaderboard",
    { accessToken }
  );
  return response.leaderboard;
}
