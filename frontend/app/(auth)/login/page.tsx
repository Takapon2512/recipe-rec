"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { signIn, signInWithRedirect } from "aws-amplify/auth";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";

const loginSchema = z.object({
  email: z
    .email("メールアドレスの形式が正しくありません")
    .min(1, "メールアドレスを入力してください"),
  password: z.string().min(1, "パスワードを入力してください"),
});

type LoginValues = z.infer<typeof loginSchema>;

export default function LoginPage() {
  const router = useRouter();
  const [submitError, setSubmitError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      email: "",
      password: "",
    },
  });

  async function onSubmit(values: LoginValues) {
    setSubmitError(null);
    try {
      const { isSignedIn, nextStep } = await signIn({
        username: values.email,
        password: values.password,
      });

      if (isSignedIn) {
        router.replace("/home");
        return;
      }

      // メール未確認のユーザーは確認画面へ
      if (nextStep?.signInStep === "CONFIRM_SIGN_UP") {
        router.push(`/signup/verify?email=${encodeURIComponent(values.email)}`);
        return;
      }

      setSubmitError(
        "ログインを完了できませんでした。しばらく経ってから再試行してください。",
      );
    } catch (e) {
      const name = (e as { name?: string }).name ?? "";
      if (
        name === "NotAuthorizedException" ||
        name === "UserNotFoundException"
      ) {
        setSubmitError("メールアドレスまたはパスワードが正しくありません。");
      } else if (name === "UserNotConfirmedException") {
        router.push(`/signup/verify?email=${encodeURIComponent(values.email)}`);
      } else if (name === "UserAlreadyAuthenticatedException") {
        router.replace("/home");
      } else {
        setSubmitError(
          "ログインに失敗しました。しばらく経ってから再試行してください。",
        );
      }
    }
  }

  async function onGoogleLogin() {
    setSubmitError(null);
    try {
      await signInWithRedirect({ provider: "Google" });
    } catch (e) {
      const name = (e as { name?: string }).name ?? "";
      if (name === "UserAlreadyAuthenticatedException") {
        router.replace("/home");
        return;
      }
      setSubmitError(
        "Googleでのログインに失敗しました。しばらく経ってから再試行してください。",
      );
    }
  }

  return (
    <div className="space-y-6">
      <header className="flex h-14 items-center">
        <h1 className="text-xl font-semibold text-foreground">ログイン</h1>
      </header>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
        <div className="space-y-1.5">
          <Label htmlFor="email">メールアドレス</Label>
          <Input
            id="email"
            type="email"
            autoComplete="email"
            inputMode="email"
            aria-invalid={!!errors.email}
            {...register("email")}
          />
          {errors.email && (
            <p className="text-xs text-destructive">{errors.email.message}</p>
          )}
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="password">パスワード</Label>
          <Input
            id="password"
            type="password"
            autoComplete="current-password"
            aria-invalid={!!errors.password}
            {...register("password")}
          />
          {errors.password && (
            <p className="text-xs text-destructive">
              {errors.password.message}
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
          disabled={isSubmitting}
        >
          {isSubmitting ? "ログイン中..." : "ログイン"}
        </Button>
      </form>

      <div className="relative">
        <Separator />
        <span className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 bg-background px-2 text-xs text-muted-foreground">
          または
        </span>
      </div>

      <Button
        type="button"
        variant="outline"
        size="lg"
        className="w-full"
        disabled={isSubmitting}
        onClick={onGoogleLogin}
      >
        Googleでログイン
      </Button>

      <p className="text-center text-sm text-muted-foreground">
        アカウントをお持ちでない方は{" "}
        <Link
          href="/signup"
          className="text-primary underline-offset-4 hover:underline"
        >
          新規登録
        </Link>
      </p>
    </div>
  );
}
