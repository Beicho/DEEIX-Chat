import { authedFetch, authedRequest } from "@/shared/api/authed-client";
import type { PagePayload } from "@/shared/api/common.types";
import type {
  AdminBillingConfigData,
  AdminBillingAccountData,
  AdminBalanceDeltaData,
  AdminBalanceTransactionDTO,
  AdminBalanceTransactionPage,
  AdminBillingRiskSummaryData,
  AdminBillingPlanDTO,
  AdminBillingPlanData,
  AdminPaymentOrderDTO,
  AdminPaymentOrderActionRequest,
  AdminPaymentOrderData,
  AdminPaymentOrderPage,
  AdminRedemptionCodeDTO,
  AdminRedemptionCodeBatchDeleteData,
  AdminRedemptionCodeBatchDeleteRequest,
  AdminRedemptionCodeCreateData,
  AdminRedemptionCodeData,
  AdminRedemptionCodeDeleteData,
  AdminRedemptionCodePage,
  AdminModelPricingDTO,
  AdminModelPricingData,
  AdminModelPricingPage,
  AdminOfficialPricingCatalogData,
  CreateAdminRedemptionCodeRequest,
  UpdateAdminRedemptionCodeRequest,
  UpdateAdminBillingConfigRequest,
  UpdateAdminBillingPlanRequest,
  UpdateAdminBillingAccountBalanceRequest,
  UpsertAdminModelPricingRequest,
} from "@/features/admin/api/billing.types";

import { normalizeAdminPagePayload, resolveAdminPage, type AdminPageOptions } from "./shared";

type ListAdminModelPricingOptions = AdminPageOptions & {
  query?: string;
};

type ListAdminRedemptionCodeOptions = AdminPageOptions & {
  query?: string;
  mode?: string;
  status?: string;
  availability?: string;
};

type ListAdminPaymentOrderOptions = AdminPageOptions & {
  userID?: number;
  status?: string;
  orderType?: string;
  provider?: string;
  query?: string;
  sort?: string;
};

type ListAdminBalanceTransactionOptions = AdminPageOptions & {
  userID?: number;
  type?: string;
  query?: string;
  sort?: string;
  createdFrom?: string;
  createdTo?: string;
};

export async function listAdminBillingPlans(accessToken: string): Promise<AdminBillingPlanDTO[]> {
  return authedRequest<AdminBillingPlanDTO[]>("/api/v1/admin/billing/plans", { accessToken }, true);
}

export async function updateAdminBillingPlan(
  accessToken: string,
  planID: number,
  payload: UpdateAdminBillingPlanRequest,
): Promise<AdminBillingPlanData> {
  return authedRequest<AdminBillingPlanData>(
    `/api/v1/admin/billing/plans/${planID}`,
    { method: "PATCH", accessToken, body: payload },
    true,
  );
}

export async function createAdminBillingPlan(
  accessToken: string,
  payload: CreateAdminBillingPlanRequest,
): Promise<AdminBillingPlanData> {
  return authedRequest<AdminBillingPlanData>(
    "/api/v1/admin/billing/plans",
    { method: "POST", accessToken, body: payload },
    true,
  );
}

export async function deleteAdminBillingPlan(accessToken: string, planID: number): Promise<{ deleted: boolean }> {
  return authedRequest<{ deleted: boolean }>(
    `/api/v1/admin/billing/plans/${planID}`,
    { method: "DELETE", accessToken },
    true,
  );
}

export async function getAdminBillingConfig(accessToken: string): Promise<AdminBillingConfigData> {
  return authedRequest<AdminBillingConfigData>("/api/v1/admin/billing/config", { accessToken }, true);
}

export async function patchAdminBillingConfig(accessToken: string, payload: UpdateAdminBillingConfigRequest): Promise<AdminBillingConfigData> {
  return authedRequest<AdminBillingConfigData>(
    "/api/v1/admin/billing/config",
    { method: "PATCH", accessToken, body: payload },
    true,
  );
}

export async function updateAdminBillingAccountBalance(
  accessToken: string,
  userID: number,
  payload: UpdateAdminBillingAccountBalanceRequest,
): Promise<AdminBillingAccountData> {
  return authedRequest<AdminBillingAccountData>(
    `/api/v1/admin/billing/accounts/${userID}/balance`,
    { method: "PATCH", accessToken, body: payload },
    true,
  );
}

export async function adjustAdminBillingAccountBalance(
  accessToken: string,
  userID: number,
  payload: AdjustAdminBillingAccountBalanceRequest,
): Promise<AdminBalanceDeltaData> {
  return authedRequest<AdminBalanceDeltaData>(
    `/api/v1/admin/billing/accounts/${userID}/balance-delta`,
    { method: "POST", accessToken, body: payload },
    true,
  );
}

export async function listAdminBalanceTransactions(
  accessToken: string,
  options: ListAdminBalanceTransactionOptions = {},
): Promise<AdminBalanceTransactionPage> {
  const { page, pageSize } = resolveAdminPage(options);
  const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
  if (options.userID && options.userID > 0) params.set("user_id", String(options.userID));
  if (options.type?.trim()) params.set("type", options.type.trim());
  if (options.query?.trim()) params.set("q", options.query.trim());
  if (options.sort?.trim()) params.set("sort", options.sort.trim());
  if (options.createdFrom?.trim()) params.set("created_from", options.createdFrom.trim());
  if (options.createdTo?.trim()) params.set("created_to", options.createdTo.trim());
  const data = await authedRequest<PagePayload<AdminBalanceTransactionDTO>>(
    `/api/v1/admin/billing/balance-transactions?${params.toString()}`,
    { accessToken },
    true,
  );
  return normalizeAdminPagePayload(data);
}

export async function listAdminPaymentOrders(
  accessToken: string,
  options: ListAdminPaymentOrderOptions = {},
): Promise<AdminPaymentOrderPage> {
  const { page, pageSize } = resolveAdminPage(options);
  const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
  if (options.userID && options.userID > 0) params.set("user_id", String(options.userID));
  if (options.status?.trim()) params.set("status", options.status.trim());
  if (options.orderType?.trim()) params.set("order_type", options.orderType.trim());
  if (options.provider?.trim()) params.set("provider", options.provider.trim());
  if (options.query?.trim()) params.set("q", options.query.trim());
  if (options.sort?.trim()) params.set("sort", options.sort.trim());
  const data = await authedRequest<PagePayload<AdminPaymentOrderDTO>>(
    `/api/v1/admin/billing/payment-orders?${params.toString()}`,
    { accessToken },
    true,
  );
  return normalizeAdminPagePayload(data);
}

export async function applyAdminPaymentOrderAction(
  accessToken: string,
  orderNo: string,
  payload: AdminPaymentOrderActionRequest,
): Promise<AdminPaymentOrderData> {
  return authedRequest<AdminPaymentOrderData>(
    `/api/v1/admin/billing/payment-orders/${encodeURIComponent(orderNo)}/actions`,
    { method: "POST", accessToken, body: payload },
    true,
  );
}

export async function getAdminBillingRiskSummary(accessToken: string): Promise<AdminBillingRiskSummaryData> {
  return authedRequest<AdminBillingRiskSummaryData>("/api/v1/admin/billing/risk-summary", { accessToken }, true);
}

export function adminBillingUsageCSVURL(options: { query?: string; platformModelName?: string; billingMode?: string; userID?: number; createdFrom?: string; createdTo?: string; sort?: string } = {}): string {
  const params = new URLSearchParams();
  if (options.query?.trim()) params.set("query", options.query.trim());
  if (options.platformModelName?.trim()) params.set("platform_model_name", options.platformModelName.trim());
  if (options.billingMode?.trim()) params.set("billing_mode", options.billingMode.trim());
  if (options.userID && options.userID > 0) params.set("user_id", String(options.userID));
  if (options.createdFrom?.trim()) params.set("created_from", options.createdFrom.trim());
  if (options.createdTo?.trim()) params.set("created_to", options.createdTo.trim());
  if (options.sort?.trim()) params.set("sort", options.sort.trim());
  const query = params.toString();
  return `/api/v1/admin/billing/usage.csv${query ? `?${query}` : ""}`;
}

export async function exportAdminBillingUsageCSV(
  accessToken: string,
  options: { query?: string; platformModelName?: string; billingMode?: string; userID?: number; createdFrom?: string; createdTo?: string; sort?: string } = {},
): Promise<Blob> {
  const response = await authedFetch(adminBillingUsageCSVURL(options), { accessToken }, true);
  return response.blob();
}

export async function listAdminRedemptionCodes(
  accessToken: string,
  options: ListAdminRedemptionCodeOptions = {},
): Promise<AdminRedemptionCodePage> {
  const { page, pageSize } = resolveAdminPage(options);
  const params = new URLSearchParams({
    page: String(page),
    page_size: String(pageSize),
  });
  if (options.query?.trim()) params.set("q", options.query.trim());
  if (options.mode?.trim()) params.set("mode", options.mode.trim());
  if (options.status?.trim()) params.set("status", options.status.trim());
  if (options.availability?.trim()) params.set("availability", options.availability.trim());
  const data = await authedRequest<PagePayload<AdminRedemptionCodeDTO>>(
    `/api/v1/admin/billing/redemption-codes?${params.toString()}`,
    { accessToken },
    true,
  );
  return normalizeAdminPagePayload(data);
}

export async function createAdminRedemptionCodes(
  accessToken: string,
  payload: CreateAdminRedemptionCodeRequest,
): Promise<AdminRedemptionCodeCreateData> {
  return authedRequest<AdminRedemptionCodeCreateData>(
    "/api/v1/admin/billing/redemption-codes",
    { method: "POST", accessToken, body: payload },
    true,
  );
}

export async function updateAdminRedemptionCode(
  accessToken: string,
  codeID: number,
  payload: UpdateAdminRedemptionCodeRequest,
): Promise<AdminRedemptionCodeData> {
  return authedRequest<AdminRedemptionCodeData>(
    `/api/v1/admin/billing/redemption-codes/${codeID}`,
    { method: "PATCH", accessToken, body: payload },
    true,
  );
}

export async function revealAdminRedemptionCode(
  accessToken: string,
  codeID: number,
): Promise<AdminRedemptionCodeData> {
  return authedRequest<AdminRedemptionCodeData>(
    `/api/v1/admin/billing/redemption-codes/${codeID}/code`,
    { accessToken },
    true,
  );
}

export async function deleteAdminRedemptionCode(
  accessToken: string,
  codeID: number,
): Promise<AdminRedemptionCodeDeleteData> {
  return authedRequest<AdminRedemptionCodeDeleteData>(
    `/api/v1/admin/billing/redemption-codes/${codeID}`,
    { method: "DELETE", accessToken },
    true,
  );
}

export async function batchDeleteAdminRedemptionCodes(
  accessToken: string,
  payload: AdminRedemptionCodeBatchDeleteRequest,
): Promise<AdminRedemptionCodeBatchDeleteData> {
  return authedRequest<AdminRedemptionCodeBatchDeleteData>(
    "/api/v1/admin/billing/redemption-codes/batch-delete",
    { method: "POST", accessToken, body: payload },
    true,
  );
}

export async function listAdminModelPricing(
  accessToken: string,
  options: ListAdminModelPricingOptions = {},
): Promise<AdminModelPricingPage> {
  const { page, pageSize } = resolveAdminPage(options);
  const params = new URLSearchParams({
    page: String(page),
    page_size: String(pageSize),
  });
  if (options.query?.trim()) {
    params.set("q", options.query.trim());
  }
  const data = await authedRequest<PagePayload<AdminModelPricingDTO>>(
    `/api/v1/admin/billing/model-prices?${params.toString()}`,
    { accessToken },
    true,
  );
  return normalizeAdminPagePayload(data);
}

export async function upsertAdminModelPricing(
  accessToken: string,
  payload: UpsertAdminModelPricingRequest,
): Promise<AdminModelPricingData> {
  return authedRequest<AdminModelPricingData>(
    "/api/v1/admin/billing/model-prices",
    { method: "PUT", accessToken, body: payload },
    true,
  );
}

export async function getAdminOpenRouterOfficialPricing(
  accessToken: string,
  options: { refresh?: boolean } = {},
): Promise<AdminOfficialPricingCatalogData> {
  const params = new URLSearchParams();
  if (options.refresh) {
    params.set("refresh", "true");
  }
  const suffix = params.size > 0 ? `?${params.toString()}` : "";
  return authedRequest<AdminOfficialPricingCatalogData>(
    `/api/v1/admin/billing/official-pricing/openrouter${suffix}`,
    { accessToken },
    true,
  );
}
