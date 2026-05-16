"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { resetPassword, confirmResetPassword } from "aws-amplify/auth";
import { ArrowLeftIcon, EyeIcon, EyeOffIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

const requestSchema = z.object({
  email: z
    .email("メールアドレスの形式が正しくありません")
    .min(1, "メールアドレスを入力してください")
    .max(254, "メールアドレスが長すぎます"),
});

const confirmSchema = z
  .object({
    newPassword: z
      .string()
      .min(8, "パスワードは8文字以上で、英大小・数字・記号を含めてください")
      .regex(
        /[a-z]/,
        "パスワードは8文字以上で、英大小・数字・記号を含めてください",
      )
      .regex(
        /[A-Z]/,
        "パスワードは8文字以上で、英大小・数字・記号を含めてください",
      )
      .regex(
        /[0-9]/,
        "パスワードは8文字以上で、英大小・数字・記号を含めてください",
      )
      .regex(
        /[^a-zA-Z0-9]/,
        "パスワードは8文字以上で、英大小・数字・記号を含めてください",
      ),
    confirmPassword: z.string().min(1, "確認用パスワードを入力してください"),
  })
  .refine((v) => v.newPassword === v.confirmPassword, {
    path: ["confirmPassword"],
    message: "パスワードが一致しません",
  });

type RequestValues = z.infer<typeof requestSchema>;
type ConfirmValues = z.infer<typeof confirmSchema>;
type Step = "request" | "confirm" | "done";

function popEmailFromStorage(): string {
  if (typeof window === "undefined") return "";
  const val = sessionStorage.getItem("passwordResetEmail") ?? "";
  if (val) sessionStorage.removeItem("passwordResetEmail");
  return val;
}

function msFromNow(durationMs: number): number {
  return Date.now() + durationMs;
}

function StepIndicator({ step }: { step: Step }) {
  const atConfirm = step === "confirm" || step === "done";
  return (
    <div className="mb-4 flex items-center gap-1.5 text-xs text-muted-foreground">
      <span className={step === "request" ? "font-medium text-primary" : ""}>
        ①メール送信
      </span>
      <span>→</span>
      <span className={atConfirm ? "font-medium text-primary" : ""}>
        ②新パスワード設定
      </span>
    </div>
  );
}

function PasswordChecklist({ password }: { password: string }) {
  const checks = [
    { label: "8文字以上", ok: password.length >= 8 },
    { label: "英大文字を含む", ok: /[A-Z]/.test(password) },
    { label: "英小文字を含む", ok: /[a-z]/.test(password) },
    { label: "数字を含む", ok: /[0-9]/.test(password) },
    { label: "記号を含む", ok: /[^a-zA-Z0-9]/.test(password) },
  ];
  return (
    <ul className="space-y-0.5 text-xs">
      {checks.map(({ label, ok }) => (
        <li
          key={label}
          className={ok ? "text-emerald-600" : "text-muted-foreground"}
        >
          {ok ? "✓" : "□"} {label}
        </li>
      ))}
    </ul>
  );
}

export default function PasswordResetPage() {
  const router = useRouter();

  const [step, setStep] = useState<Step>("request");
  const [email, setEmail] = useState(popEmailFromStorage);

  const [submitError, setSubmitError] = useState<string | null>(null);

  const [resendSecondsLeft, setResendSecondsLeft] = useState(0);
  const [resendLoading, setResendLoading] = useState(false);
  const [resendToast, setResendToast] = useState<{
    type: "success" | "error";
    message: string;
  } | null>(null);

  const [remainingAttempts, setRemainingAttempts] = useState(5);
  const [isLocked, setIsLocked] = useState(false);
  const [lockUntil, setLockUntil] = useState<number | null>(null);

  const [softExpireLeft, setSoftExpireLeft] = useState<number | null>(null);
  const [codeExpired, setCodeExpired] = useState(false);

  const [codeValues, setCodeValues] = useState(["", "", "", "", "", ""]);
  const [codeResetCount, setCodeResetCount] = useState(0);
  const codeRefs = useRef<(HTMLInputElement | null)[]>([]);

  const [passwordVisible, setPasswordVisible] = useState(false);
  const [confirmPasswordVisible, setConfirmPasswordVisible] = useState(false);

  const [showAbortDialog, setShowAbortDialog] = useState(false);
  const [showChangeEmailDialog, setShowChangeEmailDialog] = useState(false);

  const requestForm = useForm<RequestValues>({
    resolver: zodResolver(requestSchema),
    mode: "onBlur",
    defaultValues: { email },
  });

  const requestEmail = useWatch({
    control: requestForm.control,
    name: "email",
  });

  const confirmForm = useForm<ConfirmValues>({
    resolver: zodResolver(confirmSchema),
    mode: "onBlur",
    defaultValues: { newPassword: "", confirmPassword: "" },
  });

  const newPassword = useWatch({
    control: confirmForm.control,
    name: "newPassword",
  });
  const confirmPassword = useWatch({
    control: confirmForm.control,
    name: "confirmPassword",
  });

  useEffect(() => {
    if (resendSecondsLeft <= 0) return;
    const id = setInterval(() => {
      setResendSecondsLeft((s) => Math.max(0, s - 1));
    }, 1000);
    return () => clearInterval(id);
  }, [resendSecondsLeft]);

  useEffect(() => {
    if (softExpireLeft === null || softExpireLeft <= 0) return;
    const id = setInterval(() => {
      setSoftExpireLeft((s) => {
        if (s === null || s <= 1) {
          setCodeExpired(true);
          return 0;
        }
        return s - 1;
      });
    }, 1000);
    return () => clearInterval(id);
  }, [softExpireLeft]);

  useEffect(() => {
    if (lockUntil === null) return;
    const id = setInterval(() => {
      if (Date.now() >= lockUntil) {
        setIsLocked(false);
        setLockUntil(null);
      }
    }, 1000);
    return () => clearInterval(id);
  }, [lockUntil]);

  useEffect(() => {
    if (!resendToast) return;
    const id = setTimeout(() => setResendToast(null), 3000);
    return () => clearTimeout(id);
  }, [resendToast]);

  useEffect(() => {
    if (codeResetCount === 0) return;
    codeRefs.current[0]?.focus();
  }, [codeResetCount]);

  function handleBack() {
    if (step === "confirm") {
      setShowAbortDialog(true);
    } else {
      router.push("/login");
    }
  }

  function enterConfirmStep(submittedEmail: string) {
    setEmail(submittedEmail);
    setStep("confirm");
    setResendSecondsLeft(60);
    setSoftExpireLeft(15 * 60);
    setCodeExpired(false);
    setRemainingAttempts(5);
    setCodeValues(["", "", "", "", "", ""]);
    setSubmitError(null);
  }

  async function onRequestSubmit(values: RequestValues) {
    setSubmitError(null);
    try {
      await resetPassword({ username: values.email });
    } catch (e) {
      const name = (e as { name?: string }).name ?? "";
      if (name === "LimitExceededException") {
        setSubmitError(
          "送信回数の上限に達しました。しばらくしてからお試しください",
        );
        setIsLocked(true);
        setLockUntil(msFromNow(10 * 1000));
        return;
      }
      if (name === "TooManyRequestsException") {
        setSubmitError("しばらく時間をおいて再度お試しください");
        setIsLocked(true);
        setLockUntil(msFromNow(10 * 1000));
        return;
      }
      if (name === "NotAuthorizedException") {
        setSubmitError(
          "このアカウントではパスワードリセットができません。サポートへお問い合わせください",
        );
        return;
      }
      if (name === "CodeDeliveryFailureException") {
        setSubmitError(
          "コードの送信に失敗しました。しばらくしてからお試しください",
        );
        return;
      }
      if (name === "InvalidParameterException") {
        requestForm.setError("email", {
          message: "メールアドレスの形式が正しくありません",
        });
        return;
      }
      if (name !== "UserNotFoundException") {
        setSubmitError("送信に失敗しました。通信状況をご確認ください");
        return;
      }
    }
    enterConfirmStep(values.email);
  }

  async function onConfirmSubmit(values: ConfirmValues) {
    setSubmitError(null);
    const code = codeValues.join("");
    try {
      await confirmResetPassword({
        username: email,
        newPassword: values.newPassword,
        confirmationCode: code,
      });
      setStep("done");
    } catch (e) {
      const name = (e as { name?: string }).name ?? "";
      if (name === "CodeMismatchException") {
        const next = remainingAttempts - 1;
        setRemainingAttempts(next);
        if (next <= 0) {
          setIsLocked(true);
          setLockUntil(msFromNow(15 * 60 * 1000));
          setSubmitError(
            "試行回数が上限に達しました。しばらくしてからお試しください",
          );
        } else {
          setSubmitError(`コードが正しくありません（残り ${next} 回）`);
          setCodeValues(["", "", "", "", "", ""]);
          setCodeResetCount((c) => c + 1);
        }
        return;
      }
      if (name === "ExpiredCodeException") {
        setSubmitError("コードの有効期限が切れました。再送してください");
        setCodeExpired(true);
        return;
      }
      if (name === "InvalidPasswordException") {
        confirmForm.setError("newPassword", {
          message: "パスワードがポリシーを満たしていません",
        });
        return;
      }
      if (
        name === "LimitExceededException" ||
        name === "TooManyFailedAttemptsException"
      ) {
        setIsLocked(true);
        setLockUntil(msFromNow(15 * 60 * 1000));
        setSubmitError(
          "試行回数が上限に達しました。しばらくしてからお試しください",
        );
        return;
      }
      if (name === "UserNotFoundException") {
        setSubmitError("ユーザーが見つかりません。最初からやり直してください");
        return;
      }
      setSubmitError("リセットに失敗しました。通信状況をご確認ください");
    }
  }

  async function handleResend() {
    if (resendSecondsLeft > 0 || resendLoading || isLocked) return;
    setResendLoading(true);
    setResendToast(null);
    setSubmitError(null);
    try {
      await resetPassword({ username: email });
      setResendToast({ type: "success", message: "コードを再送しました" });
      setResendSecondsLeft(60);
      setSoftExpireLeft(15 * 60);
      setCodeExpired(false);
    } catch (e) {
      const name = (e as { name?: string }).name ?? "";
      if (name === "LimitExceededException") {
        setSubmitError("しばらく時間をおいて再度お試しください");
        setIsLocked(true);
        setLockUntil(msFromNow(15 * 60 * 1000));
      } else {
        setResendToast({ type: "error", message: "再送に失敗しました" });
      }
    } finally {
      setResendLoading(false);
    }
  }

  function handleCodeChange(index: number, value: string) {
    const digit = value.replace(/\D/g, "").slice(-1);
    const next = [...codeValues];
    next[index] = digit;
    setCodeValues(next);
    if (digit && index < 5) {
      codeRefs.current[index + 1]?.focus();
    }
  }

  function handleCodeKeyDown(
    index: number,
    e: React.KeyboardEvent<HTMLInputElement>,
  ) {
    if (e.key === "Backspace" && codeValues[index] === "" && index > 0) {
      const next = [...codeValues];
      next[index - 1] = "";
      setCodeValues(next);
      codeRefs.current[index - 1]?.focus();
    }
  }

  function handleCodePaste(e: React.ClipboardEvent<HTMLInputElement>) {
    e.preventDefault();
    const text = e.clipboardData.getData("text").replace(/\D/g, "").slice(0, 6);
    if (!text) return;
    const next = Array(6)
      .fill("")
      .map((_, i) => text[i] ?? "");
    setCodeValues(next);
    codeRefs.current[Math.min(text.length, 5)]?.focus();
  }

  const codeComplete = codeValues.every((v) => v !== "");
  const allPasswordChecks =
    newPassword.length >= 8 &&
    /[A-Z]/.test(newPassword) &&
    /[a-z]/.test(newPassword) &&
    /[0-9]/.test(newPassword) &&
    /[^a-zA-Z0-9]/.test(newPassword);
  const isStep2Valid =
    codeComplete &&
    allPasswordChecks &&
    confirmPassword === newPassword &&
    !isLocked &&
    !codeExpired;

  return (
    <div className="space-y-6">
      {/* Header */}
      <header className="flex h-14 items-center gap-3">
        <h1 className="text-xl font-semibold text-foreground">
          パスワードをリセット
        </h1>
      </header>

      {/* Step 1: Email */}
      {step === "request" && (
        <div className="space-y-6">
          <div>
            <p className="mt-1 text-sm text-muted-foreground">
              ご登録のメールアドレスを入力してください。
              <br />
              確認コードをお送りします。
            </p>
          </div>

          <form
            onSubmit={requestForm.handleSubmit(onRequestSubmit)}
            className="space-y-4"
            noValidate
          >
            <div className="space-y-1.5">
              <Label htmlFor="email">メールアドレス</Label>
              <Input
                id="email"
                type="email"
                autoComplete="email"
                inputMode="email"
                aria-invalid={!!requestForm.formState.errors.email}
                {...requestForm.register("email")}
              />
              {requestForm.formState.errors.email && (
                <p className="text-xs text-destructive">
                  {requestForm.formState.errors.email.message}
                </p>
              )}
            </div>

            <div className="rounded-md bg-muted px-3 py-3 text-xs text-muted-foreground">
              Googleアカウントでログインされている方は、Googleからパスワードを変更してください。
            </div>

            {submitError && (
              <p className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {submitError}
              </p>
            )}

            <Button
              type="submit"
              size="lg"
              className="w-full"
              disabled={
                requestForm.formState.isSubmitting || isLocked || !requestEmail
              }
            >
              {requestForm.formState.isSubmitting
                ? "送信中..."
                : "コードを送信"}
            </Button>
          </form>

          <p className="text-center text-sm">
            <Link
              href="/login"
              className="text-primary underline-offset-4 hover:underline"
            >
              ログイン画面に戻る
            </Link>
          </p>
        </div>
      )}

      {/* Step 2: Code + Password */}
      {step === "confirm" && (
        <div className="space-y-6">
          <div>
            <p className="text-2xl font-bold text-foreground">
              新しいパスワードを設定
            </p>
            <div className="mt-1 flex items-baseline gap-2 text-sm text-muted-foreground">
              <span>
                下記のメールアドレスに6桁のコードを送信しました。
                <br />
                <span className="font-medium text-foreground">{email}</span>
              </span>
              <button
                type="button"
                className="shrink-0 text-xs text-primary underline-offset-4 hover:underline"
                onClick={() => setShowChangeEmailDialog(true)}
              >
                変更
              </button>
            </div>
          </div>

          <form
            onSubmit={confirmForm.handleSubmit(onConfirmSubmit)}
            className="space-y-6"
            noValidate
          >
            {/* 6-digit code input */}
            <div className="space-y-2">
              <Label>確認コード</Label>
              <div className="flex gap-2">
                {codeValues.map((val, i) => (
                  <input
                    key={i}
                    ref={(el) => {
                      codeRefs.current[i] = el;
                    }}
                    type="text"
                    inputMode="numeric"
                    autoComplete={i === 0 ? "one-time-code" : "off"}
                    pattern="[0-9]*"
                    maxLength={1}
                    value={val}
                    disabled={isLocked || codeExpired}
                    onChange={(e) => handleCodeChange(i, e.target.value)}
                    onKeyDown={(e) => handleCodeKeyDown(i, e)}
                    onPaste={handleCodePaste}
                    className="h-14 w-10 rounded-md border border-border bg-background text-center text-2xl font-semibold focus:border-2 focus:border-primary focus:outline-none disabled:opacity-50"
                    aria-label={`確認コード ${i + 1}桁目`}
                  />
                ))}
              </div>

              {/* Resend */}
              <div className="space-y-1">
                <button
                  type="button"
                  className="text-sm disabled:opacity-50"
                  onClick={handleResend}
                  disabled={resendSecondsLeft > 0 || resendLoading || isLocked}
                >
                  {resendLoading ? (
                    <span className="text-muted-foreground">送信中...</span>
                  ) : resendSecondsLeft > 0 ? (
                    <span className="text-muted-foreground">
                      コードを再送（あと{resendSecondsLeft}秒）
                    </span>
                  ) : (
                    <span
                      className={
                        codeExpired
                          ? "font-medium text-primary"
                          : "text-primary"
                      }
                    >
                      コードを再送
                    </span>
                  )}
                </button>
                <p className="text-xs text-muted-foreground">
                  メールが届かない場合は、迷惑メールフォルダもご確認ください。
                </p>
              </div>

              {resendToast && (
                <p
                  className={`rounded-md px-3 py-2 text-sm ${
                    resendToast.type === "success"
                      ? "bg-emerald-50 text-emerald-700"
                      : "bg-destructive/10 text-destructive"
                  }`}
                >
                  {resendToast.message}
                </p>
              )}
            </div>

            {/* New password */}
            <div className="space-y-1.5">
              <Label htmlFor="newPassword">新しいパスワード</Label>
              <div className="relative">
                <Input
                  id="newPassword"
                  type={passwordVisible ? "text" : "password"}
                  autoComplete="new-password"
                  aria-invalid={!!confirmForm.formState.errors.newPassword}
                  {...confirmForm.register("newPassword")}
                />
                <button
                  type="button"
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground"
                  onClick={() => setPasswordVisible((v) => !v)}
                  tabIndex={-1}
                  aria-label={
                    passwordVisible ? "パスワードを隠す" : "パスワードを表示"
                  }
                >
                  {passwordVisible ? (
                    <EyeOffIcon className="h-5 w-5" />
                  ) : (
                    <EyeIcon className="h-5 w-5" />
                  )}
                </button>
              </div>
              {confirmForm.formState.errors.newPassword && (
                <p className="text-xs text-destructive">
                  {confirmForm.formState.errors.newPassword.message}
                </p>
              )}
              {newPassword.length > 0 && (
                <PasswordChecklist password={newPassword} />
              )}
            </div>

            {/* Confirm password */}
            <div className="space-y-1.5">
              <Label htmlFor="confirmPassword">
                新しいパスワード（確認用）
              </Label>
              <div className="relative">
                <Input
                  id="confirmPassword"
                  type={confirmPasswordVisible ? "text" : "password"}
                  autoComplete="new-password"
                  aria-invalid={!!confirmForm.formState.errors.confirmPassword}
                  {...confirmForm.register("confirmPassword")}
                />
                <button
                  type="button"
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground"
                  onClick={() => setConfirmPasswordVisible((v) => !v)}
                  tabIndex={-1}
                  aria-label={
                    confirmPasswordVisible
                      ? "パスワードを隠す"
                      : "パスワードを表示"
                  }
                >
                  {confirmPasswordVisible ? (
                    <EyeOffIcon className="h-5 w-5" />
                  ) : (
                    <EyeIcon className="h-5 w-5" />
                  )}
                </button>
              </div>
              {confirmForm.formState.errors.confirmPassword && (
                <p className="text-xs text-destructive">
                  {confirmForm.formState.errors.confirmPassword.message}
                </p>
              )}
            </div>

            {submitError && (
              <p className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {submitError}
              </p>
            )}

            <Button
              type="submit"
              size="lg"
              className="w-full"
              disabled={confirmForm.formState.isSubmitting || !isStep2Valid}
            >
              {confirmForm.formState.isSubmitting
                ? "リセット中..."
                : "パスワードをリセット"}
            </Button>
          </form>
        </div>
      )}

      {/* Step 3: Done */}
      {step === "done" && (
        <div className="space-y-6 pt-8 text-center">
          <div className="flex flex-col items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-emerald-100">
              <svg
                className="h-6 w-6 text-emerald-600"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2.5}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M5 13l4 4L19 7"
                />
              </svg>
            </div>
            <div>
              <p className="text-2xl font-bold text-foreground">
                パスワードをリセットしました
              </p>
              <p className="mt-1 text-sm text-muted-foreground">
                新しいパスワードでログインしてください。
              </p>
            </div>
          </div>

          <Button
            size="lg"
            className="w-full"
            onClick={() => {
              sessionStorage.setItem("passwordResetEmail", email);
              router.push("/login");
            }}
          >
            ログインへ
          </Button>
        </div>
      )}

      {/* Abort dialog (step 2 back) */}
      <Dialog open={showAbortDialog} onOpenChange={setShowAbortDialog}>
        <DialogContent showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>パスワードリセットを中断しますか？</DialogTitle>
            <DialogDescription>
              中断すると、送信されたコードは無効になります
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAbortDialog(false)}>
              続ける
            </Button>
            <Button
              variant="destructive"
              onClick={() => {
                setShowAbortDialog(false);
                router.push("/login");
              }}
            >
              中断する
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Change email dialog */}
      <Dialog
        open={showChangeEmailDialog}
        onOpenChange={setShowChangeEmailDialog}
      >
        <DialogContent showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>メールアドレスを変更しますか？</DialogTitle>
            <DialogDescription>
              これまでに送信されたコードは無効になります
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowChangeEmailDialog(false)}
            >
              キャンセル
            </Button>
            <Button
              onClick={() => {
                setShowChangeEmailDialog(false);
                requestForm.setValue("email", email);
                confirmForm.reset();
                setCodeValues(["", "", "", "", "", ""]);
                setSubmitError(null);
                setIsLocked(false);
                setLockUntil(null);
                setCodeExpired(false);
                setSoftExpireLeft(null);
                setStep("request");
              }}
            >
              変更する
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
