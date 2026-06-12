import { authedFetch, authedRequest } from "@/shared/api/authed-client";
import { apiRequest } from "@/shared/api/http-client";
import type { PagePayload } from "@/shared/api/common.types";
import type {
  BillingAccountData,
  BillingBalanceTransactionDTO,
  BillingConfigData,
  BillingOverviewData,
  BillingPaymentOrderDTO,
  BillingPaymentOrderData,
  BillingRedemptionDTO,
  BillingUsageDailyDTO,
  BillingPlanDTO,
  BillingUsageLedgerDTO,
  BillingUsageMonthlyDTO,
  CheckoutData,
  CreateCheckoutRequest,
  RedeemBillingCodeData,
  RedeemBillingCodeRequest,
  SubscribeData,
} from "@/shared/api/billing.types";

export async function getBillingConfig(accessToken: string): Promise<BillingConfigData> {
  return authedRequest<BillingConfigData>("/api/v1/billing/config", { accessToken }, true);
}

export async function listBillingPlans(accessToken: string): Promise<BillingPlanDTO[]> {
  return authedRequest<BillingPlanDTO[]>("/api/v1/billing/plans", { accessToken }, true);
}

export async function listPublicBillingPlans(): Promise<BillingPlanDTO[]> {
  return apiRequest<BillingPlanDTO[]>("/api/v1/public/billing/plans");
}

export async function getBillingAccount(accessToken: string): Promise<BillingAccountData> {
  return authedRequest<BillingAccountData>("/api/v1/billing/account", { accessToken }, true);
}

export async function getBillingOverview(accessToken: string): Promise<BillingOverviewData> {
  return authedRequest<BillingOverviewData>("/api/v1/billing/overview", { accessToken }, true);
}

export async function listBillingUsage(
  accessToken: string,
  options: { page?: number; pageSize?: number; query?: string; status?: string; sort?: string; createdFrom?: string; createdTo?: string } = {},
): Promise<PagePayload<BillingUsageLedgerDTO>> {
  const page = options.page && options.page > 0 ? options.page : 1;
  const pageSize = options.pageSize && options.pageSize > 0 ? options.pageSize : 10;
  const params = new URLSearchParams({
    page: String(page),
    page_size: String(pageSize),
  });
  if (options.query?.trim()) params.set("query", options.query.trim());
  if (options.status?.trim()) params.set("status", options.status.trim());
  if (options.sort?.trim()) params.set("sort", options.sort.trim());
  if (options.createdFrom?.trim()) params.set("created_from", options.createdFrom.trim());
  if (options.createdTo?.trim()) params.set("created_to", options.createdTo.trim());
  return authedRequest<PagePayload<BillingUsageLedgerDTO>>(
    `/api/v1/billing/usage?${params.toString()}`,
    { accessToken },
    true,
  );
}

export async function listBillingMonthlyUsage(accessToken: string, months = 12): Promise<BillingUsageMonthlyDTO[]> {
  const params = new URLSearchParams({ months: String(months) });
  return authedRequest<BillingUsageMonthlyDTO[]>(
    `/api/v1/billing/usage/monthly?${params.toString()}`,
    { accessToken },
    true,
  );
}

export async function listBillingDailyUsage(
  accessToken: string,
  options: { days?: number; startDate?: string; endDate?: string } = {},
): Promise<BillingUsageDailyDTO[]> {
  const params = new URLSearchParams();
  if (options.startDate && options.endDate) {
    params.set("start_date", options.startDate);
    params.set("end_date", options.endDate);
  } else if (options.days && options.days > 0) {
    params.set("days", String(options.days && options.days > 0 ? options.days : 30));
  }
  const query = params.toString();
  return authedRequest<BillingUsageDailyDTO[]>(
    `/api/v1/billing/usage/daily${query ? `?${query}` : ""}`,
    { accessToken },
    true,
  );
}

export async function createBillingCheckout(accessToken: string, payload: CreateCheckoutRequest): Promise<CheckoutData> {
  return authedRequest<CheckoutData>(
    "/api/v1/billing/payments/checkout",
    { method: "POST", accessToken, body: payload },
    true,
  );
}

export async function listBillingPaymentOrders(
  accessToken: string,
  options: { page?: number; pageSize?: number; status?: string; orderType?: string; provider?: string; query?: string; sort?: string } = {},
): Promise<PagePayload<BillingPaymentOrderDTO>> {
  const page = options.page && options.page > 0 ? options.page : 1;
  const pageSize = options.pageSize && options.pageSize > 0 ? options.pageSize : 20;
  const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
  if (options.status?.trim()) params.set("status", options.status.trim());
  if (options.orderType?.trim()) params.set("order_type", options.orderType.trim());
  if (options.provider?.trim()) params.set("provider", options.provider.trim());
  if (options.query?.trim()) params.set("q", options.query.trim());
  if (options.sort?.trim()) params.set("sort", options.sort.trim());
  return authedRequest<PagePayload<BillingPaymentOrderDTO>>(
    `/api/v1/billing/payments?${params.toString()}`,
    { accessToken },
    true,
  );
}

export async function getBillingPaymentOrder(accessToken: string, orderNo: string): Promise<BillingPaymentOrderData> {
  return authedRequest<BillingPaymentOrderData>(
    `/api/v1/billing/payments/${encodeURIComponent(orderNo)}`,
    { accessToken },
    true,
  );
}

export async function redeemBillingCode(accessToken: string, payload: RedeemBillingCodeRequest): Promise<RedeemBillingCodeData> {
  return authedRequest<RedeemBillingCodeData>(
    "/api/v1/billing/redemptions",
    { method: "POST", accessToken, body: payload },
    true,
  );
}

export async function listBillingRedemptions(
  accessToken: string,
  options: { page?: number; pageSize?: number; mode?: string; query?: string; sort?: string } = {},
): Promise<PagePayload<BillingRedemptionDTO>> {
  const page = options.page && options.page > 0 ? options.page : 1;
  const pageSize = options.pageSize && options.pageSize > 0 ? options.pageSize : 20;
  const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
  if (options.mode?.trim()) params.set("mode", options.mode.trim());
  if (options.query?.trim()) params.set("q", options.query.trim());
  if (options.sort?.trim()) params.set("sort", options.sort.trim());
  return authedRequest<PagePayload<BillingRedemptionDTO>>(
    `/api/v1/billing/redemptions?${params.toString()}`,
    { accessToken },
    true,
  );
}

export async function listBillingBalanceTransactions(
  accessToken: string,
  options: { page?: number; pageSize?: number; type?: string; query?: string; sort?: string } = {},
): Promise<PagePayload<BillingBalanceTransactionDTO>> {
  const page = options.page && options.page > 0 ? options.page : 1;
  const pageSize = options.pageSize && options.pageSize > 0 ? options.pageSize : 20;
  const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
  if (options.type?.trim()) params.set("type", options.type.trim());
  if (options.query?.trim()) params.set("q", options.query.trim());
  if (options.sort?.trim()) params.set("sort", options.sort.trim());
  return authedRequest<PagePayload<BillingBalanceTransactionDTO>>(
    `/api/v1/billing/balance-transactions?${params.toString()}`,
    { accessToken },
    true,
  );
}

export function billingUsageCSVURL(options: { query?: string; status?: string; sort?: string; createdFrom?: string; createdTo?: string } = {}): string {
  const params = new URLSearchParams();
  if (options.query?.trim()) params.set("query", options.query.trim());
  if (options.status?.trim()) params.set("status", options.status.trim());
  if (options.sort?.trim()) params.set("sort", options.sort.trim());
  if (options.createdFrom?.trim()) params.set("created_from", options.createdFrom.trim());
  if (options.createdTo?.trim()) params.set("created_to", options.createdTo.trim());
  const query = params.toString();
  return `/api/v1/billing/usage.csv${query ? `?${query}` : ""}`;
}

export async function exportBillingUsageCSV(
  accessToken: string,
  options: { query?: string; status?: string; sort?: string; createdFrom?: string; createdTo?: string } = {},
): Promise<Blob> {
  const response = await authedFetch(billingUsageCSVURL(options), { accessToken }, true);
  return response.blob();
}

export async function subscribeBillingPlan(accessToken: string, priceID: number, cycles = 1): Promise<SubscribeData> {
  return authedRequest<SubscribeData>(
    "/api/v1/billing/subscriptions",
    { method: "POST", accessToken, body: { priceID: priceID, cycles } },
    true,
  );
}
