"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { completeEmailCodeLogin, completeEmailRegistration, completePasswordReset, getLoginOptions, getLoginPageSettings, login, startEmailCodeLogin, startEmailRegistration, startPasswordReset, startTwoFactorEmailVerification, verifyTwoFactorLogin } from "@/shared/api/auth";
import type { LoginOptionsData, LoginPageSettings, SecurityVerificationMethod } from "@/shared/api/auth.types";
import { resolveApiBaseURL } from "@/shared/api/http-client";
import { isPasswordPolicyValid } from "@/shared/auth/account-policy";
import { normalizeAuthNextPath } from "@/shared/auth/local-path";
import { resolveAccessToken } from "@/shared/auth/resolve-access-token";
import { writeSessionSnapshot } from "@/shared/auth/session";
import { useLocalizedErrorMessage } from "@/i18n/use-localized-error";
import { getSuspendedAccountReason, isSuspendedAccountError } from "@/features/auth/lib/suspended-account-message";
import {
  createProviderPKCE,
  DEFAULT_LOGIN_OPTIONS,
  DEFAULT_LOGIN_SETTINGS,
  isTwoFactorChallengeExpired,
  normalizeRegisterCode,
  normalizeTwoFactorInput,
  providerPKCEStorageKey,
  TWO_FACTOR_CHALLENGE_STORAGE_KEY,
  TWO_FACTOR_METHODS_STORAGE_KEY,
  type LoginMode,
  type ProviderAuthIntent,
} from "@/features/auth/model/login-page";

type UseLoginPageInput = {
  nextPath: string;
};

const VERIFICATION_CODE_RESEND_COOLDOWN_MS = 60_000;

function parseSecurityVerificationMethods(value: string | null): SecurityVerificationMethod[] {
  if (!value) {
    return ["two_factor"];
  }
  try {
    const parsed = JSON.parse(value) as unknown;
    if (!Array.isArray(parsed)) {
      return ["two_factor"];
    }
    const methods = parsed.filter((item): item is SecurityVerificationMethod => item === "two_factor" || item === "email");
    return methods.length > 0 ? methods : ["two_factor"];
  } catch {
    return ["two_factor"];
  }
}

export function useLoginPage({ nextPath }: UseLoginPageInput) {
  const router = useRouter();
  const t = useTranslations("login");
  const resolveErrorMessage = useLocalizedErrorMessage();
  const [settings, setSettings] = React.useState<LoginPageSettings>(DEFAULT_LOGIN_SETTINGS);
  const [options, setOptions] = React.useState<LoginOptionsData>(DEFAULT_LOGIN_OPTIONS);
  const [username, setUsername] = React.useState("");
  const [password, setPassword] = React.useState("");
  const [twoFactorChallengeToken, setTwoFactorChallengeToken] = React.useState("");
  const [twoFactorVerificationMethods, setTwoFactorVerificationMethods] = React.useState<SecurityVerificationMethod[]>(["two_factor"]);
  const [twoFactorVerificationMethod, setTwoFactorVerificationMethod] = React.useState<SecurityVerificationMethod>("two_factor");
  const [twoFactorCode, setTwoFactorCode] = React.useState("");
  const [mode, setMode] = React.useState<LoginMode>("login");
  const [registerEmail, setRegisterEmail] = React.useState("");
  const [registerPassword, setRegisterPassword] = React.useState("");
  const [registerCode, setRegisterCode] = React.useState("");
  const [registerInviteCode, setRegisterInviteCode] = React.useState("");
  const [registerTurnstileToken, setRegisterTurnstileToken] = React.useState("");
  const [registerTurnstileResetSignal, setRegisterTurnstileResetSignal] = React.useState(0);
  const [emailCodeLoginEmail, setEmailCodeLoginEmail] = React.useState("");
  const [emailCodeLoginCode, setEmailCodeLoginCode] = React.useState("");
  const [passwordResetEmail, setPasswordResetEmail] = React.useState("");
  const [passwordResetCode, setPasswordResetCode] = React.useState("");
  const [passwordResetNewPassword, setPasswordResetNewPassword] = React.useState("");
  const [codeSent, setCodeSent] = React.useState(false);
  const [configReady, setConfigReady] = React.useState(false);
  const [submitting, setSubmitting] = React.useState(false);
  const [sendingCode, setSendingCode] = React.useState(false);
  const [registerCodeResendAt, setRegisterCodeResendAt] = React.useState(0);
  const [twoFactorEmailCodeResendAt, setTwoFactorEmailCodeResendAt] = React.useState(0);
  const [emailCodeLoginResendAt, setEmailCodeLoginResendAt] = React.useState(0);
  const [passwordResetCodeResendAt, setPasswordResetCodeResendAt] = React.useState(0);
  const [cooldownNow, setCooldownNow] = React.useState(() => Date.now());
  const registerCodeCooldownSeconds = Math.max(0, Math.ceil((registerCodeResendAt - cooldownNow) / 1000));
  const twoFactorEmailCodeCooldownSeconds = Math.max(0, Math.ceil((twoFactorEmailCodeResendAt - cooldownNow) / 1000));
  const emailCodeLoginCooldownSeconds = Math.max(0, Math.ceil((emailCodeLoginResendAt - cooldownNow) / 1000));
  const passwordResetCodeCooldownSeconds = Math.max(0, Math.ceil((passwordResetCodeResendAt - cooldownNow) / 1000));

  const fallbackNextPath = normalizeAuthNextPath(settings.defaultNextPath);
  const resolvedNextPath = normalizeAuthNextPath(nextPath, fallbackNextPath);
  const passwordLoginEnabled = options.usernameEnabled || options.emailEnabled;
  const loginProviders = React.useMemo(
    () => options.providers.filter((provider) => provider.loginEnabled),
    [options.providers],
  );
  const emailRegistrationEnabled = options.emailEnabled && options.emailRegistrationEnabled;
  const emailVerificationEnabled = options.emailVerificationEnabled;
  const emailCodeLoginEnabled = options.emailCodeLoginEnabled;
  const passwordResetEnabled = passwordLoginEnabled && options.passwordResetEnabled;
  const inviteRegistrationRequired = options.inviteRegistrationRequired;
  const registerTurnstileSiteKey = options.turnstileSiteKey?.trim() ?? "";
  const registerTurnstileRequired = options.turnstileRegistrationEnabled && Boolean(registerTurnstileSiteKey);
  const canShowRegister = emailRegistrationEnabled;

  React.useEffect(() => {
    if (registerCodeCooldownSeconds === 0 && twoFactorEmailCodeCooldownSeconds === 0 && emailCodeLoginCooldownSeconds === 0 && passwordResetCodeCooldownSeconds === 0) {
      return undefined;
    }
    const timer = window.setInterval(() => setCooldownNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [emailCodeLoginCooldownSeconds, passwordResetCodeCooldownSeconds, registerCodeCooldownSeconds, twoFactorEmailCodeCooldownSeconds]);

  React.useEffect(() => {
    let mounted = true;
    void resolveAccessToken()
      .then((token) => {
        if (mounted && token) router.replace(resolvedNextPath);
      })
      .catch((): undefined => undefined);
    return () => {
      mounted = false;
    };
  }, [resolvedNextPath, router]);

  React.useEffect(() => {
    const challenge = window.sessionStorage.getItem(TWO_FACTOR_CHALLENGE_STORAGE_KEY);
    if (challenge) {
      window.sessionStorage.removeItem(TWO_FACTOR_CHALLENGE_STORAGE_KEY);
      const rawMethods = window.sessionStorage.getItem(TWO_FACTOR_METHODS_STORAGE_KEY);
      window.sessionStorage.removeItem(TWO_FACTOR_METHODS_STORAGE_KEY);
      const parsedMethods = parseSecurityVerificationMethods(rawMethods);
      setTwoFactorChallengeToken(challenge);
      setTwoFactorVerificationMethods(parsedMethods);
      setTwoFactorVerificationMethod(parsedMethods[0] ?? "two_factor");
      setMode("login");
    }
  }, []);

  React.useEffect(() => {
    let cancelled = false;
    void Promise.all([getLoginPageSettings(), getLoginOptions()])
      .then(([pageSettings, loginOptions]) => {
        if (cancelled) {
          return;
        }
        setSettings(pageSettings);
        setOptions(loginOptions);
      })
      .catch((): undefined => undefined)
      .finally(() => {
        if (!cancelled) {
          setConfigReady(true);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  React.useEffect(() => {
    document.title = settings.title?.trim() || t("title");
  }, [settings.title, t]);

  React.useEffect(() => {
    if (mode === "login" && !passwordLoginEnabled && loginProviders.length === 0 && canShowRegister) {
      setMode("register");
    } else if (mode === "register" && !canShowRegister) {
      setMode("login");
    } else if (mode === "emailCodeLogin" && !emailCodeLoginEnabled) {
      setMode("login");
    } else if (mode === "passwordReset" && !passwordResetEnabled) {
      setMode("login");
    }
  }, [canShowRegister, emailCodeLoginEnabled, loginProviders.length, mode, passwordLoginEnabled, passwordResetEnabled]);

  const completeAuth = React.useCallback((accessToken: string, sessionID: string) => {
    writeSessionSnapshot({ accessToken, sessionID });
    router.replace(resolvedNextPath);
  }, [resolvedNextPath, router]);

  const resetRegisterTurnstile = React.useCallback(() => {
    setRegisterTurnstileToken("");
    setRegisterTurnstileResetSignal((current) => current + 1);
  }, []);

  const onLoginSubmit = React.useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      if (submitting) {
        return;
      }

      setSubmitting(true);
      try {
        const formData = new FormData(event.currentTarget);
        const submittedUsername = String(formData.get("username") ?? username).trim();
        const submittedPassword = String(formData.get("password") ?? password);
        const submittedOTP = twoFactorVerificationMethod === "email"
          ? normalizeRegisterCode(String(formData.get("otp") ?? twoFactorCode))
          : normalizeTwoFactorInput(String(formData.get("otp") ?? twoFactorCode));
        if (twoFactorChallengeToken && submittedOTP.length < 6) {
          toast.error(t("toasts.codeRequired"));
          return;
        }
        const result = twoFactorChallengeToken
          ? await verifyTwoFactorLogin(twoFactorChallengeToken, submittedOTP, twoFactorVerificationMethod)
          : await login(submittedUsername, submittedPassword);
        if (result.twoFactorRequired) {
          const methods: SecurityVerificationMethod[] = result.verificationMethods?.length ? result.verificationMethods : ["two_factor"];
          setTwoFactorChallengeToken(result.twoFactorChallengeToken ?? "");
          setTwoFactorVerificationMethods(methods);
          setTwoFactorVerificationMethod(methods[0] ?? "two_factor");
          setTwoFactorCode("");
          return;
        }
        if (!result.accessToken) {
          toast.error(t("toasts.loginFailed"));
          return;
        }

        completeAuth(result.accessToken, result.sessionID);
      } catch (error) {
        if (isTwoFactorChallengeExpired(error)) {
          setTwoFactorChallengeToken("");
          setTwoFactorVerificationMethods(["two_factor"]);
          setTwoFactorVerificationMethod("two_factor");
          setTwoFactorCode("");
          setTwoFactorEmailCodeResendAt(0);
          toast.error(t("toasts.challengeExpired"));
          return;
        }
        const suspendedReason = getSuspendedAccountReason(error);
        if (suspendedReason) {
          toast.error(t("toasts.accountSuspendedWithReason", { reason: suspendedReason }));
          return;
        }
        if (isSuspendedAccountError(error)) {
          toast.error(t("toasts.accountSuspended"));
          return;
        }
        toast.error(resolveErrorMessage(error, t("toasts.loginRetry")));
      } finally {
        setSubmitting(false);
      }
    },
    [completeAuth, password, resolveErrorMessage, submitting, t, twoFactorChallengeToken, twoFactorCode, twoFactorVerificationMethod, username],
  );

  const handleProviderLogin = React.useCallback(async (slug: string, intent: ProviderAuthIntent = "login") => {
    try {
      const params = new URLSearchParams();
      const redirectURI = `${window.location.origin}/auth/callback?provider=${encodeURIComponent(slug)}`;
      const pkce = await createProviderPKCE();
      window.sessionStorage.setItem(providerPKCEStorageKey(slug), pkce.verifier);
      params.set("redirect_uri", redirectURI);
      params.set("next", resolvedNextPath);
      params.set("code_challenge", pkce.challenge);
      params.set("intent", intent);
      window.location.href = `${resolveApiBaseURL()}/api/v1/auth/providers/${encodeURIComponent(slug)}/start?${params.toString()}`;
    } catch {
      toast.error(t("toasts.providerStartFailed"));
    }
  }, [resolvedNextPath, t]);

  const requestRegisterCode = React.useCallback(async () => {
    if (!emailVerificationEnabled || sendingCode || registerCodeCooldownSeconds > 0) {
      return;
    }
    if (registerTurnstileRequired && !registerTurnstileToken) {
      toast.error(t("toasts.turnstileRequired"));
      return;
    }
    setSendingCode(true);
    try {
      const result = await startEmailRegistration(registerEmail, registerTurnstileRequired ? registerTurnstileToken : undefined);
      setCodeSent(result.sent);
      if (result.sent) {
        const now = Date.now();
        setCooldownNow(now);
        setRegisterCodeResendAt(now + VERIFICATION_CODE_RESEND_COOLDOWN_MS);
      }
    } catch (error) {
      toast.error(resolveErrorMessage(error, t("toasts.codeSendFailed")));
    } finally {
      if (registerTurnstileRequired && registerTurnstileToken) {
        resetRegisterTurnstile();
      }
      setSendingCode(false);
    }
  }, [emailVerificationEnabled, registerCodeCooldownSeconds, registerEmail, registerTurnstileRequired, registerTurnstileToken, resetRegisterTurnstile, resolveErrorMessage, sendingCode, t]);

  const requestTwoFactorEmailCode = React.useCallback(async () => {
    if (!twoFactorChallengeToken || twoFactorVerificationMethod !== "email" || sendingCode || twoFactorEmailCodeCooldownSeconds > 0) {
      return;
    }
    setSendingCode(true);
    try {
      const result = await startTwoFactorEmailVerification(twoFactorChallengeToken);
      if (result.sent) {
        const now = Date.now();
        setCooldownNow(now);
        setTwoFactorEmailCodeResendAt(now + VERIFICATION_CODE_RESEND_COOLDOWN_MS);
      }
    } catch (error) {
      toast.error(resolveErrorMessage(error, t("toasts.codeSendFailed")));
    } finally {
      setSendingCode(false);
    }
  }, [resolveErrorMessage, sendingCode, t, twoFactorChallengeToken, twoFactorEmailCodeCooldownSeconds, twoFactorVerificationMethod]);

  const resolveClientLocale = React.useCallback(() => navigator.language || "zh-CN", []);

  const resolveClientTimezone = React.useCallback(() => Intl.DateTimeFormat().resolvedOptions().timeZone || "Etc/UTC", []);

  const onRegisterSubmit = React.useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      if (submitting) {
        return;
      }
      if (!isPasswordPolicyValid(registerPassword)) {
        toast.error(t("toasts.passwordInvalid"));
        return;
      }
      if (registerTurnstileRequired && !emailVerificationEnabled && !registerTurnstileToken) {
        toast.error(t("toasts.turnstileRequired"));
        return;
      }
      setSubmitting(true);
      try {
        const result = await completeEmailRegistration(
          registerEmail,
          registerPassword,
          emailVerificationEnabled ? registerCode : "",
          {
            turnstileToken: registerTurnstileRequired && !emailVerificationEnabled ? registerTurnstileToken : undefined,
            inviteCode: inviteRegistrationRequired ? registerInviteCode : undefined,
            locale: resolveClientLocale(),
            timezone: resolveClientTimezone(),
          },
        );
        completeAuth(result.accessToken, result.sessionID);
      } catch (error) {
        toast.error(resolveErrorMessage(error, t("toasts.registerFailed")));
      } finally {
        if (registerTurnstileRequired && !emailVerificationEnabled && registerTurnstileToken) {
          resetRegisterTurnstile();
        }
        setSubmitting(false);
      }
    },
    [completeAuth, emailVerificationEnabled, inviteRegistrationRequired, registerCode, registerEmail, registerInviteCode, registerPassword, registerTurnstileRequired, registerTurnstileToken, resetRegisterTurnstile, resolveClientLocale, resolveClientTimezone, resolveErrorMessage, submitting, t],
  );

  const updateRegisterEmail = React.useCallback((value: string) => {
    setRegisterEmail(value);
    setCodeSent(false);
    setRegisterCodeResendAt(0);
  }, []);

  const requestEmailCodeLoginCode = React.useCallback(async () => {
    if (!emailCodeLoginEnabled || sendingCode || emailCodeLoginCooldownSeconds > 0) {
      return;
    }
    setSendingCode(true);
    try {
      const result = await startEmailCodeLogin(emailCodeLoginEmail);
      if (result.sent) {
        const now = Date.now();
        setCooldownNow(now);
        setEmailCodeLoginResendAt(now + VERIFICATION_CODE_RESEND_COOLDOWN_MS);
        toast.success(t("toasts.codeSent"));
      }
    } catch (error) {
      toast.error(resolveErrorMessage(error, t("toasts.codeSendFailed")));
    } finally {
      setSendingCode(false);
    }
  }, [emailCodeLoginCooldownSeconds, emailCodeLoginEmail, emailCodeLoginEnabled, resolveErrorMessage, sendingCode, t]);

  const onEmailCodeLoginSubmit = React.useCallback(async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting) {
      return;
    }
    setSubmitting(true);
    try {
      const result = await completeEmailCodeLogin(emailCodeLoginEmail, emailCodeLoginCode);
      if (result.twoFactorRequired) {
        const methods: SecurityVerificationMethod[] = result.verificationMethods?.length ? result.verificationMethods : ["two_factor"];
        setTwoFactorChallengeToken(result.twoFactorChallengeToken ?? "");
        setTwoFactorVerificationMethods(methods);
        setTwoFactorVerificationMethod(methods[0] ?? "two_factor");
        setTwoFactorCode("");
        setMode("login");
        return;
      }
      completeAuth(result.accessToken, result.sessionID);
    } catch (error) {
      toast.error(resolveErrorMessage(error, t("toasts.loginRetry")));
    } finally {
      setSubmitting(false);
    }
  }, [completeAuth, emailCodeLoginCode, emailCodeLoginEmail, resolveErrorMessage, submitting, t]);

  const requestPasswordResetCode = React.useCallback(async () => {
    if (!passwordResetEnabled || sendingCode || passwordResetCodeCooldownSeconds > 0) {
      return;
    }
    setSendingCode(true);
    try {
      const result = await startPasswordReset(passwordResetEmail);
      if (result.sent) {
        const now = Date.now();
        setCooldownNow(now);
        setPasswordResetCodeResendAt(now + VERIFICATION_CODE_RESEND_COOLDOWN_MS);
        toast.success(t("toasts.codeSent"));
      }
    } catch (error) {
      toast.error(resolveErrorMessage(error, t("toasts.codeSendFailed")));
    } finally {
      setSendingCode(false);
    }
  }, [passwordResetCodeCooldownSeconds, passwordResetEmail, passwordResetEnabled, resolveErrorMessage, sendingCode, t]);

  const onPasswordResetSubmit = React.useCallback(async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting) {
      return;
    }
    if (!isPasswordPolicyValid(passwordResetNewPassword)) {
      toast.error(t("toasts.passwordInvalid"));
      return;
    }
    setSubmitting(true);
    try {
      await completePasswordReset(passwordResetEmail, passwordResetCode, passwordResetNewPassword);
      toast.success(t("toasts.passwordResetDone"));
      setMode("login");
      setPasswordResetCode("");
      setPasswordResetNewPassword("");
    } catch (error) {
      toast.error(resolveErrorMessage(error, t("toasts.passwordResetFailed")));
    } finally {
      setSubmitting(false);
    }
  }, [passwordResetCode, passwordResetEmail, passwordResetNewPassword, resolveErrorMessage, submitting, t]);

  const cancelTwoFactorChallenge = React.useCallback(() => {
    setTwoFactorChallengeToken("");
    setTwoFactorVerificationMethods(["two_factor"]);
    setTwoFactorVerificationMethod("two_factor");
    setTwoFactorCode("");
    setTwoFactorEmailCodeResendAt(0);
  }, []);

  const switchTwoFactorVerificationMethod = React.useCallback((method: SecurityVerificationMethod) => {
    setTwoFactorVerificationMethod(method);
    setTwoFactorCode("");
  }, []);

  const toggleLoginMode = React.useCallback(() => {
    if (mode !== "login") {
      setMode("login");
      return;
    }
    if (canShowRegister) {
      setMode("register");
    }
  }, [canShowRegister, mode]);

  return {
    codeSent,
    configReady,
    emailCodeLoginCode,
    emailCodeLoginCooldownSeconds,
    emailCodeLoginEmail,
    emailCodeLoginEnabled,
    emailRegistrationEnabled,
    emailVerificationEnabled,
    handleProviderLogin,
    inviteRegistrationRequired,
    loginProviders,
    mode,
    onEmailCodeLoginSubmit,
    onLoginSubmit,
    onPasswordResetSubmit,
    onRegisterSubmit,
    options,
    password,
    passwordLoginEnabled,
    passwordResetCode,
    passwordResetCodeCooldownSeconds,
    passwordResetEmail,
    passwordResetEnabled,
    passwordResetNewPassword,
    registerCode,
    registerCodeCooldownSeconds,
    registerEmail,
    registerInviteCode,
    registerPassword,
    registerTurnstileRequired,
    registerTurnstileResetSignal,
    registerTurnstileSiteKey,
    registerTurnstileToken,
    requestEmailCodeLoginCode,
    requestPasswordResetCode,
    requestRegisterCode,
    requestTwoFactorEmailCode,
    sendingCode,
    setEmailCodeLoginCode: (value: string) => setEmailCodeLoginCode(normalizeRegisterCode(value)),
    setEmailCodeLoginEmail,
    setMode,
    setPassword,
    setPasswordResetCode: (value: string) => setPasswordResetCode(normalizeRegisterCode(value)),
    setPasswordResetEmail,
    setPasswordResetNewPassword,
    setRegisterCode: (value: string) => setRegisterCode(normalizeRegisterCode(value)),
    setRegisterInviteCode,
    setRegisterPassword,
    setRegisterTurnstileToken,
    setTwoFactorCode: (value: string) => setTwoFactorCode(twoFactorVerificationMethod === "email" ? normalizeRegisterCode(value) : normalizeTwoFactorInput(value)),
    switchTwoFactorVerificationMethod,
    setUsername,
    submitting,
    toggleLoginMode,
    twoFactorChallengeToken,
    twoFactorCode,
    twoFactorEmailCodeCooldownSeconds,
    twoFactorVerificationMethod,
    twoFactorVerificationMethods,
    updateRegisterEmail,
    username,
    cancelTwoFactorChallenge,
    canShowRegisterSwitch: canShowRegister,
  };
}
