"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { getCurrentUser } from "aws-amplify/auth";

export default function AuthCallbackPage() {
  const router = useRouter();

  useEffect(() => {
    // Amplify のコード交換完了を待ってポーリング（Hub は timing 次第で取り逃がすため）
    const MAX_ATTEMPTS = 20;
    let attempts = 0;

    const interval = setInterval(async () => {
      try {
        await getCurrentUser();
        clearInterval(interval);
        router.replace("/home");
      } catch {
        attempts++;
        if (attempts >= MAX_ATTEMPTS) {
          clearInterval(interval);
          router.replace("/signup?error=google_failed");
        }
      }
    }, 500);

    return () => clearInterval(interval);
  }, [router]);

  return (
    <div className="flex min-h-screen items-center justify-center">
      <p className="text-sm text-muted-foreground">認証処理中...</p>
    </div>
  );
}
