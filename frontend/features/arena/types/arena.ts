export interface ArenaModel {
  modelName: string;
  displayName: string;
}

export interface ArenaResponse {
  modelName: string;
  branchPublicID: string;
  content: string;
  status: "streaming" | "complete" | "error";
  error?: string;
}

export interface ArenaVoteRequest {
  messageGroupID: string;
  winnerModel: string;
  blindMode: boolean;
}

export interface ArenaVoteResult {
  messageGroupID: string;
  winnerModel: string;
  alreadyVoted: boolean;
}

export interface ArenaLeaderboardEntry {
  model: string;
  winCount: number;
  totalBattles: number;
  winRate: number;
}
