import type { ReactNode } from "react"

export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <main className="flex flex-1 justify-center bg-background px-4 pt-8 pb-[max(env(safe-area-inset-bottom),2rem)] sm:pt-16">
      <div className="w-full max-w-[480px]">{children}</div>
    </main>
  )
}
