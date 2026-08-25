import type {
  BillingAccountDataResponse,
  BillingAccountResponse,
  BillingConfigDataResponse,
  BillingConfigResponse,
  BillingOverviewDataResponse,
  BillingOverviewResponse,
  BillingPlanResponse,
  BillingPriceResponse,
  CheckoutDataResponse,
  CheckoutResponse,
  CreateCheckoutRequest as ContractCreateCheckoutRequest,
  NativeToolPricingResponse,
  RedeemCodeRequest,
  RedemptionApplyDataResponse,
  RedemptionResponse,
  SubscriptionDataResponse,
  SubscriptionEntitlementResponse,
  SubscriptionResponse,
  UsageDailyModelResponse,
  UsageDailyResponse,
  UsageLedgerResponse,
  UsageMonthlyResponse,
} from "@deeix/api-contract";

export type BillingPlanPriceDTO = BillingPriceResponse;

export type BillingPlanDTO = Omit<BillingPlanResponse, "permissionGroupID" | "prices"> & {
  prices: BillingPlanPriceDTO[];
};

export type CreateCheckoutRequest = ContractCreateCheckoutRequest;

export type CheckoutDTO = CheckoutResponse;

export type CheckoutData = Omit<CheckoutDataResponse, "checkout"> & {
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

export type NativeToolPricingDTO = Pick<
  NativeToolPricingResponse,
  "billable" | "priceLabel" | "priceNanousd" | "provider" | "toolKey" | "unit"
>;

export type BillingConfigData = Omit<BillingConfigDataResponse, "config"> & {
  config: Pick<BillingConfigResponse, "nativeToolBillingEnabled" | "usdToCNYRate"> & {
    mode: BillingMode;
    nativeToolPricing: NativeToolPricingDTO[];
    paymentProviders: Array<"stripe" | "epay" | string>;
    displayCurrency: "USD" | "CNY" | string;
    epayTypes: Array<{ name: string; type: string }>;
  };
};

export type BillingAccountData = Omit<BillingAccountDataResponse, "account"> & {
  account: BillingAccountResponse;
};

export type BillingOverviewData = Omit<BillingOverviewDataResponse, "overview"> & {
  overview: Omit<
    BillingOverviewResponse,
    "account" | "periodEndAt" | "periodStartAt" | "plan" | "subscriptionEntitlements"
  > & {
    mode: BillingMode;
    plan: BillingPlanDTO | null;
    periodStartAt: string | null;
    periodEndAt: string | null;
    account: BillingAccountData["account"] | null;
    subscriptionEntitlements: BillingSubscriptionEntitlementDTO[];
  };
};

export type BillingSubscriptionDTO = SubscriptionResponse;

export type BillingSubscriptionEntitlementDTO = Omit<
  SubscriptionEntitlementResponse,
  "plan"
> & {
  plan: BillingPlanDTO;
};

export type RedeemBillingCodeRequest = RedeemCodeRequest;

export type BillingRedemptionDTO = RedemptionResponse;

export type RedeemBillingCodeData = Omit<
  RedemptionApplyDataResponse,
  "account" | "overview" | "redemption" | "subscription"
> & {
  redemption: BillingRedemptionDTO;
  account?: BillingAccountData["account"];
  subscription?: SubscribeData["subscription"];
  overview: BillingOverviewData["overview"];
};

export type BillingUsageLedgerDTO = Omit<UsageLedgerResponse, "billingAt">;

export type BillingUsageMonthlyDTO = UsageMonthlyResponse;

export type BillingUsageDailyDTO = Omit<UsageDailyResponse, "models"> & {
  models: BillingUsageDailyModelDTO[];
};

export type BillingUsageDailyModelDTO = UsageDailyModelResponse;

export type SubscribeData = Omit<SubscriptionDataResponse, "subscription"> & {
  subscription: Pick<BillingSubscriptionDTO, "id" | "planID" | "priceID" | "status">;
};
