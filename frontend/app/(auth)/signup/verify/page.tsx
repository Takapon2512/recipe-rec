"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { confirmSignUp, resendSignUpCode } from "aws-amplify/auth";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const verifySchema = z.object({
  code: z
    .string()
    .length(6, "6桁のコードを入力してください")
    .regex(/^\d+$/, "数字のみ入力してください"),
});

type VerifyValues = z.infer<typeof verifySchema>;

export default function VerifyPage() {
  return (
    <Suspense fallback={<VerifyFallback />}>
      <VerifyForm />
    </Suspense>
  );
}

function VerifyFallback() {
  return (
    <div className="space-y-6">
      <header className="flex h-14 items-center">
        <h1 className="text-xl font-semibold text-foreground">メール確認</h1>
      </header>
      <p className="text-sm text-muted-foreground">読み込み中...</p>
    </div>
  );
}

function VerifyForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const email = searchParams.get("email") ?? "";

  const [submitError, setSubmitError] = useState<string | null>(null);
  const [resendMessage, setResendMessage] = useState<string | null>(null);
  const [resending, setResending] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<VerifyValues>({
    resolver: zodResolver(verifySchema),
    defaultValues: { code: "" },
  });

  if (!email) {
    router.replace("/signup");
    return null;
  }

  async function onSubmit(values: VerifyValues) {
    setSubmitError(null);
    try {
      await confirmSignUp({ username: email, confirmationCode: values.code });
      router.push("/login");
    } catch (e) {
      const message = e instanceof Error ? e.message : "確認に失敗しました";
      setSubmitError(message);
    }
  }

  async function handleResend() {
    setResendMessage(null);
    setSubmitError(null);
    setResending(true);
    try {
      await resendSignUpCode({ username: email });
      setResendMessage("確認コードを再送しました");
    } catch (e) {
      const message = e instanceof Error ? e.message : "再送に失敗しました";
      setSubmitError(message);
    } finally {
      setResending(false);
    }
  }

  return (
    <div className="space-y-6">
      <header className="flex h-14 items-center">
        <h1 className="text-xl font-semibold text-foreground">メール確認</h1>
      </header>

      <p className="text-sm text-muted-foreground">
        <span className="font-medium text-foreground">{email}</span>{" "}
        に確認コードを送信しました。6桁のコードを入力してください。
      </p>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
        <div className="space-y-1.5">
          <Label htmlFor="code">確認コード</Label>
          <Input
            id="code"
            type="text"
            inputMode="numeric"
            autoComplete="one-time-code"
            maxLength={6}
            placeholder="000000"
            aria-invalid={!!errors.code}
            {...register("code")}
          />
          {errors.code && (
            <p className="text-xs text-destructive">{errors.code.message}</p>
          )}
        </div>

        {submitError && (
          <p className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
            {submitError}
          </p>
        )}
        {resendMessage && (
          <p className="rounded-md bg-green-50 px-3 py-2 text-sm text-green-700">
            {resendMessage}
          </p>
        )}

        <Button
          type="submit"
          size="lg"
          className="w-full"
          disabled={isSubmitting}
        >
          {isSubmitting ? "確認中..." : "確認"}
        </Button>
      </form>

      <p className="text-center text-sm text-muted-foreground">
        コードが届かない場合は{" "}
        <button
          type="button"
          className="text-primary underline-offset-4 hover:underline disabled:opacity-50"
          onClick={handleResend}
          disabled={resending}
        >
          {resending ? "送信中..." : "再送する"}
        </button>
      </p>

      <p className="text-center text-sm text-muted-foreground">
        <Link
          href="/signup"
          className="text-primary underline-offset-4 hover:underline"
        >
          登録画面に戻る
        </Link>
      </p>
    </div>
  );
}
