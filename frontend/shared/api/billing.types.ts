export type BillingPlanPriceDTO = {
  id: number;
  planID: number;
  code: string;
  billingInterval: "month" | "year" | "lifetime" | string;
  currency: string;
  amountCents: number;
  isDefault: boolean;
};

export type BillingPlanDTO = {
  id: number;
  code: "free" | "pro" | "max" | "ultra" | string;
  name: string;
  description: string;
  featureJSON: string;
  periodCreditUSD: number;
  periodCreditNanousd: number;
  discountPercent: number;
  sortOrder: number;
  isActive: boolean;
  prices: BillingPlanPriceDTO[];
};

export type CreateCheckoutRequest = {
  orderType?: "subscription" | "topup";
  priceID?: number;
  amountUSD?: number;
  cycles?: number;
  paymentProvider?: "stripe" | "epay" | string;
  epayType?: string;
  successURL?: string;
  cancelURL?: string;
};

export type CheckoutDTO = {
  orderNo: string;
  orderType: "subscription" | "topup" | string;
  provider: "stripe" | "epay" | string;
  status: string;
  checkoutURL: string;
  externalCheckoutID: string;
  baseAmountCents: number;
  baseCurrency: string;
  payAmountCents: number;
  payCurrency: string;
  fxRate: string;
  creditNanousd: number;
  creditUSD: number;
  expiredAt: string | null;
};

export type CheckoutData = {
  checkout: CheckoutDTO;
};

export type BillingPaymentOrderDTO = {
  orderNo: string;
  orderType: "subscription" | "topup" | string;
  userID: number;
  planID: number;
  priceID: number;
  provider: "stripe" | "epay" | string;
  status: "pending" | "paid" | "failed" | "expired" | string;
  baseAmountCents: number;
  baseCurrency: string;
  payAmountCents: number;
  payCurrency: string;
  fxRate: string;
  creditNanousd: number;
  creditUSD: number;
  billingInterval: string;
  cycles: number;
  paidAt: string | null;
  expiredAt: string | null;
  createdAt: string;
  updatedAt: string;
};

export type BillingPaymentOrderData = {
  order: BillingPaymentOrderDTO;
  activated?: boolean;
};

export type BillingBalanceTransactionDTO = {
  id: number;
  accountID: number;
  userID: number;
  type: string;
  amountNanousd: number;
  amountUSD: number;
  balanceAfterNanousd: number;
  balanceAfterUSD: number;
  refType: string;
  refID: number;
  refNo: string;
  description: string;
  createdAt: string;
  updatedAt: string;
};

export type BillingCheckInStatusDTO = {
  todayClaimed: boolean;
  rewardUSD: number;
  rewardNanousd: number;
  consecutiveDays: number;
  lastCheckInDate: string | null;
  nextCheckInDate: string;
  account: BillingAccountData["account"] | null;
  latestTransaction?: BillingBalanceTransactionDTO | null;
};

export type BillingCheckInStatusData = {
  checkIn: BillingCheckInStatusDTO;
};

export type BillingCheckInClaimDTO = {
  id: number;
  checkInDate: string;
  alreadyClaimed: boolean;
  rewardUSD: number;
  rewardNanousd: number;
  consecutiveDays: number;
  balanceTransactionID: number;
  account: BillingAccountData["account"] | null;
  transaction?: BillingBalanceTransactionDTO | null;
};

export type BillingCheckInClaimData = {
  checkIn: BillingCheckInClaimDTO;
};

export type BillingMode = "self" | "period" | "usage";

export type NativeToolPricingDTO = {
  provider: string;
  toolKey: string;
  priceNanousd: number;
  unit: "call" | "search" | string;
  priceLabel: "included" | "notMetered" | string;
  billable: boolean;
};

export type BillingConfigData = {
  config: {
    mode: BillingMode;
    nativeToolBillingEnabled: boolean;
    nativeToolPricing: NativeToolPricingDTO[];
    paymentProviders: Array<"stripe" | "epay" | string>;
    usdToCNYRate: number;
    epayTypes: Array<{ name: string; type: string }>;
  };
};

export type BillingAccountData = {
  account: {
    userID: number;
    currency: string;
    balanceNanousd: number;
    balanceUSD: number;
    status: string;
    updatedAt: string;
  };
};

export type BillingOverviewData = {
  overview: {
    mode: BillingMode;
    plan: BillingPlanDTO | null;
    periodStartAt: string | null;
    periodEndAt: string | null;
    periodCreditUSD: number;
    periodCreditNanousd: number;
    periodUsedUSD: number;
    periodUsedNanousd: number;
    periodRemainingUSD: number;
    periodRemainingNanousd: number;
    account: BillingAccountData["account"] | null;
    subscriptionEntitlements: BillingSubscriptionEntitlementDTO[];
  };
};

export type BillingSubscriptionDTO = {
  id: number;
  userID: number;
  planID: number;
  priceID: number;
  status: string;
  startAt: string;
  currentPeriodStartAt: string;
  currentPeriodEndAt: string | null;
  cancelAtPeriodEnd: boolean;
  autoRenew: boolean;
};

export type BillingSubscriptionEntitlementDTO = BillingSubscriptionDTO & {
  plan: BillingPlanDTO;
  isCurrent: boolean;
};

export type RedeemBillingCodeRequest = {
  code: string;
};

export type BillingRedemptionDTO = {
  id: number;
  codeID: number;
  userID: number;
  mode: "usage" | "period" | string;
  rewardType: "balance" | "subscription" | string;
  creditUSD: number;
  creditNanousd: number;
  planID: number;
  subscriptionID: number;
  balanceTransactionID: number;
  createdAt: string;
};

export type RedeemBillingCodeData = {
  redemption: BillingRedemptionDTO;
  account?: BillingAccountData["account"];
  subscription?: SubscribeData["subscription"];
  overview: BillingOverviewData["overview"];
};

export type BillingUsageLedgerDTO = {
  id: number;
  userID: number;
  conversationID: number;
  providerProtocol: string;
  platformModelName: string;
  routedBindingCode: string;
  upstreamModelName: string;
  modelVendor: string;
  modelIcon: string;
  isFreeModel: boolean;
  usageDate: string;
  inputTokens: number;
  cacheReadTokens: number;
  cacheWriteTokens: number;
  cacheWrite5mTokens: number;
  cacheWrite1hTokens: number;
  outputTokens: number;
  reasoningTokens: number;
  callCount: number;
  durationSeconds: number;
  latencyMS: number;
  usageSpeed: string;
  serviceTier: string;
  billedCurrency: string;
  billedNanousd: number;
  billedUSD: number;
  pricingSnapshotJSON: string;
  createdAt: string;
  updatedAt: string;
};

export type BillingUsageMonthlyDTO = {
  monthStartAt: string;
  recordCount: number;
  inputTokens: number;
  cacheReadTokens: number;
  cacheWriteTokens: number;
  outputTokens: number;
  reasoningTokens: number;
  totalTokens: number;
  callCount: number;
  durationSeconds: number;
  avgLatencyMS: number;
  billedNanousd: number;
  billedUSD: number;
};

export type BillingUsageDailyDTO = {
  usageDate: string;
  recordCount: number;
  inputTokens: number;
  cacheReadTokens: number;
  cacheWriteTokens: number;
  outputTokens: number;
  reasoningTokens: number;
  totalTokens: number;
  callCount: number;
  durationSeconds: number;
  avgLatencyMS: number;
  billedNanousd: number;
  billedUSD: number;
  models: BillingUsageDailyModelDTO[];
};

export type BillingUsageDailyModelDTO = {
  platformModelName: string;
  recordCount: number;
  inputTokens: number;
  cacheReadTokens: number;
  cacheWriteTokens: number;
  outputTokens: number;
  reasoningTokens: number;
  totalTokens: number;
  callCount: number;
  durationSeconds: number;
  avgLatencyMS: number;
  billedNanousd: number;
  billedUSD: number;
};

export type SubscribeData = {
  subscription: {
    id: number;
    planID: number;
    priceID: number;
    status: string;
  };
};
