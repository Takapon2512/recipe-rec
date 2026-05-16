export default function HomePage() {
  return (
    <main className="flex flex-1 flex-col gap-2 p-8">
      <h1 className="text-2xl font-semibold text-foreground">ホーム</h1>
      <p className="text-sm text-muted-foreground">
        ログイン中のユーザーのみが表示できる画面です。
      </p>
    </main>
  );
}
