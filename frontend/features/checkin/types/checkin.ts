export interface CheckInStatus {
  todayClaimed: boolean;
  rewardUSD: number;
  rewardNanousd: number;
  consecutiveDays: number;
  lastCheckInDate: string | null;
  nextCheckInDate: string;
  account?: {
    id: number;
    userID: number;
    balanceUSD: number;
    balanceNanousd: number;
  };
  latestTransaction?: {
    id: number;
    type: string;
    amountUSD: number;
    description: string;
    createdAt: string;
  };
}

export interface CheckInClaim {
  id: number;
  checkInDate: string;
  alreadyClaimed: boolean;
  rewardUSD: number;
  rewardNanousd: number;
  consecutiveDays: number;
  balanceTransactionID: number;
  account?: {
    id: number;
    userID: number;
    balanceUSD: number;
    balanceNanousd: number;
  };
  transaction?: {
    id: number;
    type: string;
    amountUSD: number;
    description: string;
    createdAt: string;
  };
}
