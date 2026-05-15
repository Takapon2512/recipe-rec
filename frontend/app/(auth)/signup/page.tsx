"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { signUp, signInWithRedirect } from "aws-amplify/auth";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";

const signupSchema = z
  .object({
    email: z
      .email("メールアドレスの形式が正しくありません")
      .min(1, "メールアドレスを入力してください"),
    password: z
      .string()
      .min(8, "8文字以上で入力してください")
      .regex(/[a-z]/, "英小文字を含めてください")
      .regex(/[A-Z]/, "英大文字を含めてください")
      .regex(/[0-9]/, "数字を含めてください")
      .regex(/[^a-zA-Z0-9]/, "記号を含めてください"),
    confirmPassword: z.string().min(1, "確認のため再入力してください"),
    terms: z.boolean().refine((v) => v === true, {
      message: "利用規約への同意が必要です",
    }),
  })
  .refine((v) => v.password === v.confirmPassword, {
    path: ["confirmPassword"],
    message: "パスワードが一致しません",
  });

type SignupValues = z.infer<typeof signupSchema>;

export default function SignupPage() {
  const router = useRouter();
  const [submitError, setSubmitError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    setValue,
    control,
    formState: { errors, isSubmitting },
  } = useForm<SignupValues>({
    resolver: zodResolver(signupSchema),
    defaultValues: {
      email: "",
      password: "",
      confirmPassword: "",
      terms: false,
    },
  });

  const terms = useWatch({ control, name: "terms" });

  async function onSubmit(values: SignupValues) {
    setSubmitError(null);
    try {
      await signUp({
        username: values.email,
        password: values.password,
        options: {
          userAttributes: { email: values.email },
        },
      });
      router.push(`/signup/verify?email=${encodeURIComponent(values.email)}`);
    } catch (e) {
      // エラー内容を適切な文言に振り分け
      const name = (e as { name?: string }).name ?? "";
      if (name === "UsernameExistsException") {
        setSubmitError(
          "このメールアドレスはすでに登録済みか、登録手続き中です。",
        );
      } else if (name === "InvalidPasswordException") {
        setSubmitError("パスワードの要件を満たしていません。");
      } else {
        setSubmitError(
          "登録に失敗しました。しばらく経ってから再試行してください。",
        );
      }
    }
  }

  async function onGoogleSignup() {
    setSubmitError(null);
    try {
      await signInWithRedirect({ provider: "Google" });
    } catch (e) {
      const name = (e as { name?: string }).name ?? "";
      if (name === "UserAlreadyAuthenticatedException") {
        router.push("/home");
        return;
      }
      setSubmitError(
        "Googleでの登録に失敗しました。しばらく経ってから再試行してください。",
      );
    }
  }

  return (
    <div className="space-y-6">
      <header className="flex h-14 items-center">
        <h1 className="text-xl font-semibold text-foreground">新規登録</h1>
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
            autoComplete="new-password"
            aria-invalid={!!errors.password}
            {...register("password")}
          />
          {errors.password ? (
            <p className="text-xs text-destructive">
              {errors.password.message}
            </p>
          ) : (
            <p className="text-xs text-muted-foreground">
              8文字以上・英大小・数字・記号を含めてください
            </p>
          )}
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="confirmPassword">パスワード (確認)</Label>
          <Input
            id="confirmPassword"
            type="password"
            autoComplete="new-password"
            aria-invalid={!!errors.confirmPassword}
            {...register("confirmPassword")}
          />
          {errors.confirmPassword && (
            <p className="text-xs text-destructive">
              {errors.confirmPassword.message}
            </p>
          )}
        </div>

        <div className="flex items-start gap-2 pt-2">
          <Checkbox
            id="terms"
            checked={terms}
            onCheckedChange={(checked) =>
              setValue("terms", checked === true, { shouldValidate: true })
            }
            aria-invalid={!!errors.terms}
          />
          <div className="space-y-1">
            <Label htmlFor="terms" className="text-sm font-normal leading-snug">
              <Link
                href="/terms"
                className="text-primary underline-offset-4 hover:underline"
              >
                利用規約
              </Link>
              および
              <Link
                href="/privacy"
                className="text-primary underline-offset-4 hover:underline"
              >
                プライバシーポリシー
              </Link>
              に同意します
            </Label>
            {errors.terms && (
              <p className="text-xs text-destructive">{errors.terms.message}</p>
            )}
          </div>
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
          {isSubmitting ? "登録中..." : "登録"}
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
        onClick={onGoogleSignup}
      >
        Googleで登録
      </Button>

      <p className="text-center text-sm text-muted-foreground">
        すでにアカウントをお持ちの方は{" "}
        <Link
          href="/login"
          className="text-primary underline-offset-4 hover:underline"
        >
          ログイン
        </Link>
      </p>
    </div>
  );
}
